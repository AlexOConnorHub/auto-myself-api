package models

import (
	"auto-myself-api/helpers"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type ReportBase struct {
	ReportedTable string    `json:"reported_table,omitempty" gorm:"type:text;"`
	ReportedID    uuid.UUID `json:"reported_id,omitempty" gorm:"type:uuid;"`
	CreatedBy     uuid.UUID `json:"created_by,omitempty" gorm:"type:uuid;"`
	Reason        string    `json:"reason,omitempty" gorm:"type:text;"`
}

type Report struct {
	helpers.DatabaseMetadata
	ReportBase
	Resolved      bool       `json:"resolved,omitempty" gorm:"type:boolean;"`
	ResolvedBy    *uuid.UUID `json:"resolved_by,omitempty" gorm:"type:uuid;"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty" gorm:"type:timestamp with time zone;"`
	InternalNotes string     `json:"internal_notes,omitempty" gorm:"type:text;"`
}

func (Report) TableName() string {
	return "reports"
}

func (r *Report) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID.IsNil() {
		r.ID, err = uuid.NewV7()
	}
	return err
}
