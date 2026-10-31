package auth

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	ProviderGoogle = "google"
	ProviderApple  = "apple"

	googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"
	appleJWKSURL  = "https://appleid.apple.com/auth/keys"

	identityTokenLeeway = 30 * time.Second
	jwksHTTPTimeout     = 3 * time.Second
	jwksMaxBodyBytes    = 1 << 20
	jwksDefaultTTL      = time.Hour
	jwksRefreshCooldown = 30 * time.Second
	jwksMinModulusBits  = 2048
)

var (
	// ErrInvalidIdentityToken means the provider token itself was rejected.
	ErrInvalidIdentityToken = errors.New("invalid identity token")

	// errJWKSUnavailable means the provider's signing keys could not be obtained, so the
	// token could not be checked. It is a server-side failure, not a bad token.
	errJWKSUnavailable   = errors.New("identity provider signing keys unavailable")
	errUnknownSigningKey = errors.New("unknown identity provider signing key")

	googleIssuers = []string{"https://accounts.google.com", "accounts.google.com"}
	appleIssuers  = []string{"https://appleid.apple.com"}
)

// VerifiedIdentity is the provider identity extracted from a verified ID token.
type VerifiedIdentity struct {
	Provider string
	Subject  string
	Email    string
	Name     string
}

type IdentityTokenVerifier interface {
	Verify(ctx context.Context, rawIDToken, nonce string) (*VerifiedIdentity, error)
}

// IdentityVerifiers holds one verifier per supported mobile provider.
type IdentityVerifiers struct {
	Google IdentityTokenVerifier
	Apple  IdentityTokenVerifier
}

// NewVerifiers builds the production Google and Apple verifiers. No network I/O happens
// here; the provider JWKS is fetched lazily on the first login.
func NewVerifiers(googleAudiences, appleAudiences []string) (IdentityVerifiers, error) {
	if len(googleAudiences) == 0 {
		return IdentityVerifiers{}, errors.New("google identity audiences are empty")
	}
	if len(appleAudiences) == 0 {
		return IdentityVerifiers{}, errors.New("apple identity audiences are empty")
	}

	google, err := newJWKSVerifier(jwksVerifierConfig{
		provider:  ProviderGoogle,
		jwksURL:   googleJWKSURL,
		issuers:   googleIssuers,
		audiences: googleAudiences,
		nonceMode: nonceExact,
	})
	if err != nil {
		return IdentityVerifiers{}, fmt.Errorf("building google verifier: %w", err)
	}

	apple, err := newJWKSVerifier(jwksVerifierConfig{
		provider:  ProviderApple,
		jwksURL:   appleJWKSURL,
		issuers:   appleIssuers,
		audiences: appleAudiences,
		nonceMode: nonceSHA256Variants,
	})
	if err != nil {
		return IdentityVerifiers{}, fmt.Errorf("building apple verifier: %w", err)
	}

	return IdentityVerifiers{Google: google, Apple: apple}, nil
}

type nonceMode int

const (
	nonceExact nonceMode = iota
	nonceSHA256Variants
)

type jwksVerifierConfig struct {
	provider        string
	jwksURL         string
	issuers         []string
	audiences       []string
	nonceMode       nonceMode
	httpClient      *http.Client
	now             func() time.Time
	refreshCooldown time.Duration
	cacheTTL        time.Duration
}

type jwksVerifier struct {
	provider        string
	jwksURL         string
	issuers         []string
	audiences       []string
	nonceMode       nonceMode
	client          *http.Client
	now             func() time.Time
	refreshCooldown time.Duration
	cacheTTL        time.Duration

	refreshMu sync.Mutex
	snapshot  atomic.Pointer[jwksSnapshot]
}

// jwksSnapshot is immutable after it is published.
type jwksSnapshot struct {
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	expiresAt time.Time
	lastErr   error
}

func newJWKSVerifier(cfg jwksVerifierConfig) (*jwksVerifier, error) {
	if cfg.provider == "" {
		return nil, errors.New("identity provider is empty")
	}
	if len(cfg.issuers) == 0 {
		return nil, errors.New("identity token issuers are empty")
	}
	if len(cfg.audiences) == 0 {
		return nil, errors.New("identity token audiences are empty")
	}

	parsedURL, err := url.Parse(cfg.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("parsing JWKS URL: %w", err)
	}
	if parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return nil, errors.New("JWKS URL must be an https URL")
	}

	client := cfg.httpClient
	if client == nil {
		client = &http.Client{Timeout: jwksHTTPTimeout}
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	now := cfg.now
	if now == nil {
		now = time.Now
	}
	cooldown := cfg.refreshCooldown
	if cooldown <= 0 {
		cooldown = jwksRefreshCooldown
	}
	cacheTTL := cfg.cacheTTL
	if cacheTTL <= 0 {
		cacheTTL = jwksDefaultTTL
	}

	return &jwksVerifier{
		provider:        cfg.provider,
		jwksURL:         cfg.jwksURL,
		issuers:         cfg.issuers,
		audiences:       cfg.audiences,
		nonceMode:       cfg.nonceMode,
		client:          &clientCopy,
		now:             now,
		refreshCooldown: cooldown,
		cacheTTL:        cacheTTL,
	}, nil
}

type identityTokenClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Name  string `json:"name"`
	Nonce string `json:"nonce"`
}

func (v *jwksVerifier) Verify(ctx context.Context, rawIDToken, nonce string) (*VerifiedIdentity, error) {
	if rawIDToken == "" || nonce == "" {
		return nil, ErrInvalidIdentityToken
	}

	claims := &identityTokenClaims{}
	var keyErr error

	token, err := jwt.ParseWithClaims(
		rawIDToken,
		claims,
		func(token *jwt.Token) (any, error) {
			kid, ok := token.Header["kid"].(string)
			if !ok || kid == "" {
				keyErr = ErrInvalidIdentityToken
				return nil, ErrInvalidIdentityToken
			}

			key, lookupErr := v.keyForKID(ctx, kid)
			if lookupErr != nil {
				keyErr = lookupErr
				return nil, lookupErr
			}
			return key, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithAudience(v.audiences...),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(identityTokenLeeway),
		jwt.WithTimeFunc(v.now),
	)
	if keyErr != nil {
		if errors.Is(keyErr, ErrInvalidIdentityToken) || errors.Is(keyErr, errUnknownSigningKey) {
			return nil, ErrInvalidIdentityToken
		}
		return nil, keyErr
	}
	if err != nil || !token.Valid {
		return nil, ErrInvalidIdentityToken
	}

	if !slices.Contains(v.issuers, claims.Issuer) {
		return nil, ErrInvalidIdentityToken
	}
	if claims.Subject == "" || claims.IssuedAt == nil {
		return nil, ErrInvalidIdentityToken
	}
	if !nonceMatches(claims.Nonce, nonce, v.nonceMode) {
		return nil, ErrInvalidIdentityToken
	}

	return &VerifiedIdentity{
		Provider: v.provider,
		Subject:  claims.Subject,
		Email:    claims.Email,
		Name:     claims.Name,
	}, nil
}

func (v *jwksVerifier) keyForKID(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	now := v.now()
	if snapshot := v.snapshot.Load(); snapshot != nil {
		if key := snapshot.keys[kid]; key != nil && now.Before(snapshot.expiresAt) {
			return key, nil
		}
	}

	v.refreshMu.Lock()
	defer v.refreshMu.Unlock()

	now = v.now()
	if snapshot := v.snapshot.Load(); snapshot != nil {
		if key := snapshot.keys[kid]; key != nil && now.Before(snapshot.expiresAt) {
			return key, nil
		}
		if now.Sub(snapshot.fetchedAt) < v.refreshCooldown {
			if key := snapshot.keys[kid]; key != nil {
				return key, nil
			}
			if snapshot.lastErr != nil {
				return nil, fmt.Errorf("%w: %w", errJWKSUnavailable, snapshot.lastErr)
			}
			return nil, errUnknownSigningKey
		}
	}

	snapshot := v.refresh(ctx)
	v.snapshot.Store(snapshot)

	if key := snapshot.keys[kid]; key != nil {
		return key, nil
	}
	if snapshot.lastErr != nil {
		return nil, fmt.Errorf("%w: %w", errJWKSUnavailable, snapshot.lastErr)
	}
	return nil, errUnknownSigningKey
}

func (v *jwksVerifier) refresh(ctx context.Context) *jwksSnapshot {
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), jwksHTTPTimeout)
	defer cancel()

	keys, err := v.fetchJWKS(fetchCtx)
	now := v.now()
	if err != nil {
		snapshot := &jwksSnapshot{
			fetchedAt: now,
			lastErr:   err,
		}
		if previous := v.snapshot.Load(); previous != nil {
			snapshot.keys = previous.keys
			snapshot.expiresAt = previous.expiresAt
		}
		return snapshot
	}

	return &jwksSnapshot{
		keys:      keys,
		fetchedAt: now,
		expiresAt: now.Add(v.cacheTTL),
	}
}

type jwksDocument struct {
	Keys []jsonWebKey `json:"keys"`
}

type jsonWebKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (v *jwksVerifier) fetchJWKS(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating JWKS request: %w", err)
	}

	response, err := v.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetching JWKS: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned %s", response.Status)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, jwksMaxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading JWKS: %w", err)
	}
	if len(body) > jwksMaxBodyBytes {
		return nil, errors.New("JWKS response is too large")
	}

	var document jwksDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("decoding JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, key := range document.Keys {
		if key.Kid == "" || key.Kty != "RSA" {
			continue
		}
		if key.Alg != "" && key.Alg != jwt.SigningMethodRS256.Alg() {
			continue
		}
		if key.Use != "" && key.Use != "sig" {
			continue
		}

		publicKey, err := parseRSAPublicKey(key)
		if err != nil {
			continue
		}
		keys[key.Kid] = publicKey
	}
	if len(keys) == 0 {
		return nil, errors.New("JWKS contained no usable RSA signing keys")
	}

	return keys, nil
}

func parseRSAPublicKey(key jsonWebKey) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("decoding RSA modulus: %w", err)
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("decoding RSA exponent: %w", err)
	}

	modulus := new(big.Int).SetBytes(modulusBytes)
	if modulus.BitLen() < jwksMinModulusBits {
		return nil, errors.New("RSA modulus is too small")
	}
	if len(exponentBytes) == 0 || len(exponentBytes) > 4 {
		return nil, errors.New("RSA exponent is invalid")
	}

	exponent := new(big.Int).SetBytes(exponentBytes).Int64()
	if exponent < 3 || exponent%2 == 0 {
		return nil, errors.New("RSA exponent is invalid")
	}

	return &rsa.PublicKey{N: modulus, E: int(exponent)}, nil
}

func nonceMatches(tokenNonce, requestNonce string, mode nonceMode) bool {
	if tokenNonce == "" || requestNonce == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(tokenNonce), []byte(requestNonce)) == 1 {
		return true
	}
	if mode != nonceSHA256Variants {
		return false
	}

	sum := sha256.Sum256([]byte(requestNonce))
	hexHash := hex.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(tokenNonce), []byte(hexHash)) == 1 {
		return true
	}

	base64Hash := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(tokenNonce), []byte(base64Hash)) == 1
}
