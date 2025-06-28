package models

import (
	"github.com/gofrs/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=postgres user=postgres password=password dbname=appdb port=5423 sslmode=disable TimeZone=US/Eastern",
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{})

	if err != nil {
		panic("Failed to connect to database!")
	}

	DB = db
}

type DatabaseMetadata struct {
	gorm.Model
	ID uuid.UUID `json:"ID" gorm:"type:uuid;primaryKey;not null"`
}

func ParseUUID(uuidStr string) (uuid.UUID, error) {
	parsedUUID, err := uuid.FromString(uuidStr)
	if err != nil {
		return uuid.Nil, err
	}
	return parsedUUID, nil
}
