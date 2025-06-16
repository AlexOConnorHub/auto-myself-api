package models

import (
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type User struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	Username   string `gorm:"type:text;"`
	CreatedAt  string `gorm:"type:timestamptz;default:now()"`
	UpdatedAt  string `gorm:"type:timestamptz;default:now()"`
	DeletedAt  string `gorm:"type:timestamptz"`
	PublicKey  string `gorm:"type:text;"`
	PrivateKey string `gorm:"type:text;"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = uuidv7.New().String()

	return
}
