package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
	gorm_postgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	mu   sync.RWMutex
	gorm *gorm.DB
	db   *sql.DB
}

func NewDatabase(db *sql.DB) (*Database, error) {
	gormDB, err := ConnectGormPostgres(db)
	if err != nil {
		return nil, err
	}
	return &Database{
		gorm: gormDB,
		db:   db,
	}, nil
}

func NewDatabaseFromGorm(gormDB *gorm.DB) *Database {
	return &Database{gorm: gormDB}
}

func (d *Database) GetDB() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

func (d *Database) GetGorm() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.gorm
}

func (d *Database) Ping(ctx context.Context) error {
	if d == nil {
		return errors.New("database is not configured")
	}

	d.mu.RLock()
	db, gormDB := d.db, d.gorm
	d.mu.RUnlock()

	if db != nil {
		return db.PingContext(ctx)
	}
	if gormDB == nil {
		return errors.New("database is not configured")
	}
	return gormDB.WithContext(ctx).Exec("SELECT 1").Error
}

func (d *Database) Replace(newDB *sql.DB) (*sql.DB, error) {
	newGorm, err := ConnectGormPostgres(newDB)
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	oldDB := d.db

	d.db = newDB
	d.gorm = newGorm

	return oldDB, nil
}

func (d *Database) Close() error {
	d.mu.RLock()
	db := d.db
	d.mu.RUnlock()
	if db == nil {
		return nil
	}
	return db.Close()
}

func ConnectPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("pinging postgres: %w (close failed: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return db, nil
}

func ConnectGormPostgres(db *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(gorm_postgres.New(gorm_postgres.Config{
		Conn: db,
	}), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}
	return gormDB, nil
}
