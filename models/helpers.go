package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type DatabaseMetadata struct {
	ID        uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;not null"`
	CreatedAt time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *gorm.DeletedAt `json:"deleted_at" gorm:"index" swaggertype:"primitive,string"`
}

func ParseUUID(uuidStr string) (uuid.UUID, error) {
	parsedUUID, err := uuid.FromString(uuidStr)
	if err != nil {
		return uuid.Nil, err
	}
	return parsedUUID, nil
}

func BeforeCreateSetupDatabaseMetadata(d *DatabaseMetadata) (err error) {
	d.ID, err = uuid.NewV7()
	if err != nil {
		return err
	}
	return nil
}
