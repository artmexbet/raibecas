package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/ingestion"
	"github.com/artmexbet/raibecas/services/index/internal/neuro"
	"github.com/artmexbet/raibecas/services/index/internal/pipeline"
	"github.com/artmexbet/raibecas/services/index/internal/qdrant"
)

type App struct{}

func New() *App { return &App{} }

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	slog.Info("config loaded", "cfg", cfg)

	qClient, err := qdrant.New(&cfg.Qdrant)
	if err != nil {
		return fmt.Errorf("qdrant client: %w", err)
	}
	if err := qClient.EnsureCollection(ctx); err != nil {
		return fmt.Errorf("ensure collection: %w", err)
	}

	writer := pipeline.NewQdrantWriter(qClient)

	ollama, err := neuro.NewConnector(&cfg.Ollama)
	if err != nil {
		return fmt.Errorf("ollama connector: %w", err)
	}

	pipe := pipeline.New(cfg, ollama, writer)

	// TODO: wire NATS ingestion depending on cfg.UseNATS

	if cfg.UseHTTP {
		httpIngestor := ingestion.NewHTTPIngestor(pipe)
		go func() {
			if err := httpIngestor.Start(cfg.HTTP.Address()); err != nil {
				slog.Error("http server", "err", err)
				stop()
			}
		}()
	}

	<-ctx.Done()
	slog.Info("shutdown signal received")
	err = qClient.Close()
	if err != nil {
		slog.Error("close qdrant client", "err", err)
	}
	slog.Info("qdrant client closed")

	slog.Info("shutdown completed")
	return nil
}
