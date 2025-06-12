package models

import (
	"github.com/samborkent/uuidv7"
)

type User struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	Username  string `gorm:"type:text;"`
	CreatedAt string `gorm:"type:timestamptz;default:now()"`
	UpdatedAt string `gorm:"type:timestamptz;default:now()"`
	DeletedAt string `gorm:"type:timestamptz"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate() error {
	// Generate a new UUID for the user ID
	u.ID = uuidv7.New().String()
	return nil
}
