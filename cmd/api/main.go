package main

import (
	"auto-myself-api/internal/server"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = server.UploadRequestTimeout + time.Minute
	writeTimeout      = server.UploadRequestTimeout + time.Minute
	idleTimeout       = 2 * time.Minute
	maxHeaderBytes    = 1 << 20
	shutdownGrace     = server.UploadRequestTimeout + time.Minute
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configureGinMode()

	engine := server.NewEngine()

	shutdownApp, err := Setup(ctx, engine)
	if err != nil {
		log.Fatalf("setup failed: %s", err)
	}
	defer shutdownApp()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := newHTTPServer(":"+port, engine.Handler())

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown Server ...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server Shutdown: %s", err)
	}
	log.Println("Server exiting")
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

func configureGinMode() {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode) // Override default of debug mode
	}
}
