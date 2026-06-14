package models

import (
	"auto-myself-api/helpers"

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
	User User `gorm:"foreignKey:UserID;references:ID;"`
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
