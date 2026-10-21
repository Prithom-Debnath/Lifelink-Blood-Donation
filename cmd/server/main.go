package main

import (
	"log"

	"lifelink/internal/config"
	"lifelink/internal/routes"
)

func main() {
	// 1. Load application configuration from environment
	cfg := config.Load()

	// 2. Initialize router with configured routes and middleware
	router := routes.SetupRouter(cfg)

	// 3. Start the HTTP server
	addr := ":" + cfg.Port
	log.Printf("LifeLink Blood Donation System server starting on port %s (env: %s)...", cfg.Port, cfg.AppEnv)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Server failed to start on %s: %v", addr, err)
	}
}
