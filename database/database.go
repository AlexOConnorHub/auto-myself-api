package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"

	"cloud.google.com/go/storage"
	"github.com/golang-migrate/migrate/v4"
	migrate_postgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	gorm_postgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB() *sql.DB {
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	dbname := os.Getenv("POSTGRES_DB")

	return connect(host, user, pass, dbname, port)
}

func connect(host, user, pass, dbname, port string) *sql.DB {
	var err error

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s password=%s port=%s sslmode=disable",
		host, user, dbname, pass, port)
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

func ConnectGoogleClient() *storage.Client {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		panic("failed to create Google Cloud Storage client: " + err.Error())
	}
	return client
}

func DatabaseFetchError(c *gin.Context, err error, args ...any) {
	if err != gorm.ErrRecordNotFound {
		slog.Get(c).Error("Error fetching by ID", append([]any{"error", err}, args...)...)
		c.Status(http.StatusInternalServerError)
	} else {
		slog.Get(c).Info("Record not found for ID", args...)
		c.Status(http.StatusNotFound)
	}
}

func TestConnectDB(tb testing.TB) *sql.DB {
	if tb != nil {
		tb.Helper()
	}

	user := os.Getenv("POSTGRES_TEST_USER")
	pass := os.Getenv("POSTGRES_TEST_PASSWORD")
	host := os.Getenv("POSTGRES_TEST_HOST")
	port := os.Getenv("POSTGRES_TEST_PORT")
	dbname := os.Getenv("POSTGRES_TEST_DB")

	return connect(host, user, pass, dbname, port)
}

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
		panic("failed to create migrate instance for seed: " + err.Error())
	}
	if err = m.Down(); err != nil {
		if err != migrate.ErrNoChange {
			panic("failed to migrate seed down: " + err.Error())
		}
	}
	if err = m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			panic("failed to migrate seed: " + err.Error())
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
