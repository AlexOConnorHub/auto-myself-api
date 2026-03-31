package app

import (
	"database/sql"

	"cloud.google.com/go/storage"
	"gorm.io/gorm"
)

type App struct {
	Gorm    *gorm.DB
	DB      *sql.DB
	Gclient *storage.Client
}
