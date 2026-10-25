package auth

import (
	"auto-myself-api/internal/server"
	"auto-myself-api/internal/users"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

type AuthService interface {
	// FindUserByRefreshToken(string) (string, error)
	LoginWebInitiate(context.Context, string) (string, error)
	LoginWebCallback(context.Context, string, string) (*UserAuthResponse, error)
	RefreshToken(context.Context, RefreshTokenRequest) (*UserAuthResponse, error)
	DevelopmentLogin(context.Context, DevelopmentLoginRequest) (*UserAuthResponse, error)
}

type authService struct {
	authRepository *authRepository
	tokenService   *tokenService
}

func NewAuthService(authRepository *authRepository, tokenService *tokenService) *authService {
	return &authService{
		authRepository: authRepository,
		tokenService:   tokenService,
	}
}

func googleProvider(c *gin.Context, a *server.App) {
	conf := &oauth2.Config{
		ClientID:     a.Secrets.Get("GOOGLE_OAUTH2_CLIENT_ID").Value,
		ClientSecret: a.Secrets.Get("GOOGLE_OAUTH2_CLIENT_SECRET").Value,
		RedirectURL:  "http://localhost.automyself.com:8080/auth/google",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}
	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL("state")
	fmt.Printf("Visit the URL for the auth dialog: %v", url)

	// Handle the exchange code to initiate a transport.
	tok, err := conf.Exchange(c, "authorization-code")
	if err != nil {
		log.Fatal(err)
	}

	client := conf.Client(c, tok)
	client.Get("...")
}

func appleProvider(c *gin.Context, a *server.App) {
}

func (s *authService) LoginWebInitiate(ctx context.Context, provider string) (string, error) {
	switch provider {
	case "google":
		return s.googleWebInitiate(ctx)
	// case "apple":
	// Implement Apple provider login logic here
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (s *authService) LoginWebCallback(ctx context.Context, provider string, code string) (*UserAuthResponse, error) {
	switch provider {
	case "google":
		return s.googleWebCallback(ctx, code)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (s *authService) RefreshToken(ctx context.Context, request RefreshTokenRequest) (*UserAuthResponse, error) {
	hash := sha256.Sum256([]byte(request.RefreshToken))
	hashedUserToken := fmt.Sprintf("%x", hash[:])

	user, err := s.authRepository.FindUserByRefreshToken(ctx, hashedUserToken)
	if err != nil {
		return nil, fmt.Errorf("authService.RefreshToken: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("authService.RefreshToken: user not found")
	}

	userAuth, err := s.userToUserAuthResponse(user)
	if err != nil {
		return nil, fmt.Errorf("authService.RefreshToken: %w", err)
	}

	return userAuth, nil
}

func (s *authService) DevelopmentLogin(ctx context.Context, request DevelopmentLoginRequest) (*UserAuthResponse, error) {
	user, err := s.authRepository.FindUserByID(ctx, request.UserID)
	if err != nil {
		return nil, fmt.Errorf("authService.DevelopmentLogin: %w", err)
	}

	return s.userToUserAuthResponse(user)
}

func (s *authService) googleWebInitiate(ctx context.Context) (string, error) {
	conf := &oauth2.Config{
		ClientID:     s.authRepository.secrets.Get("GOOGLE_OAUTH2_CLIENT_ID").Value,
		ClientSecret: s.authRepository.secrets.Get("GOOGLE_OAUTH2_CLIENT_SECRET").Value,
		RedirectURL:  os.Getenv("DOMAIN") + "/auth/callback/google",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	return url, nil
}

func (s *authService) googleWebCallback(ctx context.Context, code string) (*UserAuthResponse, error) {
	conf := &oauth2.Config{
		ClientID:     s.authRepository.secrets.Get("GOOGLE_OAUTH2_CLIENT_ID").Value,
		ClientSecret: s.authRepository.secrets.Get("GOOGLE_OAUTH2_CLIENT_SECRET").Value,
		RedirectURL:  os.Getenv("DOMAIN") + "/auth/callback/google",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, errors.New("Failed to exchange code for token")
	}

	client := conf.Client(ctx, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, errors.New("Failed to get user info")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("Failed to read user info response")
	}

	googleData := googleUserInfo{}
	if err := json.Unmarshal(body, &googleData); err != nil {
		return nil, errors.New("Failed to decode user info")
	}

	user, err := s.authRepository.FindUserByProviderAndSubject(ctx, "google", googleData.Sub)

	if errors.Is(err, gorm.ErrRecordNotFound) {

		user = &users.User{
			UserWeb: users.UserWeb{
				UserPost: users.UserPost{
					Username: googleData.Name,
				},
			},
		}

		if err := s.authRepository.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("Failed to create user, %w", err)
		}

		identity := Identity{
			IdentityBase: IdentityBase{
				Provider: "google",
				Subject:  googleData.Sub,
				Email:    googleData.Email,
			},
			User: *user,
		}

		if err := s.authRepository.CreateIdentity(ctx, &identity); err != nil {
			return nil, fmt.Errorf("Failed to create identity, %w", err)
		}

	}

	return s.userToUserAuthResponse(user)
}

func (s *authService) userToUserAuthResponse(user *users.User) (*UserAuthResponse, error) {
	if user == nil {
		return nil, fmt.Errorf("authService.userToUserAuthResponse: user is nil")
	}

	jwt, err := s.tokenService.tokenRepository.GenerateJWTForUser(user)
	if err != nil {
		return nil, fmt.Errorf("authService.userToUserAuthResponse: %w", err)
	}

	refreshToken, err := s.tokenService.tokenRepository.GenerateRefreshTokenForUser(user)
	if err != nil {
		return nil, fmt.Errorf("authService.userToUserAuthResponse: %w", err)
	}

	return &UserAuthResponse{
		JWT:     jwt,
		Refresh: refreshToken,
	}, nil
}

/**

type keyCacheItem struct {
	PubKey     string
	expiration int64
}

var keyCache = make(map[string]keyCacheItem)

func getAuthKey(provider string, kid string) (string, error) {
	cacheKey := provider + ":" + kid
	if item, found := keyCache[cacheKey]; found {
		if item.expiration > time.Now().Unix() {
			return item.PubKey, nil
		}
		delete(keyCache, cacheKey) // Remove expired key
	}

	var key keyCacheItem
	if provider == "google" {

	} else if provider == "apple" {

	}

	keyCache[cacheKey] = key
	return key.PubKey, nil

}
*/

// type TokenService interface {
// 	// GenerateJWTForUser(*users.User) (string, error)
// 	RefreshToken(string) (string, error)
// 	userToUserAuthResponse(user *users.User) (UserAuthResponse, error)
// }

type tokenService struct {
	tokenRepository *tokenRepository
}

func NewTokenService(tokenRepository *tokenRepository) *tokenService {
	return &tokenService{
		tokenRepository: tokenRepository,
	}
}
