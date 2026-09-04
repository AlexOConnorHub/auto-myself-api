package app

import (
	"auto-myself-api/helpers"
	"database/sql"

	"gorm.io/gorm"
)

type App struct {
	Gorm    *gorm.DB
	DB      *sql.DB
	Secrets *helpers.SecretStore
}

func MakeApp() *App {
	secrets, err := helpers.SetupSecrets([]string{"DOMAIN", "KEYRING", "POSTGRES_DSN", "GOOGLE_OAUTH2_CLIENT_ID", "GOOGLE_OAUTH2_CLIENT_SECRET"})
	if err != nil {
		panic(err)
	}
	db := helpers.ConnectDB(secrets)
	gorm := helpers.ConnectGorm(db)

	return &App{
		Gorm:    gorm,
		DB:      db,
		Secrets: secrets,
	}
}
