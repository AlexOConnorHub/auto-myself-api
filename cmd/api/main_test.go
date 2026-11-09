package main

import (
	"auto-myself-api/internal/server"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

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

func TestHTTPServerTimeouts(t *testing.T) {
	srv := newHTTPServer(":0", http.NewServeMux())

	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %s, want 10s", srv.ReadHeaderTimeout)
	}
	if srv.IdleTimeout != 2*time.Minute {
		t.Errorf("IdleTimeout = %s, want 2m", srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != 1<<20 {
		t.Errorf("MaxHeaderBytes = %d, want 1 MiB", srv.MaxHeaderBytes)
	}
	if srv.ReadTimeout <= server.UploadRequestTimeout {
		t.Errorf("ReadTimeout = %s must exceed the upload handler timeout %s", srv.ReadTimeout, server.UploadRequestTimeout)
	}
	if srv.WriteTimeout <= server.UploadRequestTimeout {
		t.Errorf("WriteTimeout = %s must exceed the upload handler timeout %s", srv.WriteTimeout, server.UploadRequestTimeout)
	}
	if shutdownGrace <= server.UploadRequestTimeout {
		t.Errorf("shutdownGrace = %s must exceed the upload handler timeout %s", shutdownGrace, server.UploadRequestTimeout)
	}
}
