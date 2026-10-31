package users

import (
	"auto-myself-api/internal/helpers"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type UserPost struct {
	Username string `json:"username,omitempty" gorm:"type:text;"`
}

type UserWeb struct {
	UserPost
	ID uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
}

type User struct {
	UserWeb
	Role      string `json:"role" gorm:"type:text;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.Role == "" {
		u.Role = RoleUser
	}
	return helpers.BeforeCreateID(tx, u)
}

const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)
