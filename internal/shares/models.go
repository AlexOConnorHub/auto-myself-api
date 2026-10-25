package shares

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type VehicleUserAccessPending struct {
	helpers.DatabaseMetadata
	VehicleUserAccessBase
	CreatedBy     uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	CreatedByUser User      `gorm:"foreignKey:CreatedBy;references:ID;"`
	Vehicle       Vehicle   `gorm:"foreignKey:VehicleID;references:ID;"`
}

func (VehicleUserAccessPending) TableName() string {
	return "vehicle_user_access_pending"
}

func (s *VehicleUserAccessPending) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID.IsNil() {
		s.ID, err = uuid.NewV7()
	}
	return err
}

type VehicleUserAccessBase struct {
	UserID      uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;not null"`
	VehicleID   uuid.UUID `json:"vehicle_id,omitempty" gorm:"type:uuid;not null"`
	WriteAccess bool      `json:"write_access" gorm:"field:write_access;default:false"`
}

type VehicleUserAccess struct {
	helpers.DatabaseMetadata
	VehicleUserAccessBase
	CreatedBy     uuid.UUID  `json:"created_by" gorm:"type:uuid;not null"`
	CreatedByUser users.User `gorm:"foreignKey:CreatedBy;references:ID;"`
	Vehicle       Vehicle    `gorm:"foreignKey:VehicleID;references:ID;"`
}

func (VehicleUserAccess) TableName() string {
	return "vehicle_user_access"
}

func (vua *VehicleUserAccess) BeforeCreate(tx *gorm.DB) (err error) {
	if vua.ID.IsNil() {
		vua.ID, err = uuid.NewV7()
	}
	return err
}
