package auth

import (
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/users"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

var (
	ErrUnsupportedProvider      = errors.New("unsupported provider")
	ErrUnableToHashRefreshToken = errors.New("unable to hash refresh token")
	ErrInvalidRefreshToken      = errors.New("invalid refresh token")
)

const (
	refreshTokenTTL        = 30 * 24 * time.Hour
	refreshTokenBcryptCost = 13
	jwtTTL                 = 15 * time.Minute
	jwtIssuer              = "auto-myself-auth"
	jwtAudience            = "auto-myself-api"
)

type AuthService interface {
	LoginWebInitiate(context.Context, string) (string, error)
	LoginWebCallback(context.Context, string, string) (*UserAuthResponse, error)
	RefreshToken(context.Context, RefreshTokenRequest) (*UserAuthResponse, error)
	DevelopmentLogin(context.Context, DevelopmentLoginRequest) (*UserAuthResponse, error)
}

type Service struct {
	authRepository  *Repository
	tokenService    *TokenService
	oauthRepository OAuthRepository
	secrets         *secrets.SecretStore
}

func NewAuthService(
	authRepository *Repository,
	tokenService *TokenService,
	oauthRepository OAuthRepository,
	secretStore *secrets.SecretStore,
) *Service {
	return &Service{
		authRepository:  authRepository,
		tokenService:    tokenService,
		oauthRepository: oauthRepository,
		secrets:         secretStore,
	}
}

// placeholderOAuthState keeps the provider round trip conventional. These web endpoints do not
// bind state to a browser session; they are a stepping stone to the mobile PKCE flow, which will
// add proper state/verifier binding.
const placeholderOAuthState = "state"

func (s *Service) LoginWebInitiate(ctx context.Context, provider string) (string, error) {
	if provider != "google" {
		return "", ErrUnsupportedProvider
	}

	conf, err := s.oauthConfig()
	if err != nil {
		return "", err
	}

	return s.oauthRepository.AuthCodeURL(conf, placeholderOAuthState), nil
}

func (s *Service) LoginWebCallback(ctx context.Context, provider string, code string) (*UserAuthResponse, error) {
	switch provider {
	case "google":
		return s.googleWebCallback(ctx, code)
	default:
		return nil, ErrUnsupportedProvider
	}
}

func (s *Service) RefreshToken(ctx context.Context, request RefreshTokenRequest) (*UserAuthResponse, error) {
	user, newRefreshToken, err := s.tokenService.RotateRefreshToken(ctx, request.RefreshToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("rotating refresh token: %w", err)
	}

	jwt, err := s.tokenService.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("generating jwt for user %s: %w", user.ID, err)
	}

	return &UserAuthResponse{JWT: jwt, Refresh: newRefreshToken}, nil
}

func (s *Service) DevelopmentLogin(ctx context.Context, request DevelopmentLoginRequest) (*UserAuthResponse, error) {
	userID, err := uuid.FromString(request.UserID)
	if err != nil {
		return nil, fmt.Errorf("parsing development user id: %w", err)
	}

	user, err := s.authRepository.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding development user %s: %w", userID, err)
	}

	return s.userToUserAuthResponse(ctx, user)
}

func (s *Service) oauthConfig() (*oauth2.Config, error) {
	clientID, err := s.secrets.Get("GOOGLE_OAUTH2_CLIENT_ID")
	if err != nil {
		return nil, fmt.Errorf("reading GOOGLE_OAUTH2_CLIENT_ID: %w", err)
	}
	clientSecret, err := s.secrets.Get("GOOGLE_OAUTH2_CLIENT_SECRET")
	if err != nil {
		return nil, fmt.Errorf("reading GOOGLE_OAUTH2_CLIENT_SECRET: %w", err)
	}
	domain, err := s.secrets.Get("DOMAIN")
	if err != nil {
		return nil, fmt.Errorf("reading DOMAIN: %w", err)
	}

	return &oauth2.Config{
		ClientID:     clientID.Value,
		ClientSecret: clientSecret.Value,
		RedirectURL:  strings.TrimSuffix(domain.Value, "/") + "/auth/web/callback/google",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}, nil
}

func (s *Service) googleWebCallback(ctx context.Context, code string) (*UserAuthResponse, error) {
	conf, err := s.oauthConfig()
	if err != nil {
		return nil, err
	}

	googleData, err := s.oauthRepository.ExchangeUserInfo(ctx, conf, code)
	if err != nil {
		return nil, err
	}

	user, err := s.authRepository.FindUserByProviderAndSubject(ctx, "google", googleData.Sub)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("finding user for google subject %s: %w", googleData.Sub, err)
		}

		user = &users.User{
			UserWeb: users.UserWeb{
				UserPost: users.UserPost{Username: googleData.Name},
			},
		}
		identity := &Identity{
			IdentityBase: IdentityBase{
				Provider: "google",
				Subject:  googleData.Sub,
				Email:    googleData.Email,
			},
		}

		if err := s.authRepository.CreateUserWithIdentity(ctx, user, identity); err != nil {
			return nil, fmt.Errorf("creating user for google subject %s: %w", googleData.Sub, err)
		}
	}

	return s.userToUserAuthResponse(ctx, user)
}

func (s *Service) userToUserAuthResponse(ctx context.Context, user *users.User) (*UserAuthResponse, error) {
	jwt, err := s.tokenService.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("generating jwt for user %s: %w", user.ID, err)
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token for user %s: %w", user.ID, err)
	}

	return &UserAuthResponse{
		JWT:     jwt,
		Refresh: refreshToken,
	}, nil
}

type TokenService struct {
	repository  *Repository
	secretStore *secrets.SecretStore
}

func NewTokenService(repository *Repository, secretStore *secrets.SecretStore) *TokenService {
	return &TokenService{
		repository:  repository,
		secretStore: secretStore,
	}
}

func (s *TokenService) GenerateJWT(user *users.User) (string, error) {
	secretValue, err := s.secretStore.Get("JWT_SIGNING_SECRET")
	if err != nil {
		return "", fmt.Errorf("reading JWT_SIGNING_SECRET: %w", err)
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    jwtIssuer,
		Audience:  jwt.ClaimStrings{jwtAudience},
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(jwtTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretValue.Value))
}

func (s *TokenService) GenerateRefreshToken(ctx context.Context, user *users.User) (string, error) {
	rawToken, tokenID, hash, err := newRefreshTokenValues()
	if err != nil {
		return "", err
	}

	token := &RefreshToken{
		RefreshTokenBase: RefreshTokenBase{
			UserID:    user.ID,
			Token:     hash,
			ExpiresAt: time.Now().Add(refreshTokenTTL),
		},
	}
	token.ID = tokenID
	if err := s.repository.CreateRefreshToken(ctx, token); err != nil {
		return "", err
	}

	return rawToken, nil
}

// RotateRefreshToken consumes a raw token of the form "<refresh-token-id>.<verifier>", validates
// the verifier against the stored bcrypt hash, and issues a replacement token.
func (s *TokenService) RotateRefreshToken(ctx context.Context, rawToken string) (*users.User, string, error) {
	tokenID, verifier, err := parseRefreshToken(rawToken)
	if err != nil {
		return nil, "", ErrInvalidRefreshToken
	}

	newRawToken, newID, newHash, err := newRefreshTokenValues()
	if err != nil {
		return nil, "", err
	}

	newToken := &RefreshToken{
		RefreshTokenBase: RefreshTokenBase{
			Token:     newHash,
			ExpiresAt: time.Now().Add(refreshTokenTTL),
		},
	}
	newToken.ID = newID

	userID, err := s.repository.RotateRefreshToken(ctx, tokenID, verifier, newToken)
	if err != nil {
		return nil, "", err
	}

	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}

	return user, newRawToken, nil
}

func newRefreshTokenValues() (rawToken string, tokenID uuid.UUID, hash string, err error) {
	tokenID, err = uuid.NewV7()
	if err != nil {
		return "", uuid.Nil, "", fmt.Errorf("generating refresh token id: %w", err)
	}

	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", uuid.Nil, "", fmt.Errorf("generating refresh token bytes: %w", err)
	}
	verifier := hex.EncodeToString(verifierBytes)

	hash, err = hashRefreshToken(verifier)
	if err != nil {
		return "", uuid.Nil, "", err
	}

	return tokenID.String() + "." + verifier, tokenID, hash, nil
}

func hashRefreshToken(verifier string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(verifier), refreshTokenBcryptCost)
	if err != nil {
		return "", ErrUnableToHashRefreshToken
	}
	return string(hash), nil
}

// parseRefreshToken splits a raw token into its lookup selector (row ID) and secret verifier.
func parseRefreshToken(rawToken string) (uuid.UUID, string, error) {
	idPart, verifier, found := strings.Cut(rawToken, ".")
	if !found || idPart == "" || verifier == "" {
		return uuid.Nil, "", ErrInvalidRefreshToken
	}

	id, err := uuid.FromString(idPart)
	if err != nil {
		return uuid.Nil, "", ErrInvalidRefreshToken
	}
	return id, verifier, nil
}
