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

	dsn := GetSecret("POSTGRES_TEST_DSN")

	return connect(dsn)
}

func connect(dsn string) *sql.DB {
	var err error

	// dsn := fmt.Sprintf("host=%s user=%s dbname=%s password=%s port=%s sslmode=disable",
	// 	host, user, dbname, pass, port)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic("failed to connect to the database: " + err.Error())
	}
	if err = db.Ping(); err != nil {
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
