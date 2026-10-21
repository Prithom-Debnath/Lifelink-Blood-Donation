package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lifelink/internal/config"
	"lifelink/internal/database"
	"lifelink/internal/routes"
)

func main() {
	// 1. Load application configuration from environment (.env)
	cfg := config.Load()

	// 2. Connect to MongoDB with reachability check (Ping)
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Database startup failure: %v", err)
	}

	// 3. Initialize HTTP router
	router := routes.SetupRouter(cfg)

	// 4. Configure the HTTP server
	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// 5. Start the server in a separate background goroutine so it doesn't block shutdown handling
	go func() {
		log.Printf("LifeLink Blood Donation System server listening on port %s (env: %s)...", cfg.Port, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server encountered an unexpected error: %v", err)
		}
	}()

	// 6. Graceful shutdown: wait for OS interrupt signal (Ctrl+C / SIGINT / SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received termination signal (%s). Initiating clean shutdown...", sig)

	// Context with timeout to give ongoing HTTP requests up to 5 seconds to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server forced to terminate: %v", err)
	} else {
		log.Println("HTTP server stopped accepting new requests.")
	}

	// 7. Close MongoDB connection cleanly
	if err := db.Close(); err != nil {
		log.Printf("Error during MongoDB disconnection: %v", err)
	}

	log.Println("LifeLink server exited cleanly.")
}
