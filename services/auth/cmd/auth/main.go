package main

import (
	"log/slog"
	"os"

	"github.com/artmexbet/raibecas/services/auth/internal/config"
	"github.com/artmexbet/raibecas/services/auth/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	// Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
