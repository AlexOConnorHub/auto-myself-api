package database

import (
	"testing"
)

func TestConnection(t *testing.T) {
	var err error
	InitTest()

	err = sqlDB.Ping()
	if err != nil {
		t.Fatalf("Failed to connect to the test database: %v", err)
	}
}
