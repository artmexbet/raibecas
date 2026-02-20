package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/artmexbet/raibecas/libs/natsw"
	"github.com/artmexbet/raibecas/libs/telemetry"

	"github.com/artmexbet/raibecas/services/gateway/internal/config"
	"github.com/artmexbet/raibecas/services/gateway/internal/connector"
	"github.com/artmexbet/raibecas/services/gateway/internal/server"
)

type App struct {
	cfg      *config.Config
	natsConn *nats.Conn
	server   *server.Server
	tracer   *trace.TracerProvider
}

// New creates a new App instance with all dependencies initialized
func New() (*App, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	slog.Info("configuration loaded",
		"http_host", cfg.HTTP.Host,
		"http_port", cfg.HTTP.Port,
		"nats_url", cfg.NATS.URL,
	)

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

	// Initialize tracer
	tp, err := telemetry.InitTracer(telemetry.TracerConfig{
		ServiceName:    cfg.Telemetry.ServiceName,
		ServiceVersion: cfg.Telemetry.ServiceVersion,
		OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
		Enabled:        cfg.Telemetry.Enabled,
		ExportTimeout:  cfg.Telemetry.ExportTimeout,
		BatchTimeout:   cfg.Telemetry.BatchTimeout,
		MaxQueueSize:   cfg.Telemetry.MaxQueueSize,
		MaxExportBatch: cfg.Telemetry.MaxExportBatch,
	})
	if err != nil {
		natsConn.Close()
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	natsTracer := tp.Tracer("nats-client")

	// Create single NATS wrapper client for trace propagation
	natsClient := natsw.NewClient(
		natsConn,
		natsw.WithRecover(),
		natsw.WithLogger(slog.Default()),
		natsw.WithTracer(natsTracer),
		natsw.WithMiddleware(natsw.TraceHandlerMiddleware(natsTracer)),
	)

	// Create connectors with shared NATS client
	documentConnector := connector.NewNATSDocumentConnector(natsClient, cfg.NATS.RequestTimeout)
	authConnector := connector.NewNATSAuthConnector(natsClient)
	userConnector := connector.NewNATSUserConnector(natsClient)

	// Create chat WebSocket connector
	chatConnector := connector.NewChatWSConnector(cfg.ChatService.WebSocketURL)

	// Create server
	srv := server.New(&cfg.HTTP, cfg.CORS, documentConnector, authConnector, userConnector, chatConnector)

	return &App{
		cfg:      cfg,
		natsConn: natsConn,
		server:   srv,
		tracer:   tp,
	}, nil
}

func (a *App) Run() error {
	// Start server in goroutine
	go func() {
		slog.Info("starting server", "address", fmt.Sprintf("%s:%d", a.cfg.HTTP.Host, a.cfg.HTTP.Port))
		if err := a.server.Listen(&a.cfg.HTTP); err != nil {
			slog.ErrorContext(context.Background(), "server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	slog.InfoContext(ctx, "shutting down application...")

	// Shutdown server
	if err := a.server.Shutdown(); err != nil {
		slog.ErrorContext(ctx, "server shutdown error", "error", err)
	}

	// Close NATS connection
	a.natsConn.Close()
	slog.InfoContext(ctx, "NATS connection closed")

	if err := telemetry.Shutdown(ctx, a.tracer); err != nil {
		slog.ErrorContext(ctx, "tracer shutdown error", "error", err)
	}

	slog.InfoContext(ctx, "application shutdown complete")

	return nil
}
