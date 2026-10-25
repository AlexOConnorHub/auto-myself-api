package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

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

func (d *Database) SecretChange()

func ConnectPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		err_2 := db.Close()
		if err_2 != nil {
			return nil, fmt.Errorf("ping failed: %v, close failed: %v", err, err_2)
		}
		return nil, err
	}
	return db, nil
}

func ConnectGormPostgres(db *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(gorm_postgres.New(gorm_postgres.Config{
		Conn: db,
	}), &gorm.Config{})

	if err != nil {
		return nil, err
	}
	return gormDB, nil
}
