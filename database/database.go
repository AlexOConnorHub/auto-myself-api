package database

import (
	"auto-myself-api/helpers"
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	migrate_postgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	gorm_postgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var sqlDB *sql.DB

func Init() {
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	dbname := os.Getenv("POSTGRES_DB")

	connect(host, user, pass, dbname, port)
}

func InitTest(tb testing.TB) {
	if tb != nil {
		tb.Helper()
	}
	user := os.Getenv("POSTGRES_TEST_USER")
	pass := os.Getenv("POSTGRES_TEST_PASSWORD")
	host := os.Getenv("POSTGRES_TEST_HOST")
	port := os.Getenv("POSTGRES_TEST_PORT")
	dbname := os.Getenv("POSTGRES_TEST_DB")

	connect(host, user, pass, dbname, port)

	// Use migrate/migrate to migrate to the latest version.
	// Then, down/up the schema migrations to ensure the test database is clean.
	schema_driver, err := migrate_postgres.WithInstance(sqlDB, &migrate_postgres.Config{
		MigrationsTable: "_schema_migrations",
	})
	if err != nil {
		LogError(err)
		panic("failed to create schema driver: " + err.Error())
	}
	cwd := helpers.GetRelativeRootPath(tb)

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+cwd+"/migrations/schema",
		"postgres", schema_driver)
	if err != nil {
		LogError(err)
		panic("failed to create migrate instance for schema: " + err.Error())
	}
	if err = m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			LogError(err)
			panic("failed to migrate schema: " + err.Error())
		}
	}

	seed_driver, err := migrate_postgres.WithInstance(sqlDB, &migrate_postgres.Config{
		MigrationsTable: "_seed_migrations",
	})
	if err != nil {
		LogError(err)
		panic("failed to create seed driver: " + err.Error())
	}

	m, err = migrate.NewWithDatabaseInstance(
		"file://"+cwd+"/migrations/seed",
		"postgres", seed_driver)
	if err != nil {
		LogError(err)
		panic("failed to create migrate instance for seed: " + err.Error())
	}
	if err = m.Down(); err != nil {
		if err != migrate.ErrNoChange {
			LogError(err)
			panic("failed to migrate seed down: " + err.Error())
		}
	}
	if err = m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			LogError(err)
			panic("failed to migrate seed: " + err.Error())
		}
	}
}

func connect(host, user, pass, dbname, port string) {
	var err error

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s password=%s port=%s sslmode=disable",
		host, user, dbname, pass, port)
	sqlDB, err = sql.Open("pgx", dsn)
	if err != nil {
		LogError(err)
		panic("failed to connect to the database: " + err.Error())
	}

	DB, err = gorm.Open(gorm_postgres.New(gorm_postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})

	if err != nil {
		LogError(err)
		panic("failed to initialize gorm: " + err.Error())
	}
}

func LogError(err error) {
	_, file, line, _ := runtime.Caller(1)
	fmt.Printf("%s:%d: Database error: %s\n", file, line, err.Error())
}
