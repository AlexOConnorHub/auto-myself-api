package models

import (
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type VehicleUserAccessBase struct {
	Username    string    `json:"username" gorm:"type:text;"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	GrantedUser User      `gorm:"foreignKey:UserID;references:ID;constraint"`
	VehicleID   uuid.UUID `json:"vehicle_id" gorm:"type:uuid;not null"`
	CanWrite    bool      `json:"can_write" gorm:"field:write_access;default:false"`
}

type VehicleUserAccess struct {
	DatabaseMetadata
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
