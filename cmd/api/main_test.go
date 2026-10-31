package main

import (
	"log"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("No .env file found")
	}

	os.Exit(m.Run())
}

func TestConfigureGinModeDefaultsToRelease(t *testing.T) {
	t.Setenv(gin.EnvGinMode, "")
	gin.SetMode(gin.DebugMode) // Simulate Gin's default when GIN_MODE is unset.
	t.Cleanup(func() { gin.SetMode(gin.TestMode) })

	configureGinMode()

	if mode := gin.Mode(); mode != gin.ReleaseMode {
		t.Errorf("expected release mode when GIN_MODE is unset, got %q", mode)
	}
}

func TestConfigureGinModeHonorsExplicitMode(t *testing.T) {
	t.Setenv(gin.EnvGinMode, gin.DebugMode)
	gin.SetMode(gin.DebugMode)
	t.Cleanup(func() { gin.SetMode(gin.TestMode) })

	configureGinMode()

	if mode := gin.Mode(); mode != gin.DebugMode {
		t.Errorf("expected explicit debug mode to be preserved, got %q", mode)
	}
}
