package vehicles

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type VehiclePost struct {
	Nickname string `json:"nickname,omitempty" gorm:"type:text"`
	Make     string `json:"make,omitempty" gorm:"type:text"`
	MakeID   int    `json:"make_id,omitempty" gorm:"type:integer"`
	Model    string `json:"model,omitempty" gorm:"type:text"`
	ModelID  int    `json:"model_id,omitempty" gorm:"type:integer"`
	Year     int    `json:"year,omitempty" gorm:"type:integer"`
	Vin      string `json:"vin,omitempty" gorm:"type:text"`
	Lpn      string `json:"lpn,omitempty" gorm:"type:text"`
}

type VehicleUpdatePost struct {
	VehiclePost
	Version int64 `json:"version" binding:"required,gt=0"`
}

type VehicleWeb struct {
	VehiclePost
	ID      uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Version int64     `json:"version" gorm:"not null"`
}

type Vehicle struct {
	VehicleWeb
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	CreatedBy uuid.UUID      `json:"created_by" gorm:"type:uuid;not null"`
}

func (Vehicle) TableName() string {
	return "vehicles"
}

func (v *Vehicle) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID.IsNil() {
		v.ID, err = uuid.NewV7()
	}
	if v.Version == 0 {
		v.Version = 1
	}
	return err
}

// func (v *Vehicle) AfterDelete(tx *gorm.DB) (err error) {
// 	if err := tx.Where("vehicle_id = ?", v.ID).Delete(&MaintenanceRecord{}).Error; err != nil {
// 		fmt.Printf("Database error: %v\n", err)
// 		return err
// 	}

// 	if err := tx.Where("vehicle_id = ?", v.ID).Delete(&VehicleUserAccess{}).Error; err != nil {
// 		fmt.Printf("Database error: %v\n", err)
// 		return err
// 	}

// 	return nil
// }
