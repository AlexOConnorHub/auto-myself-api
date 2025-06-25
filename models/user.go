package models

import (
	"gorm.io/gorm"
)

type UserBase struct {
	Username string `gorm:"type:text;" json:"username"`
}

type User struct {
	DatabaseMetadata
	UserBase
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	err = BeforeCreateSetupDatabaseMetadata(&u.DatabaseMetadata)
	if err != nil {
		return err
	}

	return
}
