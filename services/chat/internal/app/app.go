package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/artmexbet/raibecas/libs/telemetry"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/handler/http"
	"github.com/artmexbet/raibecas/services/chat/internal/neuro"
	qdrantWrapper "github.com/artmexbet/raibecas/services/chat/internal/qdrant-wrapper"
	_redis "github.com/artmexbet/raibecas/services/chat/internal/redis"
	"github.com/artmexbet/raibecas/services/chat/internal/service"
)

// App represents the main entry point for the chat service application.
type App struct {
	cfg            *config.Config
	qdrantClient   *qdrant.Client
	redisClient    *redis.Client
	tracerProvider *trace.TracerProvider
	svc            *service.Chat
	api            *http.Handler
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
	slog.Info("configuration loaded", "qdrant_host", cfg.Qdrant.Host, "http_port", cfg.HTTP.Port)

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

	// Initialize Qdrant client
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.Qdrant.Host,
		Port: cfg.Qdrant.Port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := qdrantClient.HealthCheck(ctx); err != nil {
		qdrantClient.Close()
		return nil, fmt.Errorf("failed to connect to qdrant: %w", err)
	}
	slog.Info("connected to qdrant")

	qdrantWrap := qdrantWrapper.New(&cfg.Qdrant, qdrantClient)
	if err := qdrantWrap.CheckConnection(ctx); err != nil {
		qdrantClient.Close()
		return nil, fmt.Errorf("failed to check qdrant connection: %w", err)
	}

	// Initialize Ollama connector
	ollama, err := neuro.NewConnector(&cfg.Ollama)
	if err != nil {
		qdrantClient.Close()
		return nil, fmt.Errorf("failed to create ollama connector: %w", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.GetAddress(), DB: cfg.Redis.DB})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		qdrantClient.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	slog.Info("connected to redis")

	historyStore := _redis.New(&cfg.Redis, redisClient)

	// Create service
	svc := service.New(qdrantWrap, ollama, historyStore)

	// Create HTTP handler if enabled
	var api *http.Handler
	if cfg.UseHTTP {
		api = http.New(&cfg.HTTP, svc)
		api.RegisterRoutes()
	}

	return &App{
		cfg:            cfg,
		qdrantClient:   qdrantClient,
		redisClient:    redisClient,
		tracerProvider: tp,
		svc:            svc,
		api:            api,
	}, nil
}

// Run starts the application and blocks until shutdown signal is received.
func (a *App) Run() error {
	ctx := context.Background()

	// Start HTTP server if enabled
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	if a.api != nil {
		go func() {
			slog.Info("starting HTTP server", "address", a.cfg.HTTP.GetAddress())
			if err := a.api.Run(); err != nil {
				slog.ErrorContext(ctx, "HTTP server error", "error", err)
				quit <- syscall.SIGTERM
			}
		}()
	}

	// Wait for shutdown signal
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.InfoContext(shutdownCtx, "shutting down the service...")

	// Shutdown HTTP server
	if a.api != nil {
		if err := a.api.Shutdown(shutdownCtx); err != nil {
			slog.ErrorContext(shutdownCtx, "HTTP server shutdown error", "error", err)
		}
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

	// Close Qdrant client
	if err := a.qdrantClient.Close(); err != nil {
		slog.ErrorContext(shutdownCtx, "qdrant client shutdown error", "error", err)
	}

	slog.InfoContext(shutdownCtx, "service stopped gracefully")
	return nil
}
