package main

import (
	"log"

	"auth/internal/config"
	"auth/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Starting Auth service with NATS...")

	// Create NATS-based server
	srv, err := server.NewNATS(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
