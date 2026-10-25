package server

import (
	"auto-myself-api/internal/secrets"
	"database/sql"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type App struct {
	Gorm    *gorm.DB
	DB      *sql.DB
	Secrets *secrets.SecretStore
}

func WithApp(a *App, h func(*gin.Context, *App)) gin.HandlerFunc {
	return func(c *gin.Context) {
		h(c, a)
	}
}

func MakeApp() *App {
	secretsStore, err := secrets.SetupSecrets([]string{"DOMAIN", "KEYRING", "POSTGRES_DSN", "GOOGLE_OAUTH2_CLIENT_ID", "GOOGLE_OAUTH2_CLIENT_SECRET"})
	if err != nil {
		panic(err)
	}
	db := ConnectDB(secretsStore)
	gorm := ConnectGorm(db)

	return &App{
		Gorm:    gorm,
		DB:      db,
		Secrets: secretsStore,
	}
}

func SetupTest(t *testing.T) (*gin.Engine, *App) {
	t.Helper()

	db := TestConnectDB(t)
	MigrateDB(t, db)
	ReseedDB(t, db)

	tx, err := db.Begin()
	if err != nil {
		panic("failed to begin transaction: " + err.Error())
	}

	app := App{
		Gorm: ConnectGormTest(tx),
		DB:   db,
	}

	gin.SetMode(gin.TestMode)
	r := MakeGin()
	SetupRoutes(r, &app)

	t.Cleanup(func() {
		tx.Rollback()
		app.DB.Close()
	})
	fmt.Println("Returning")
	return r, &app
}
