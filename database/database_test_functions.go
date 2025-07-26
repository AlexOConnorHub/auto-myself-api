package database

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func getRelativeRootPath(tb testing.TB) string {
	tb.Helper()

	importPath := runGoList(tb, "list", "-f", "{{.ImportPath}}")
	modulePath := runGoList(tb, "list", "-m", "-f", "{{.Path}}")
	pkgPath := runGoList(tb, "list", "-f", "{{.Dir}}")

	relativePath, err := filepath.Rel(importPath, modulePath)
	if err != nil {
		tb.Fatal(err)
	}
	return filepath.Join(pkgPath, relativePath)
}

func runGoList(tb testing.TB, arg ...string) string {
	tb.Helper()
	cmd := exec.Command("go", arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("runGoList: %v: %s", err, string(output))
	}
	return strings.TrimSpace(string(output))
}

func InitTest(tb testing.TB) {
	tb.Helper()
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

	// _, file, _, _ := runtime.Caller(0)
	cwd := getRelativeRootPath(tb)

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+cwd+"/migrations/schema",
		"postgres", schema_driver)
	if err != nil {
		LogError(err)
		panic("failed to create migrate instance for schema: " + err.Error())
	}
	m.Up()

	m, err = migrate.NewWithDatabaseInstance(
		"file://"+cwd+"/migrations/seed",
		"postgres", seed_driver)
	if err != nil {
		LogError(err)
		panic("failed to create migrate instance for seed: " + err.Error())
	}
	m.Down()
	m.Up()
}
