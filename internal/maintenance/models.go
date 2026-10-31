package maintenance

import (
	"auto-myself-api/internal/helpers"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type MaintenanceRecordFields struct {
	Cost         string    `json:"cost,omitempty" gorm:"type:decimal(10,2)"`
	Odometer     int       `json:"odometer,omitempty" gorm:"type:integer"`
	Timestamp    time.Time `json:"timestamp,omitempty" gorm:"type:date"`
	Notes        string    `json:"notes,omitempty" gorm:"type:text"`
	Type         string    `json:"type,omitempty" gorm:"type:text"`
	Interval     int       `json:"interval,omitempty" gorm:"type:integer"`
	IntervalType string    `json:"interval_type,omitempty" gorm:"type:text"`
}

type MaintenanceRecordPost struct {
	MaintenanceRecordFields
	VehicleID uuid.UUID `json:"vehicle_id,omitempty" gorm:"type:uuid;not null" binding:"required"`
}

// MaintenanceRecordUpdatePost omits VehicleID as it may not change
type MaintenanceRecordUpdatePost struct {
	MaintenanceRecordFields
}

type MaintenanceRecordWeb struct {
	MaintenanceRecordPost
	ID        uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
}

type MaintenanceRecord struct {
	MaintenanceRecordWeb
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (MaintenanceRecord) TableName() string {
	return "maintenance_records"
}

func (m *MaintenanceRecord) BeforeCreate(tx *gorm.DB) (err error) {
	return helpers.BeforeCreateID(tx, m)
}
