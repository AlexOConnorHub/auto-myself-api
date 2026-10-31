package auth

import (
	"auto-myself-api/internal/helpers"
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
	IdentityBase
	ID        uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Identity) TableName() string {
	return "identities"
}

func (i *Identity) BeforeCreate(tx *gorm.DB) (err error) {
	return helpers.BeforeCreateID(tx, i)
}

type RefreshTokenBase struct {
	UserID    uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;not null"`
	Token     string    `json:"token,omitempty" gorm:"type:text;not null"`
	ExpiresAt time.Time `json:"expires_at,omitempty" gorm:"type:timestamp with time zone;not null"`
}

type RefreshToken struct {
	RefreshTokenBase
	ID        uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) (err error) {
	return helpers.BeforeCreateID(tx, r)
}

type MobileLoginRequest struct {
	IDToken string `json:"id_token" binding:"required,max=8192"`
	Nonce   string `json:"nonce" binding:"required,max=512"`
	Name    string `json:"name" binding:"omitempty,max=255"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type DevelopmentLoginRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

type UserAuthResponse struct {
	JWT     string `json:"jwt"`
	Refresh string `json:"refresh"`
}
