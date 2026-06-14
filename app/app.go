package app

import (
	"database/sql"

	"gorm.io/gorm"
)

type App struct {
	Gorm *gorm.DB
	DB   *sql.DB
}
