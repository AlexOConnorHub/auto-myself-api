package server

import (
	"database/sql"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/joho/godotenv/autoload"

	"github.com/golang-migrate/migrate/v4"
	migrate_postgres "github.com/golang-migrate/migrate/v4/database/postgres"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func MigrateDB(tb testing.TB, db *sql.DB) {
	if tb != nil {
		tb.Helper()
	}

	schemaDriver, err := migrate_postgres.WithInstance(db, &migrate_postgres.Config{
		MigrationsTable: "_schema_migrations",
	})
	if err != nil {
		panic("failed to create schema driver: " + err.Error())
	}

	cwd := GetRelativeRootPath(tb)

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+cwd+"/migrations/schema",
		"postgres", schemaDriver)
	if err != nil {
		panic("failed to create migrate instance for schema: " + err.Error())
	}
	if err = m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			panic("failed to migrate schema: " + err.Error())
		}
	}
}

func ReseedDB(tb testing.TB, db *sql.DB) {
	if tb != nil {
		tb.Helper()
	}

	seedDriver, err := migrate_postgres.WithInstance(db, &migrate_postgres.Config{
		MigrationsTable: "_seed_migrations",
	})
	if err != nil {
		panic("failed to create seed driver: " + err.Error())
	}
	cwd := GetRelativeRootPath(tb)

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+cwd+"/migrations/seed",
		"postgres", seedDriver)
	if err != nil {
		panic("failed to create instance for seed: " + err.Error())
	}
	if err = m.Down(); err != nil {
		if err != migrate.ErrNoChange {
			panic("failed to seed down: " + err.Error())
		}
	}
	if err = m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			panic("failed to seed: " + err.Error())
		}
	}
}

func GetRelativeRootPath(tb testing.TB) string {
	if tb != nil {
		tb.Helper()
	}
	importPath := runGoList(tb, "list", "-f", "{{.ImportPath}}")
	modulePath := runGoList(tb, "list", "-m", "-f", "{{.Path}}")
	pkgPath := runGoList(tb, "list", "-f", "{{.Dir}}")

	relativePath, err := filepath.Rel(importPath, modulePath)
	if err != nil {
		panic("failed to get relative path: " + err.Error())
	}
	return filepath.Join(pkgPath, relativePath)
}

func runGoList(tb testing.TB, arg ...string) string {
	if tb != nil {
		tb.Helper()
	}
	cmd := exec.Command("go", arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic("runGoList: " + err.Error() + "\nOutput: " + string(output))
	}
	return strings.TrimSpace(string(output))
}
