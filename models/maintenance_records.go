package models

import "github.com/gofrs/uuid"

type MaintenanceRecord struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	VehicleID   string    `gorm:"type:uuid;not null"`
	Description string    `gorm:"type:text;not null"`
	Date        string    `gorm:"type:timestamptz;not null"`
	Cost        float64   `gorm:"type:decimal(10,2);not null"`
	CreatedAt   string    `gorm:"type:timestamptz;default:now()"`
	UpdatedAt   string    `gorm:"type:timestamptz;default:now()"`
	DeletedAt   string    `gorm:"type:timestamptz"`
}

func (MaintenanceRecord) TableName() string {
	return "maintenance_records"
}

func (m *MaintenanceRecord) BeforeCreate() (err error) {
	// Generate a new UUID for the user ID
	m.ID, err = uuid.NewV7()
	return nil
}
