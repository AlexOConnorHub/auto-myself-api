
package database

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
    "log"
)

var DB *sql.DB

func Init() {
    var err error
    connStr := "postgres://user:password@localhost:5432/yourdb?sslmode=disable"
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        log.Fatalf("Error opening database: %v", err)
    }

    if err = DB.Ping(); err != nil {
        log.Fatalf("Error connecting to database: %v", err)
    }

    fmt.Println("Connected to the database successfully.")
}
