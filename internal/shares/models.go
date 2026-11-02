package shares

import (
	"auto-myself-api/internal/helpers"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type VehicleUserAccessBase struct {
	UserID      uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;not null" binding:"required"`
	VehicleID   uuid.UUID `json:"vehicle_id,omitempty" gorm:"type:uuid;not null" binding:"required"`
	WriteAccess bool      `json:"write_access" gorm:"column:write_access;not null;default:false"`
}

type VehicleUserAccessPending struct {
	VehicleUserAccessBase
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;not null"`
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (VehicleUserAccessPending) TableName() string {
	return "vehicle_user_access_pending"
}

func (s *VehicleUserAccessPending) BeforeCreate(tx *gorm.DB) (err error) {
	return helpers.BeforeCreateID(tx, s)
}

type VehicleUserAccess struct {
	VehicleUserAccessBase
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;not null"`
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	Version   int64     `json:"version" gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (VehicleUserAccess) TableName() string {
	return "vehicle_user_access"
}

func (v *VehicleUserAccess) BeforeCreate(tx *gorm.DB) (err error) {
	if err := helpers.BeforeCreateID(tx, v); err != nil {
		return err
	}
	if v.Version == 0 {
		v.Version = 1
	}
	return nil
}

type VehicleUserAccessUpdatePost struct {
	WriteAccess *bool `json:"write_access" binding:"required"`
	Version     int64 `json:"version" binding:"required,gt=0"`
}

type VehicleUserAccessWeb struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	WriteAccess bool      `json:"write_access"`
	Version     int64     `json:"version"`
	CreatedAt   string    `json:"created_at"`
}

func NewVehicleUserAccessWeb(access VehicleUserAccess) VehicleUserAccessWeb {
	return VehicleUserAccessWeb{
		ID:          access.ID,
		UserID:      access.UserID,
		WriteAccess: access.WriteAccess,
		Version:     access.Version,
		CreatedAt:   access.CreatedAt.Format(time.RFC3339),
	}
}
