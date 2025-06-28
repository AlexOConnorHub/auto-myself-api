package models

import (
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type MaintenanceRecordBase struct {
	Description  string    `json:"description" gorm:"type:text;not null"`
	Cost         float64   `json:"cost" gorm:"type:decimal(10,2);not null"`
	Odometer     int       `json:"odometer" gorm:"type:integer;not null"`
	Timestamp    string    `json:"timestamp" gorm:"type:timestamp;not null"`
	Notes        string    `json:"notes" gorm:"type:text"`
	Type         string    `json:"type" gorm:"type:text;not null"`
	Interval     string    `json:"interval" gorm:"type:text;not null"`
	IntervalType string    `json:"interval_type" gorm:"type:text;not null"`
	CreatedBy    uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	VehicleID    uuid.UUID `json:"vehicle_id" gorm:"type:uuid;not null"`
}

type MaintenanceRecord struct {
	DatabaseMetadata
	MaintenanceRecordBase
	CreatedByUser User    `gorm:"foreignKey:CreatedBy;references:ID;constraint"`
	Vehicle       Vehicle `gorm:"foreignKey:VehicleID;references:ID;constraint"`
}

func (MaintenanceRecord) TableName() string {
	return "maintenance_records"
}

func (m *MaintenanceRecord) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID.IsNil() {
		m.ID, err = uuid.NewV7()
	}
	return err
}
