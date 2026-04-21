package helpers

import (
	"database/sql"
	"testing"

	gorm_postgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB() *sql.DB {
	return connect(GetSecret("POSTGRES_DSN"))
}

func TestConnectDB(tb testing.TB) *sql.DB {
	if tb != nil {
		tb.Helper()
	}

	return connect(GetSecret("POSTGRES_TEST_DSN"))
}

func connect(dsn string) *sql.DB {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic("failed to connect to the database: " + err.Error())
	}
	if err := db.Ping(); err != nil {
		panic("failed to ping the database: " + err.Error())
	}
	return db
}

func ConnectGorm(db *sql.DB) *gorm.DB {
	Gorm, err := gorm.Open(gorm_postgres.New(gorm_postgres.Config{
		Conn: db,
	}), &gorm.Config{})

	if err != nil {
		panic("failed to initialize gorm: " + err.Error())
	}
	return Gorm
}

func ConnectGormTest(db *sql.Tx) *gorm.DB {
	Gorm, err := gorm.Open(gorm_postgres.New(gorm_postgres.Config{
		Conn: db,
	}), &gorm.Config{})

	if err != nil {
		panic("failed to initialize gorm: " + err.Error())
	}
	return Gorm
}
