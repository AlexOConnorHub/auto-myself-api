package models

import (
	"auto-myself-api/helpers"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type VehicleUserAccessBase struct {
	Username  string    `json:"username,omitempty" gorm:"type:text;"`
	UserID    uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;not null"`
	VehicleID uuid.UUID `json:"vehicle_id,omitempty" gorm:"type:uuid;not null"`
	CanWrite  bool      `json:"can_write,omitempty" gorm:"field:write_access;default:false"`
}

type VehicleUserAccess struct {
	helpers.DatabaseMetadata
	VehicleUserAccessBase
	CreatedBy     uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	CreatedByUser User      `gorm:"foreignKey:CreatedBy;references:ID;constraint"`
	Vehicle       Vehicle   `gorm:"foreignKey:VehicleID;references:ID;constraint"`
}

func (VehicleUserAccess) TableName() string {
	return "vehicle_user_access"
}

func (vua *VehicleUserAccess) BeforeCreate(tx *gorm.DB) (err error) {
	if vua.ID.IsNil() {
		vua.DatabaseMetadata.ID, err = uuid.NewV7()
	}
	return err
}
