//go:build test

package database

import (
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func InitTest() {
	user := os.Getenv("POSTGRES_TEST_USER")
	pass := os.Getenv("POSTGRES_TEST_PASSWORD")
	host := os.Getenv("POSTGRES_TEST_HOST")
	port := os.Getenv("POSTGRES_TEST_PORT")
	dbname := os.Getenv("POSTGRES_TEST_DB")

	connect(host, user, pass, dbname, port)

	// Use migrate/migrate to migrate to the latest version.
	// Then, down/up the schema migrations to ensure the test database is clean.
	schema_driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		MigrationsTable: "_schema_migrations",
	})
	if err != nil {
		LogError(err)
		panic("failed to create schema driver: " + err.Error())
	}

	seed_driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		MigrationsTable: "_seed_migrations",
	})
	if err != nil {
		LogError(err)
		panic("failed to create seed driver: " + err.Error())
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///app/migrations/schema",
		"postgres", schema_driver)
	if err != nil {
		LogError(err)
		panic("failed to create migrate instance for schema: " + err.Error())
	}
	m.Up()

	m, err = migrate.NewWithDatabaseInstance(
		"file:///app/migrations/seed",
		"postgres", seed_driver)
	if err != nil {
		LogError(err)
		panic("failed to create migrate instance for seed: " + err.Error())
	}
	m.Down()
	m.Up()
}
