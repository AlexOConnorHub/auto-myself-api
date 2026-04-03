package main

import (
	"auto-myself-api/controllers"
	"auto-myself-api/helpers"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	a := helpers.MakeApp()
	r := helpers.MakeGin()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	controllers.SetupRoutes(r, a)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r.Handler(),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Server Shutdown:", err)
	}
	log.Println("Server exiting")

}
