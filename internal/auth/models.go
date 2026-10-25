package auth

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type IdentityBase struct {
	Provider string    `json:"provider,omitempty" gorm:"type:text;"`
	Subject  string    `json:"subject,omitempty" gorm:"type:text;"`
	UserID   uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;"`
	Email    string    `json:"email,omitempty" gorm:"type:text;"`
}

type Identity struct {
	helpers.DatabaseMetadata
	IdentityBase
	User users.User `gorm:"foreignKey:UserID;references:ID;"`
}

func (Identity) TableName() string {
	return "identities"
}

func (i *Identity) BeforeCreate(tx *gorm.DB) (err error) {
	if i.ID.IsNil() {
		i.ID, err = uuid.NewV7()
	}
	return err
}

type RefreshTokenBase struct {
	UserID    uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;"`
	Token     string    `json:"token,omitempty" gorm:"type:text;"`
	ExpiresAt time.Time `json:"expires_at,omitempty" gorm:"type:timestamp;"`
}

type RefreshToken struct {
	helpers.DatabaseMetadata
	RefreshTokenBase
	User users.User `gorm:"foreignKey:UserID;references:ID;"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (i *RefreshToken) BeforeCreate(tx *gorm.DB) (err error) {
	if i.ID.IsNil() {
		i.ID, err = uuid.NewV7()
	}
	return err
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type DevelopmentLoginRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

type UserAuthResponse struct {
	JWT     string `json:"jwt"`
	Refresh string `json:"refresh"`
}
