package models

import (
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
