package models

import (
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type UserBase struct {
	Username string `json:"username" gorm:"type:text;"`
}

type User struct {
	DatabaseMetadata
	UserBase
	OwnedVehicles    []Vehicle           `gorm:"foreignKey:CreatedBy;references:ID;constraint"`
	AccessedVehicles []VehicleUserAccess `gorm:"foreignKey:UserID;references:ID;constraint"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID.IsNil() {
		u.ID, err = uuid.NewV7()
	}
	return err
}
