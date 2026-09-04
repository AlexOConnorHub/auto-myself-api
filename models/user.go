package models

import (
	"auto-myself-api/app"
	"auto-myself-api/helpers"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type UserBase struct {
	Username string `json:"username,omitempty" gorm:"type:text;"`
}

type User struct {
	helpers.DatabaseMetadata
	UserBase
	OwnedVehicles                []Vehicle                  `gorm:"foreignKey:CreatedBy;references:ID;" json:"-"`
	AccessedVehicles             []VehicleUserAccess        `gorm:"foreignKey:UserID;references:ID;" json:"-"`
	PendingVehicleSharesSent     []VehicleUserAccessPending `gorm:"foreignKey:CreatedBy;references:ID;" json:"-"`
	PendingVehicleSharesReceived []VehicleUserAccessPending `gorm:"foreignKey:UserID;references:ID;" json:"-"`
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
			fmt.Printf("Database error: %v\n", err)
		}
		return false
	}
	return result.CanRead
}

func (u *User) GenerateJWT(a *app.App) (string, error) {
	value, err := a.Secrets.Get("JWT_SIGNING_SECRET")
	if err != nil {
		return "", errors.New("JWT_SIGNING_SECRET not found in secrets: " + err.Error())
	}
	secret := value.Value

	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    "auto-myself-auth",
		Audience:  jwt.ClaimStrings{"auto-myself-auth"},
		Subject:   u.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 15)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (u *User) GenerateRefreshToken(tx *gorm.DB) (string, error) {
	b := make([]byte, 16)
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	if n != len(b) {
		return "", errors.New("could not read enough random bytes")
	}

	rawHash := sha256.Sum256(b)
	refreshTokenString := fmt.Sprintf("%x", rawHash[:])

	rawHash = sha256.Sum256([]byte(refreshTokenString))
	refreshTokenHashString := fmt.Sprintf("%x", rawHash[:])

	refreshToken := RefreshToken{
		RefreshTokenBase: RefreshTokenBase{
			UserID: u.ID,
			Token:  refreshTokenHashString,
		},
	}

	err = tx.Create(&refreshToken).Error
	if err != nil {
		return "", err
	}

	return refreshTokenString, nil
}
