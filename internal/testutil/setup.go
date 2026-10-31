package testutil

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/server"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	migrate_postgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// Env is a per-test environment: a router wired to a transaction that is rolled back afterwards.
type Env struct {
	Router  *gin.Engine
	DB      *database.Database
	Storage *FakeStorage
}

var (
	errNoTestDSN = errors.New("POSTGRES_TEST_DSN not set")

	envOnce sync.Once

	dbOnce   sync.Once
	dbErr    error
	baseGorm *gorm.DB
)

// SetupTest builds the production router over a single transaction on the test database. The
// schema and seed are applied once per test process; every test rolls its changes back.
func SetupTest(t *testing.T) *Env {
	t.Helper()

	loadEnv()

	base, err := baseDatabase()
	if errors.Is(err, errNoTestDSN) {
		t.Skip("POSTGRES_TEST_DSN not set")
	}
	if err != nil {
		t.Fatalf("setting up test database: %v", err)
	}

	tx := base.Begin()
	if tx.Error != nil {
		t.Fatalf("beginning test transaction: %v", tx.Error)
	}
	t.Cleanup(func() {
		if err := tx.Rollback().Error; err != nil && !errors.Is(err, gorm.ErrInvalidTransaction) {
			t.Logf("rolling back test transaction: %v", err)
		}
	})

	secretStore, err := secrets.NewEnvStore([]string{"JWT_SIGNING_SECRET"})
	if err != nil {
		t.Fatalf("JWT_SIGNING_SECRET must be set for tests: %v", err)
	}

	storage := NewFakeStorage()

	gin.SetMode(gin.TestMode)
	router := server.NewEngine()
	deps := server.Dependencies{
		Database: database.NewDatabaseFromGorm(tx),
		Secrets:  secretStore,
		Storage:  storage,
	}
	server.RegisterRoutes(router, server.Build(deps), deps)

	return &Env{
		Router:  router,
		DB:      deps.Database,
		Storage: storage,
	}
}

func loadEnv() {
	envOnce.Do(func() {
		root := findRepoRoot()
		if root == "" {
			return
		}
		// A missing .env is fine when the environment is provided externally.
		_ = godotenv.Load(filepath.Join(root, ".env"))
	})
}

func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func baseDatabase() (*gorm.DB, error) {
	dbOnce.Do(func() {
		dsn := os.Getenv("POSTGRES_TEST_DSN")
		if dsn == "" {
			dbErr = errNoTestDSN
			return
		}

		root := findRepoRoot()
		if root == "" {
			dbErr = errors.New("could not locate repository root")
			return
		}

		ctx := context.Background()
		migrationDB, err := database.ConnectPostgres(ctx, dsn)
		if err != nil {
			dbErr = err
			return
		}
		defer migrationDB.Close()

		if err := migrateSchema(migrationDB, root); err != nil {
			dbErr = err
			return
		}
		if err := reseed(migrationDB, root); err != nil {
			dbErr = err
			return
		}

		baseSQL, err := database.ConnectPostgres(ctx, dsn)
		if err != nil {
			dbErr = err
			return
		}
		baseGorm, dbErr = database.ConnectGormPostgres(baseSQL)
	})

	return baseGorm, dbErr
}

func migrateSchema(db *sql.DB, root string) error {
	driver, err := migrate_postgres.WithInstance(db, &migrate_postgres.Config{
		MigrationsTable: "_schema_migrations",
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+filepath.Join(root, "migrations", "schema"), "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func reseed(db *sql.DB, root string) error {
	driver, err := migrate_postgres.WithInstance(db, &migrate_postgres.Config{
		MigrationsTable: "_seed_migrations",
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+filepath.Join(root, "migrations", "seed"), "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
