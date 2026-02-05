package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/artmexbet/raibecas/services/documents/internal/app"
)

func main() {
	ctx := context.Background()

	// Create application
	application, err := app.New(ctx)
	if err != nil {
		slog.Error("failed to create application", "error", err)
		os.Exit(1)
	}

	// Run application
	if err := application.Run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}
