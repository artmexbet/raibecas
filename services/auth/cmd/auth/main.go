package main

import (
	"log"
	"log/slog"

	"github.com/artmexbet/raibecas/services/auth/internal/config"
	"github.com/artmexbet/raibecas/services/auth/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "err", err)
	}

	// CreateUser NATS-based server
	srv, err := server.NewNATS(cfg)
	if err != nil {
		slog.Error("Failed to create server", "err", err)
	}

	// Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
