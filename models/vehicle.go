package models

import (
	"auto-myself-api/database"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type VehicleBase struct {
	Nickname  string    `json:"nickname" gorm:"type:text"`
	Make      string    `json:"make" gorm:"type:text"`
	MakeID    int       `json:"make_id" gorm:"type:integer"`
	Model     string    `json:"model" gorm:"type:text"`
	ModelID   int       `json:"model_id" gorm:"type:integer"`
	Year      int       `json:"year" gorm:"type:integer"`
	Vin       string    `json:"vin" gorm:"type:text"`
	Lpn       string    `json:"lpn" gorm:"type:text"`
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
}

type Vehicle struct {
	DatabaseMetadata
	VehicleBase
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
	println("Checking write access for user:", user.ID.String(), "on vehicle:", v.ID.String(), " (Which was created by ", v.CreatedBy.String(), ")")
	if v.CreatedBy == user.ID {
		return true
	}

	type Result struct {
		CanWrite bool `json:"can_write"`
	}
	var result Result
	err := database.DB.Raw(`
	SELECT
		true AS can_write
	FROM vehicle_user_access A
	WHERE A.vehicle_id = ?
		AND A.user_id = ?
		AND A.write_access = true
	LIMIT 1`, v.ID, user.ID).Scan(&result).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			database.LogError(err)
		}
		return false
	}
	return result.CanWrite
}
