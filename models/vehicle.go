package models

import (
	"auto-myself-api/database"
	"auto-myself-api/helpers"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type VehicleBase struct {
	Nickname string `json:"nickname,omitempty" gorm:"type:text"`
	Make     string `json:"make,omitempty" gorm:"type:text"`
	MakeID   int    `json:"make_id,omitempty" gorm:"type:integer"`
	Model    string `json:"model,omitempty" gorm:"type:text"`
	ModelID  int    `json:"model_id,omitempty" gorm:"type:integer"`
	Year     int    `json:"year,omitempty" gorm:"type:integer"`
	Vin      string `json:"vin,omitempty" gorm:"type:text"`
	Lpn      string `json:"lpn,omitempty" gorm:"type:text"`
}

type Vehicle struct {
	helpers.DatabaseMetadata
	VehicleBase
	CreatedBy          uuid.UUID           `json:"created_by" gorm:"type:uuid;not null"`
	CreatedByUser      User                `gorm:"foreignKey:CreatedBy;references:ID;constraint"`
	MaintenanceRecords []MaintenanceRecord `gorm:"foreignKey:VehicleID;references:ID;constraint"`
}

func (Vehicle) TableName() string {
	return "vehicles"
}

func (v *Vehicle) GetLocation() string {
	return "/vehicle/" + v.DatabaseMetadata.ID.String()
}

func (v *Vehicle) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID.IsNil() {
		v.DatabaseMetadata.ID, err = uuid.NewV7()
	}
	return err
}

func (v *Vehicle) AfterDelete(tx *gorm.DB) (err error) {
	if err := database.DB.Where("vehicle_id = ?", v.ID).Delete(&MaintenanceRecord{}).Error; err != nil {
		database.LogError(err)
		return err
	}

	if err := database.DB.Where("vehicle_id = ?", v.ID).Delete(&VehicleUserAccess{}).Error; err != nil {
		database.LogError(err)
		return err
	}

	return nil
}

func (v *Vehicle) CanRead(user User) bool {
	if v.CreatedBy == user.ID {
		return true
	}

	type Result struct {
		CanRead bool `json:"can_read"`
	}
	var result Result
	err := database.DB.Raw(`
	SELECT
		true AS can_read
	FROM vehicle_user_access A
	WHERE A.vehicle_id = ? AND A.user_id = ?
	LIMIT 1`, v.ID, user.ID).Scan(&result).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
		}
		return false
	}

	return result.CanRead
}

func (v *Vehicle) CanWrite(user User) bool {
	if v.CreatedBy == user.ID {
		return true
	}

	type Result struct {
		WriteAccess bool `json:"write_access"`
	}
	var result Result
	err := database.DB.Raw(`
	SELECT
		A.write_access
	FROM vehicle_user_access A
	WHERE A.vehicle_id = ?
		AND A.user_id = ?
	LIMIT 1`, v.ID, user.ID).Scan(&result).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
		}
		return false
	}
	return result.WriteAccess
}
