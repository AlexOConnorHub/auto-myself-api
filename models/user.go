package models

import (
	"auto-myself-api/database"
	"auto-myself-api/helpers"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type UserBase struct {
	Username string `json:"username,omitempty" gorm:"type:text;"`
}

type User struct {
	helpers.DatabaseMetadata
	UserBase
	OwnedVehicles    []Vehicle           `gorm:"foreignKey:CreatedBy;references:ID;constraint"`
	AccessedVehicles []VehicleUserAccess `gorm:"foreignKey:UserID;references:ID;constraint"`
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

func (u *User) GetLocation() string {
	return "/user/" + u.ID.String()
}

func (u *User) CanRead(user User) bool {
	if u.ID == user.ID {
		return true
	}

	type Result struct {
		CanRead bool `json:"can_read"`
	}
	var result Result

	err := database.DB.Raw(`
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
