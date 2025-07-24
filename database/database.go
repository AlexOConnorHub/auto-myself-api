package database

import (
	"database/sql"
	"fmt"
	"os"
	"runtime"

	"gorm.io/driver/postgres"
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

	DB, err = gorm.Open(postgres.New(postgres.Config{
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
