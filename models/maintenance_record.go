package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type MaintenanceRecordBase struct {
	Cost         string    `json:"cost" gorm:"type:decimal(10,2)"`
	Odometer     int       `json:"odometer" gorm:"type:integer"`
	Timestamp    time.Time `json:"timestamp" gorm:"type:date"`
	Notes        string    `json:"notes" gorm:"type:text"`
	Type         string    `json:"type" gorm:"type:text"`
	Interval     int       `json:"interval" gorm:"type:integer"`
	IntervalType string    `json:"interval_type" gorm:"type:text"`
	VehicleID    uuid.UUID `json:"vehicle_id" gorm:"type:uuid;not null"`
}

type MaintenanceRecord struct {
	DatabaseMetadata
	MaintenanceRecordBase
	CreatedBy     uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	CreatedByUser User      `gorm:"foreignKey:CreatedBy;references:ID;constraint"`
	Vehicle       Vehicle   `gorm:"foreignKey:VehicleID;references:ID;constraint"`
}

func (MaintenanceRecord) TableName() string {
	return "maintenance_records"
}

func (m *MaintenanceRecord) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID.IsNil() {
		m.DatabaseMetadata.ID, err = uuid.NewV7()
	}
	return err
}

func (m *MaintenanceRecord) GetLocation() string {
	return "/maintenance/" + m.DatabaseMetadata.ID.String()
}
