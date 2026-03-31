package models

import (
	"auto-myself-api/app"
	"auto-myself-api/database"
	"auto-myself-api/helpers"
	"errors"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type UserBase struct {
	Username string `json:"username,omitempty" gorm:"type:text;"`
}

type User struct {
	helpers.DatabaseMetadata
	UserBase
	OwnedVehicles                []Vehicle                  `gorm:"foreignKey:CreatedBy;references:ID;constraint" json:"-"`
	AccessedVehicles             []VehicleUserAccess        `gorm:"foreignKey:UserID;references:ID;constraint" json:"-"`
	PendingVehicleSharesSent     []VehicleUserAccessPending `gorm:"foreignKey:CreatedBy;references:ID;constraint" json:"-"`
	PendingVehicleSharesReceived []VehicleUserAccessPending `gorm:"foreignKey:UserID;references:ID;constraint" json:"-"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID.IsNil() {
		u.DatabaseMetadata.ID, err = uuid.NewV7()
	}
	return err
}

func (u *User) CanRead(a *app.App, user User) bool {
	if u.ID == user.ID {
		return true
	}

	type Result struct {
		CanRead bool `json:"can_read"`
	}
	var result Result

	err := a.Gorm.Raw(`
	SELECT true AS can_read
	FROM vehicles V
	LEFT JOIN vehicle_user_access A ON V.id = A.vehicle_id
	WHERE
		( V.created_by = ? AND A.user_id = ?)
		OR
		(
			V.id IN (
				SELECT V_INNER.id
				FROM vehicles V_INNER
				LEFT JOIN vehicle_user_access A_INNER ON V_INNER.id = A_INNER.vehicle_id
				WHERE A_INNER.user_id = ? OR V_INNER.created_by = ?
			)
			AND
			(
				( A.user_id = ? AND A.write_access = true )
				OR
				V.created_by = ?
			)
		)
	LIMIT 1`, u.ID, user.ID, u.ID, u.ID, user.ID, user.ID).Scan(&result).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
			return false
		}
		return false
	}
	return result.CanRead
}

func (u *User) GenerateJWT() (string, error) {
	secret := os.Getenv("JWT_SIGNING_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SIGNING_SECRET environment variable is not set")
	}

	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    "auto-myself-api",
		Audience:  jwt.ClaimStrings{"auto-myself-api"},
		Subject:   u.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
