package reports

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type ReportPost struct {
	ReportedTopic string    `json:"reported_topic,omitempty" gorm:"type:text;"`
	ReportedID    uuid.UUID `json:"reported_id,omitempty" gorm:"type:uuid;"`
	Reason        string    `json:"reason,omitempty" gorm:"type:text;"`
}

type ReportResolutionPost struct {
	Resolved      bool   `json:"resolved,omitempty"`
	PublicNotes   string `json:"public_notes,omitempty"`
	InternalNotes string `json:"internal_notes,omitempty"`
}

type ReportWeb struct {
	ReportPost
	CreatedBy   uuid.UUID `json:"created_by,omitempty" gorm:"type:uuid;"`
	PublicNotes string    `json:"public_notes,omitempty" gorm:"type:text;"`
	Resolved    bool      `json:"resolved,omitempty" gorm:"type:boolean;"`
	ResolvedAt  time.Time `json:"resolved_at,omitempty" gorm:"type:timestamp with time zone;"`
	ID          uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
}

type Report struct {
	ReportWeb
	ResolvedBy    uuid.UUID `json:"resolved_by,omitempty" gorm:"type:uuid;"`
	InternalNotes string    `json:"internal_notes,omitempty" gorm:"type:text;"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	CreatedBy     uuid.UUID      `json:"created_by" gorm:"type:uuid;not null"`
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
