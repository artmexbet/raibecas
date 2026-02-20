package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/artmexbet/raibecas/libs/natsw"
	"github.com/artmexbet/raibecas/libs/telemetry"
	"github.com/nats-io/nats.go"
	"github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	natshandler "github.com/artmexbet/raibecas/services/chat/internal/handler/nats"
	"github.com/artmexbet/raibecas/services/chat/internal/neuro"
	qdrantWrapper "github.com/artmexbet/raibecas/services/chat/internal/qdrant-wrapper"
	_redis "github.com/artmexbet/raibecas/services/chat/internal/redis"
	"github.com/artmexbet/raibecas/services/chat/internal/service"
)

// App represents the main entry point for the chat service application.
type App struct {
	cfg            *config.Config
	natsConn       *nats.Conn
	natsClient     *natsw.Client
	qdrantClient   *qdrant.Client
	redisClient    *redis.Client
	tracerProvider *trace.TracerProvider
	svc            *service.Chat
	natsHandler    *natshandler.Handler
}

// New creates and returns a new App instance with all dependencies initialized.
func New() (*App, error) {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	slog.Info("configuration loaded", "qdrant_host", cfg.Qdrant.Host, "nats_url", cfg.NATS.URL)

	// Initialize tracer
	tp, err := telemetry.InitTracer(telemetry.TracerConfig{
		ServiceName:    "chat",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "localhost:4317",
		Enabled:        true,
		ExportTimeout:  30 * time.Second,
		BatchTimeout:   5 * time.Second,
		MaxQueueSize:   2048,
		MaxExportBatch: 512,
	})
	if err != nil {
		slog.Warn("failed to initialize tracer, continuing without tracing", "error", err)
		// Continue without tracer rather than failing startup
	}

	// Connect to NATS
	natsConn, err := nats.Connect(
		cfg.NATS.URL,
		nats.MaxReconnects(cfg.NATS.MaxReconnects),
		nats.ReconnectWait(cfg.NATS.ReconnectWait),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	slog.Info("connected to NATS", "url", cfg.NATS.URL)

	// Create NATS wrapper client
	natsClient := natsw.NewClient(
		natsConn,
		natsw.WithRecover(),
		natsw.WithLogger(slog.Default()),
	)

	// Initialize Qdrant client
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.Qdrant.Host,
		Port: cfg.Qdrant.Port,
	})
	if err != nil {
		natsConn.Close()
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := qdrantClient.HealthCheck(ctx); err != nil {
		natsConn.Close()
		qdrantClient.Close()
		return nil, fmt.Errorf("failed to connect to qdrant: %w", err)
	}
	slog.Info("connected to qdrant")

	qdrantWrap := qdrantWrapper.New(&cfg.Qdrant, qdrantClient)
	if err := qdrantWrap.CheckConnection(ctx); err != nil {
		natsConn.Close()
		qdrantClient.Close()
		return nil, fmt.Errorf("failed to check qdrant connection: %w", err)
	}

	// Initialize Ollama connector
	ollama, err := neuro.NewConnector(&cfg.Ollama)
	if err != nil {
		natsConn.Close()
		qdrantClient.Close()
		return nil, fmt.Errorf("failed to create ollama connector: %w", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.GetAddress(), DB: cfg.Redis.DB})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		natsConn.Close()
		qdrantClient.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	slog.Info("connected to redis")

	historyStore := _redis.New(&cfg.Redis, redisClient)

	// Create service
	svc := service.New(qdrantWrap, ollama, historyStore)

	// Create NATS handler
	natsHandler := natshandler.NewHandler(natsClient, svc)
	if err := natsHandler.Subscribe(); err != nil {
		natsConn.Close()
		qdrantClient.Close()
		redisClient.Close()
		return nil, fmt.Errorf("failed to subscribe to NATS: %w", err)
	}
	slog.Info("NATS handler subscribed")

	return &App{
		cfg:            cfg,
		natsConn:       natsConn,
		natsClient:     natsClient,
		qdrantClient:   qdrantClient,
		redisClient:    redisClient,
		tracerProvider: tp,
		svc:            svc,
		natsHandler:    natsHandler,
	}, nil
}

// Run starts the application and blocks until shutdown signal is received.
func (a *App) Run() error {
	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	slog.Info("chat service started, waiting for messages...")

	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.InfoContext(shutdownCtx, "shutting down the service...")

	// Shutdown tracer provider (flush all pending spans)
	if a.tracerProvider != nil {
		if err := telemetry.Shutdown(shutdownCtx, a.tracerProvider); err != nil {
			slog.ErrorContext(shutdownCtx, "tracer shutdown error", "error", err)
		}
	}

	// Close Redis client
	if err := a.redisClient.Close(); err != nil {
		slog.ErrorContext(shutdownCtx, "redis client shutdown error", "error", err)
	}

	// Close Qdrant client
	if err := a.qdrantClient.Close(); err != nil {
		slog.ErrorContext(shutdownCtx, "qdrant client shutdown error", "error", err)
	}

	// Close NATS connection
	a.natsConn.Close()
	slog.InfoContext(shutdownCtx, "NATS connection closed")

	slog.InfoContext(shutdownCtx, "service stopped gracefully")
	return nil
}
