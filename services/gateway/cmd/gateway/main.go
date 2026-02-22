package main

import (
	"log/slog"
	"os"

	"github.com/artmexbet/raibecas/services/gateway/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	if err := a.Run(); err != nil {
		slog.Error("app error", "error", err)
		os.Exit(1)
	}
}
