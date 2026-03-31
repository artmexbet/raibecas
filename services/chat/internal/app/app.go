package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/artmexbet/raibecas/libs/natsw"
	"github.com/artmexbet/raibecas/libs/telemetry"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	httphandler "github.com/artmexbet/raibecas/services/chat/internal/handler/http"
	natshandler "github.com/artmexbet/raibecas/services/chat/internal/handler/nats"
	"github.com/artmexbet/raibecas/services/chat/internal/neuro"
	"github.com/artmexbet/raibecas/services/chat/internal/postgres"
	qdrantWrapper "github.com/artmexbet/raibecas/services/chat/internal/qdrant-wrapper"
	"github.com/artmexbet/raibecas/services/chat/internal/service"
	"github.com/artmexbet/raibecas/services/chat/migrations"
)

// App represents the main entry point for the chat service application.
type App struct {
	cfg            *config.Config
	natsConn       *nats.Conn
	natsClient     *natsw.Client
	qdrantClient   *qdrant.Client
	redisClient    *redis.Client
	pgStore        *postgres.Store
	tracerProvider *trace.TracerProvider
	svc            *service.Chat
	natsHandler    *natshandler.Handler
	httpHandler    *httphandler.Handler
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

	if err := migrations.Up(cfg.Database.GetDSN()); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

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
		Host:   cfg.Qdrant.Host,
		Port:   cfg.Qdrant.Port,
		UseTLS: false,
	})
	if err != nil {
		natsConn.Close()
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if _, err := qdrantClient.HealthCheck(ctx); err != nil {
		natsConn.Close()
		qdrantClient.Close() //nolint:errcheck // safe to ignore error on close during setup failure
		return nil, fmt.Errorf("failed to connect to qdrant: %w", err)
	}
	slog.Info("connected to qdrant")

	qdrantWrap := qdrantWrapper.New(&cfg.Qdrant, qdrantClient)
	if err := qdrantWrap.CheckConnection(ctx); err != nil {
		natsConn.Close()
		qdrantClient.Close() //nolint:errcheck // safe to ignore error on close during setup failure
		return nil, fmt.Errorf("failed to check qdrant connection: %w", err)
	}

	// Initialize Ollama connector
	ollama, err := neuro.NewConnector(&cfg.Ollama)
	if err != nil {
		natsConn.Close()
		qdrantClient.Close() //nolint:errcheck // safe to ignore error on close during setup failure
		return nil, fmt.Errorf("failed to create ollama connector: %w", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.GetAddress(), DB: cfg.Redis.DB})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		natsConn.Close()
		qdrantClient.Close() //nolint:errcheck // safe to ignore error on close during setup failure
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	slog.Info("connected to redis")

	// Initialize PostgreSQL store for chat history
	pgStore, err := postgres.New(ctx, &cfg.Database)
	if err != nil {
		natsConn.Close()
		qdrantClient.Close() //nolint:errcheck // safe to ignore error on close during setup failure
		redisClient.Close()  //nolint:errcheck // safe to ignore error on close during setup failure
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	slog.Info("connected to postgres (chat history)")

	// Create service
	svc := service.New(qdrantWrap, ollama, pgStore)

	// Create NATS handler
	natsHandler := natshandler.NewHandler(natsClient, svc)
	if err := natsHandler.Subscribe(); err != nil {
		natsConn.Close()
		qdrantClient.Close() //nolint:errcheck // safe to ignore error on close during setup failure
		redisClient.Close()  //nolint:errcheck // safe to ignore error on close during setup failure
		return nil, fmt.Errorf("failed to subscribe to NATS: %w", err)
	}
	slog.Info("NATS handler subscribed")

	// Create HTTP handler (always, port driven by config)
	httpHandler := httphandler.New(&cfg.HTTP, svc)
	httpHandler.RegisterRoutes()

	return &App{
		cfg:            cfg,
		natsConn:       natsConn,
		natsClient:     natsClient,
		qdrantClient:   qdrantClient,
		redisClient:    redisClient,
		pgStore:        pgStore,
		tracerProvider: tp,
		svc:            svc,
		natsHandler:    natsHandler,
		httpHandler:    httpHandler,
	}, nil
}

// Run starts the application and blocks until shutdown signal is received.
func (a *App) Run() error {
	// Start HTTP server in a goroutine
	go func() {
		slog.Info("starting HTTP server", "address", a.cfg.HTTP.GetAddress())
		if err := a.httpHandler.Run(); err != nil {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	slog.Info("chat service started, waiting for messages...")

	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.InfoContext(shutdownCtx, "shutting down the service...")

	// Shutdown HTTP server gracefully
	if err := a.httpHandler.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(shutdownCtx, "HTTP server shutdown error", "error", err)
	}

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

	// Close PostgreSQL store
	a.pgStore.Close()
	slog.InfoContext(shutdownCtx, "postgres connection closed")

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
