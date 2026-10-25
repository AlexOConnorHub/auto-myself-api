package auth

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/users"
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthRepository interface {
	FindUserByID(context.Context, string) (*users.User, error)
	FindUserByRefreshToken(context.Context, string) (*users.User, error)
	FindUserByProviderAndSubject(context.Context, string, string) (*users.User, error)
	CreateIdentity(context.Context, *Identity) error
	CreateUser(context.Context, *users.User) error
}

type authRepository struct {
	database *database.Database
	secrets  *secrets.SecretStore
}

func NewAuthRepository(database *database.Database, secrets *secrets.SecretStore) *authRepository {
	return &authRepository{
		database: database,
		secrets:  secrets,
	}
}

func (r *authRepository) FindUserByID(ctx context.Context, id string) (*users.User, error) {
	user, err := gorm.G[users.User](r.database.GetGorm()).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *authRepository) FindUserByRefreshToken(ctx context.Context, token string) (*users.User, error) {
	refreshToken, err := gorm.G[RefreshToken](r.database.GetGorm()).Where("token = ?", token).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var user users.User
	if err := r.database.GetGorm().Model(&refreshToken).Association("User").Find(&user); err != nil {
		return nil, err
	}

	if err := r.database.GetGorm().Delete(&refreshToken).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *authRepository) FindUserByProviderAndSubject(ctx context.Context, provider string, subject string) (*users.User, error) {
	identity, err := gorm.G[Identity](r.database.GetGorm()).Preload("User", nil).Where("provider = ? AND subject = ?", provider, subject).First(ctx)
	if err != nil {
		return nil, err
	}

	return &identity.User, nil
}

func (r *authRepository) CreateIdentity(ctx context.Context, identity *Identity) error {
	return gorm.G[Identity](r.database.GetGorm()).Create(ctx, identity)
}

func (r *authRepository) CreateUser(ctx context.Context, user *users.User) error {
	return gorm.G[users.User](r.database.GetGorm()).Create(ctx, user)
}

type TokenRepository interface {
	GenerateJWTForUser(*users.User) (string, error)
	GenerateRefreshTokenForUser(*users.User) (string, error)
	RefreshToken(string) (string, error)
	FindUserByRefreshToken(context.Context, string) (string, error)
}

type tokenRepository struct {
	secretStore *secrets.SecretStore
}

func NewTokenRepository(secretStore *secrets.SecretStore) *tokenRepository {
	return &tokenRepository{
		secretStore: secretStore,
	}
}

func (r *tokenRepository) GenerateJWTForUser(user *users.User) (string, error) {
	value := r.secretStore.Get("JWT_SIGNING_SECRET")
	secret := value.Value

	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    "auto-myself-auth",
		Audience:  jwt.ClaimStrings{"auto-myself-auth"},
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 15)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (r *tokenRepository) GenerateRefreshTokenForUser(user *users.User) (string, error) {
	// Implement the logic to generate a refresh token for the user here
	return "", nil
}

func (r *tokenRepository) RefreshToken(token string) (string, error) {
	// Implement the logic to refresh the token here
	return "", nil
}
