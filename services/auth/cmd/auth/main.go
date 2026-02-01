package main

import (
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

	// CreateUser App-based server
	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("Failed to create server", "err", err)
	}

	// Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		slog.Error("Server error", "err", err)
	}
}
