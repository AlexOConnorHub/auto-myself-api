package models

import (
	"auto-myself-api/helpers"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type RefreshTokenBase struct {
	UserID    uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;"`
	Token     string    `json:"token,omitempty" gorm:"type:text;"`
	ExpiresAt time.Time `json:"expires_at,omitempty" gorm:"type:timestamp;"`
}

type RefreshToken struct {
	helpers.DatabaseMetadata
	RefreshTokenBase
	User User `gorm:"foreignKey:UserID;references:ID;"`
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
