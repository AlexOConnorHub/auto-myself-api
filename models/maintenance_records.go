package models

import (
	"github.com/samborkent/uuidv7"
)

type MaintenanceRecord struct {
	ID          string  `gorm:"type:uuid;primaryKey"`
	VehicleID   string  `gorm:"type:uuid;not null"`
	Description string  `gorm:"type:text;not null"`
	Date        string  `gorm:"type:timestamptz;not null"`
	Cost        float64 `gorm:"type:decimal(10,2);not null"`
	CreatedAt   string  `gorm:"type:timestamptz;default:now()"`
	UpdatedAt   string  `gorm:"type:timestamptz;default:now()"`
	DeletedAt   string  `gorm:"type:timestamptz"`
}

func (MaintenanceRecord) TableName() string {
	return "maintenance_records"
}

func (m *MaintenanceRecord) BeforeCreate() error {
	// Generate a new UUID for the user ID
	m.ID = uuidv7.New().String()
	return nil
}
