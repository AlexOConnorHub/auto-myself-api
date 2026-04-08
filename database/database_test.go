package database

import (
	"auto-myself-api/helpers"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("No .env file found")
	}

	myEnv, err := godotenv.Read("../.env")
	if err != nil {
		log.Println("No .env file found")
	}

	for key, value := range myEnv {
		fmt.Printf("%s = %s\n", key, value)
		os.Setenv(key, value)
	}

	os.Exit(m.Run())
}

func TestConnection(t *testing.T) {
	var err error
	db := helpers.TestConnectDB(t)

	err = db.Ping()
	if err != nil {
		t.Fatalf("Failed to connect to the test database: %v", err)
	}
}
