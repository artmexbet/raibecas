package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/ingestion"
	"github.com/artmexbet/raibecas/services/index/internal/neuro"
	"github.com/artmexbet/raibecas/services/index/internal/pipeline"
	"github.com/artmexbet/raibecas/services/index/internal/qdrant"
	"github.com/artmexbet/raibecas/services/index/internal/storage"
)

type App struct{}

func New() *App {
	return &App{}
}

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	slog.Info("config loaded", "cfg", cfg)

	// Инициализируем storage
	store, err := storage.NewFileSystem(cfg.Storage.BaseDir)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	slog.Info("storage initialized", "base_dir", cfg.Storage.BaseDir)

	// Инициализируем Qdrant
	qClient, err := qdrant.New(&cfg.Qdrant)
	if err != nil {
		return fmt.Errorf("qdrant client: %w", err)
	}
	defer func() {
		if err := qClient.Close(); err != nil {
			slog.Error("close qdrant client", "err", err)
		} else {
			slog.Info("qdrant client closed")
		}
	}()

	if err := qClient.EnsureCollection(ctx); err != nil {
		return fmt.Errorf("ensure collection: %w", err)
	}

	// Инициализируем Ollama
	ollama, err := neuro.NewConnector(&cfg.Ollama)
	if err != nil {
		return fmt.Errorf("ollama client: %w", err)
	}
	slog.Info("ollama client initialized", "address", cfg.Ollama.Address())

	// Создаем writer для векторной БД
	writer := pipeline.NewQdrantWriter(qClient)

	// Создаем pipeline с storage reader
	pipe := pipeline.New(cfg, ollama, writer, store)

	// Запускаем NATS consumer если включен
	if cfg.UseNATS {
		nc, err := nats.Connect(cfg.NATS.URL)
		if err != nil {
			return fmt.Errorf("nats connect: %w", err)
		}
		defer nc.Close()

		consumer, err := ingestion.NewConsumer(&cfg.NATS, nc, pipe)
		if err != nil {
			return fmt.Errorf("create consumer: %w", err)
		}

		go func() {
			if err := consumer.Start(ctx); err != nil {
				slog.Error("nats consumer", "err", err)
				stop()
			}
		}()
		slog.Info("NATS consumer started", "subject", cfg.NATS.Subject)
	}

	// Запускаем HTTP server
	httpIngestor := ingestion.NewHTTPIngestor(pipe, store)
	go func() {
		if err := httpIngestor.Start(cfg.HTTP.Address()); err != nil {
			slog.Error("http server", "err", err)
			stop()
		}
	}()
	slog.Info("HTTP server started", "address", cfg.HTTP.Address())

	// Ожидаем сигнала завершения
	<-ctx.Done()
	slog.Info("shutdown signal received")

	// Graceful shutdown
	if err := httpIngestor.Shutdown(); err != nil {
		slog.Error("shutdown http server", "err", err)
	}

	slog.Info("shutdown completed")
	return nil
}
