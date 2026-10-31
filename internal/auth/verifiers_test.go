package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type testSigningKey struct {
	kid string
	key *rsa.PrivateKey
}

var (
	testKeysOnce     sync.Once
	primaryTestKey   *testSigningKey
	secondaryTestKey *testSigningKey
)

func primaryKey(t *testing.T) *testSigningKey {
	t.Helper()
	testKeysOnce.Do(func() {
		primaryTestKey = generateTestKey(t, "primary-kid")
		secondaryTestKey = generateTestKey(t, "secondary-kid")
	})
	return primaryTestKey
}

func secondaryKey(t *testing.T) *testSigningKey {
	t.Helper()
	primaryKey(t)
	return secondaryTestKey
}

func generateTestKey(t *testing.T, kid string) *testSigningKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return &testSigningKey{kid: kid, key: key}
}

type jwksTestServer struct {
	server *httptest.Server
	calls  atomic.Int64

	mu     sync.Mutex
	keys   []*testSigningKey
	status int
	body   string
}

func newJWKSTestServer(t *testing.T, keys ...*testSigningKey) *jwksTestServer {
	t.Helper()

	server := &jwksTestServer{keys: keys, status: http.StatusOK}
	server.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		server.calls.Add(1)

		server.mu.Lock()
		defer server.mu.Unlock()

		if server.body != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(server.status)
			_, _ = w.Write([]byte(server.body))
			return
		}
		if server.status != http.StatusOK {
			w.WriteHeader(server.status)
			return
		}

		document := jwksDocument{Keys: make([]jsonWebKey, 0, len(server.keys))}
		for _, key := range server.keys {
			document.Keys = append(document.Keys, jsonWebKey{
				Kty: "RSA",
				Kid: key.kid,
				Alg: jwt.SigningMethodRS256.Alg(),
				Use: "sig",
				N:   base64.RawURLEncoding.EncodeToString(key.key.PublicKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.key.PublicKey.E)).Bytes()),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(document)
	}))
	t.Cleanup(server.server.Close)

	return server
}

func (s *jwksTestServer) setKeys(keys ...*testSigningKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = keys
	s.status = http.StatusOK
	s.body = ""
}

func (s *jwksTestServer) setResponse(status int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.body = body
}

func (s *jwksTestServer) URL() string          { return s.server.URL }
func (s *jwksTestServer) Client() *http.Client { return s.server.Client() }
func (s *jwksTestServer) callCount() int64     { return s.calls.Load() }

func googleTestConfig(server *jwksTestServer, now func() time.Time) jwksVerifierConfig {
	return jwksVerifierConfig{
		provider:        ProviderGoogle,
		jwksURL:         server.URL(),
		issuers:         googleIssuers,
		audiences:       []string{"google-client-id"},
		nonceMode:       nonceExact,
		httpClient:      server.Client(),
		now:             now,
		refreshCooldown: 30 * time.Second,
		cacheTTL:        time.Hour,
	}
}

func appleTestConfig(server *jwksTestServer, now func() time.Time) jwksVerifierConfig {
	return jwksVerifierConfig{
		provider:        ProviderApple,
		jwksURL:         server.URL(),
		issuers:         appleIssuers,
		audiences:       []string{"com.example.automyself", "com.example.automyself.web"},
		nonceMode:       nonceSHA256Variants,
		httpClient:      server.Client(),
		now:             now,
		refreshCooldown: 30 * time.Second,
		cacheTTL:        time.Hour,
	}
}

func newTestVerifier(t *testing.T, cfg jwksVerifierConfig) *jwksVerifier {
	t.Helper()
	verifier, err := newJWKSVerifier(cfg)
	if err != nil {
		t.Fatalf("building verifier: %v", err)
	}
	return verifier
}

func testClaims(now time.Time, issuer, audience, subject, nonce string) identityTokenClaims {
	return identityTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
		Email: "user@example.com",
		Name:  "Test User",
		Nonce: nonce,
	}
}

func signTestToken(t *testing.T, key *testSigningKey, claims identityTokenClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = key.kid
	signed, err := token.SignedString(key.key)
	if err != nil {
		t.Fatalf("signing identity token: %v", err)
	}
	return signed
}

func TestNewVerifiersRejectsEmptyAudience(t *testing.T) {
	if _, err := NewVerifiers(nil, []string{"apple-client"}); err == nil {
		t.Fatal("expected an error for empty Google audiences")
	}
	if _, err := NewVerifiers([]string{"google-client"}, nil); err == nil {
		t.Fatal("expected an error for empty Apple audiences")
	}
}

func TestNewJWKSVerifierRejectsNonHTTPSURL(t *testing.T) {
	_, err := newJWKSVerifier(jwksVerifierConfig{
		provider:  ProviderGoogle,
		jwksURL:   "http://example.com/keys",
		issuers:   googleIssuers,
		audiences: []string{"client"},
		nonceMode: nonceExact,
	})
	if err == nil {
		t.Fatal("expected an error for a non-HTTPS JWKS URL")
	}
}

func TestJWKSVerifierValidToken(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))

	identity, err := verifier.Verify(context.Background(), token, "nonce-1")
	if err != nil {
		t.Fatalf("verifying token: %v", err)
	}
	if identity.Provider != ProviderGoogle {
		t.Errorf("expected provider google, got %q", identity.Provider)
	}
	if identity.Subject != "subject-1" {
		t.Errorf("expected subject-1, got %q", identity.Subject)
	}
	if identity.Email != "user@example.com" {
		t.Errorf("expected the email claim, got %q", identity.Email)
	}
	if identity.Name != "Test User" {
		t.Errorf("expected the name claim, got %q", identity.Name)
	}
}

func TestJWKSVerifierAcceptsAlternateGoogleIssuer(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := signTestToken(t, key, testClaims(now, "accounts.google.com", "google-client-id", "subject-1", "nonce-1"))

	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); err != nil {
		t.Fatalf("expected accounts.google.com to be accepted: %v", err)
	}
}

func TestJWKSVerifierRejectsExpiredToken(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	claims := testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1")
	claims.IssuedAt = jwt.NewNumericDate(now.Add(-2 * time.Hour))
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Hour))

	token := signTestToken(t, key, claims)
	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
	}
}

func TestJWKSVerifierRequiresExpiration(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	claims := testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1")
	claims.ExpiresAt = nil

	token := signTestToken(t, key, claims)
	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
	}
}

func TestJWKSVerifierRequiresIssuedAt(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	claims := testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1")
	claims.IssuedAt = nil

	token := signTestToken(t, key, claims)
	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
	}
}

func TestJWKSVerifierRejectsWrongIssuer(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := signTestToken(t, key, testClaims(now, "https://evil.example.com", "google-client-id", "subject-1", "nonce-1"))
	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
	}
}

func TestJWKSVerifierRejectsWrongAudience(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "someone-else", "subject-1", "nonce-1"))
	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
	}
}

func TestJWKSVerifierRejectsBadSignature(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	other := secondaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))
	token.Header["kid"] = key.kid
	signed, err := token.SignedString(other.key)
	if err != nil {
		t.Fatalf("signing token with the wrong key: %v", err)
	}

	if _, err := verifier.Verify(context.Background(), signed, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
	}
}

func TestJWKSVerifierRejectsUnsupportedAlgorithm(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))
	claims := testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1")

	hs256, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("signing HS256 token: %v", err)
	}
	if _, err := verifier.Verify(context.Background(), hs256, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected HS256 to be rejected, got %v", err)
	}

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing none token: %v", err)
	}
	if _, err := verifier.Verify(context.Background(), none, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected alg=none to be rejected, got %v", err)
	}
}

func TestJWKSVerifierRequiresSubject(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "", "nonce-1"))
	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
	}
}

func TestJWKSVerifierNonce(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	google := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))
	apple := newTestVerifier(t, appleTestConfig(server, func() time.Time { return now }))

	const rawNonce = "nonce-1"
	sum := sha256.Sum256([]byte(rawNonce))
	hexHash := hex.EncodeToString(sum[:])
	base64Hash := base64.RawURLEncoding.EncodeToString(sum[:])

	tests := []struct {
		name         string
		verifier     *jwksVerifier
		tokenNonce   string
		requestNonce string
		wantErr      bool
	}{
		{"google raw match", google, rawNonce, rawNonce, false},
		{"google mismatch", google, rawNonce, "nonce-2", true},
		{"google empty request nonce", google, rawNonce, "", true},
		{"google missing token nonce", google, "", rawNonce, true},
		{"apple raw match", apple, rawNonce, rawNonce, false},
		{"apple sha256 hex", apple, hexHash, rawNonce, false},
		{"apple sha256 base64url", apple, base64Hash, rawNonce, false},
		{"apple mismatch", apple, hexHash, "nonce-2", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issuer := "https://accounts.google.com"
			audience := "google-client-id"
			if test.verifier == apple {
				issuer = "https://appleid.apple.com"
				audience = "com.example.automyself"
			}

			token := signTestToken(t, key, testClaims(now, issuer, audience, "subject-1", test.tokenNonce))
			_, err := test.verifier.Verify(context.Background(), token, test.requestNonce)
			if test.wantErr && !errors.Is(err, ErrInvalidIdentityToken) {
				t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected a valid token, got %v", err)
			}
		})
	}
}

func TestJWKSVerifierCachesKeys(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	first := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))
	second := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-2", "nonce-2"))

	if _, err := verifier.Verify(context.Background(), first, "nonce-1"); err != nil {
		t.Fatalf("verifying first token: %v", err)
	}
	if _, err := verifier.Verify(context.Background(), second, "nonce-2"); err != nil {
		t.Fatalf("verifying second token: %v", err)
	}
	if calls := server.callCount(); calls != 1 {
		t.Errorf("expected one JWKS fetch, got %d", calls)
	}
}

func TestJWKSVerifierRefreshesOnUnknownKidAfterCooldown(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	other := secondaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	first := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))
	if _, err := verifier.Verify(context.Background(), first, "nonce-1"); err != nil {
		t.Fatalf("verifying first token: %v", err)
	}

	now = now.Add(31 * time.Second)
	server.setKeys(key, other)

	second := signTestToken(t, other, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-2", "nonce-2"))
	if _, err := verifier.Verify(context.Background(), second, "nonce-2"); err != nil {
		t.Fatalf("verifying token with a rotated key: %v", err)
	}
	if calls := server.callCount(); calls != 2 {
		t.Errorf("expected a refresh after the cooldown, got %d fetches", calls)
	}
}

func TestJWKSVerifierUnknownKidWithinCooldown(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	other := secondaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	first := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))
	if _, err := verifier.Verify(context.Background(), first, "nonce-1"); err != nil {
		t.Fatalf("verifying first token: %v", err)
	}

	second := signTestToken(t, other, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-2", "nonce-2"))
	if _, err := verifier.Verify(context.Background(), second, "nonce-2"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected the unknown kid to be rejected, got %v", err)
	}
	if calls := server.callCount(); calls != 1 {
		t.Errorf("expected no refresh within the cooldown, got %d fetches", calls)
	}
}

func TestJWKSVerifierUsesCachedKeyWhenRefreshFails(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	first := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))
	if _, err := verifier.Verify(context.Background(), first, "nonce-1"); err != nil {
		t.Fatalf("verifying first token: %v", err)
	}

	now = now.Add(2 * time.Hour)
	server.setResponse(http.StatusInternalServerError, "")

	second := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-2", "nonce-2"))
	if _, err := verifier.Verify(context.Background(), second, "nonce-2"); err != nil {
		t.Fatalf("expected the cached key to keep working: %v", err)
	}
}

func TestJWKSVerifierFetchFailure(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	server.setResponse(http.StatusInternalServerError, "")
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))
	_, err := verifier.Verify(context.Background(), token, "nonce-1")
	if !errors.Is(err, errJWKSUnavailable) {
		t.Fatalf("expected errJWKSUnavailable, got %v", err)
	}
	if errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected a server-side error, got an invalid-token error")
	}
}

func TestJWKSVerifierRejectsMalformedJWKS(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	server.setResponse(http.StatusOK, "{invalid")
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))
	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); !errors.Is(err, errJWKSUnavailable) {
		t.Fatalf("expected errJWKSUnavailable, got %v", err)
	}
}

func TestJWKSVerifierRejectsUnusableKeys(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	server.setResponse(http.StatusOK, `{"keys":[{"kty":"EC","kid":"unsupported"}]}`)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	token := signTestToken(t, key, testClaims(now, "https://accounts.google.com", "google-client-id", "subject-1", "nonce-1"))
	if _, err := verifier.Verify(context.Background(), token, "nonce-1"); !errors.Is(err, errJWKSUnavailable) {
		t.Fatalf("expected errJWKSUnavailable, got %v", err)
	}
}

func TestJWKSVerifierRejectsMalformedToken(t *testing.T) {
	now := time.Now()
	key := primaryKey(t)
	server := newJWKSTestServer(t, key)
	verifier := newTestVerifier(t, googleTestConfig(server, func() time.Time { return now }))

	if _, err := verifier.Verify(context.Background(), "not-a-token", "nonce-1"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatalf("expected ErrInvalidIdentityToken, got %v", err)
	}
}
