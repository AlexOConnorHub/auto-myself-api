package models

import (
	"auto-myself-api/helpers"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type VehicleUserAccessPending struct {
	helpers.DatabaseMetadata
	VehicleUserAccessBase
	CreatedBy     uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	CreatedByUser User      `gorm:"foreignKey:CreatedBy;references:ID;constraint"`
	Vehicle       Vehicle   `gorm:"foreignKey:VehicleID;references:ID;constraint"`
}

func (VehicleUserAccessPending) TableName() string {
	return "vehicle_user_access_pending"
}

func (s *VehicleUserAccessPending) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID.IsNil() {
		s.DatabaseMetadata.ID, err = uuid.NewV7()
	}
	return err
}
