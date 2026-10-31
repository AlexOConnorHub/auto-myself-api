package auth

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type AuthRepository interface {
	FindUserByID(context.Context, uuid.UUID) (*users.User, error)
	FindUserByProviderAndSubject(context.Context, string, string) (*users.User, error)
	CreateUserWithIdentity(context.Context, *users.User, *Identity) error
	CreateRefreshToken(context.Context, *RefreshToken) error
	RotateRefreshToken(context.Context, uuid.UUID, string, *RefreshToken) (uuid.UUID, error)
}

type Repository struct {
	database *database.Database
}

func NewAuthRepository(database *database.Database) *Repository {
	return &Repository{
		database: database,
	}
}

func (r *Repository) FindUserByID(ctx context.Context, id uuid.UUID) (*users.User, error) {
	user, err := gorm.G[users.User](r.database.GetGorm()).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindUserByProviderAndSubject(ctx context.Context, provider string, subject string) (*users.User, error) {
	identity, err := gorm.G[Identity](r.database.GetGorm()).
		Where("provider = ? AND subject = ?", provider, subject).
		First(ctx)
	if err != nil {
		return nil, err
	}

	user, err := gorm.G[users.User](r.database.GetGorm()).Where("id = ?", identity.UserID).First(ctx)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) CreateUserWithIdentity(ctx context.Context, user *users.User, identity *Identity) error {
	return r.database.GetGorm().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := gorm.G[users.User](tx).Create(ctx, user); err != nil {
			return err
		}
		identity.UserID = user.ID
		return gorm.G[Identity](tx).Create(ctx, identity)
	})
}

func (r *Repository) CreateRefreshToken(ctx context.Context, token *RefreshToken) error {
	return helpers.Create[RefreshToken](ctx, r.database.GetGorm(), token)
}

// RotateRefreshToken validates the verifier against the stored bcrypt hash and, in one
// transaction, consumes the old row and inserts newToken. Unknown, expired, or mismatched
// tokens yield gorm.ErrRecordNotFound.
func (r *Repository) RotateRefreshToken(ctx context.Context, tokenID uuid.UUID, verifier string, newToken *RefreshToken) (uuid.UUID, error) {
	var userID uuid.UUID

	err := r.database.GetGorm().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stored, err := gorm.G[RefreshToken](tx).
			Where("id = ? AND expires_at > ?", tokenID, time.Now()).
			First(ctx)
		if err != nil {
			return err
		}

		if err := bcrypt.CompareHashAndPassword([]byte(stored.Token), []byte(verifier)); err != nil {
			return gorm.ErrRecordNotFound
		}

		rows, err := gorm.G[RefreshToken](tx).Where("id = ?", stored.ID).Delete(ctx)
		if err != nil {
			return err
		}
		if rows != 1 {
			return helpers.ErrRowCountOff
		}

		newToken.UserID = stored.UserID
		if err := gorm.G[RefreshToken](tx).Create(ctx, newToken); err != nil {
			return err
		}

		userID = stored.UserID
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

// compile-time assertion that the concrete repository satisfies its interface.
var _ AuthRepository = (*Repository)(nil)

type OAuthRepository interface {
	AuthCodeURL(conf *oauth2.Config, state string) string
	ExchangeUserInfo(context.Context, *oauth2.Config, string) (*googleUserInfo, error)
}

type GoogleOAuthRepository struct{}

func NewGoogleOAuthRepository() *GoogleOAuthRepository {
	return &GoogleOAuthRepository{}
}

func (r *GoogleOAuthRepository) AuthCodeURL(conf *oauth2.Config, state string) string {
	return conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (r *GoogleOAuthRepository) ExchangeUserInfo(ctx context.Context, conf *oauth2.Config, code string) (*googleUserInfo, error) {
	token, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchanging oauth code: %w", err)
	}

	response, err := conf.Client(ctx, token).Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("fetching google user info: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google user info returned %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading google user info: %w", err)
	}

	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decoding google user info: %w", err)
	}
	return &info, nil
}

var _ OAuthRepository = (*GoogleOAuthRepository)(nil)
