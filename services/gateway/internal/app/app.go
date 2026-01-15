package app

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/services/gateway/internal/config"
	"github.com/artmexbet/raibecas/services/gateway/internal/connector"
	"github.com/artmexbet/raibecas/services/gateway/internal/server"
)

type App struct {
	cfg      *config.Config
	natsConn *nats.Conn
	server   *server.Server
}

func New() *App {
	// Load configuration
	cfg := &config.Config{
		HTTP: config.HTTPConfig{
			Host:    getEnvOrDefault("HTTP_HOST", "0.0.0.0"),
			Port:    8080,
			Timeout: 30 * time.Second,
			RPS:     100,
		},
		NATS: config.NATSConfig{
			URL:            getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
			RequestTimeout: 5 * time.Second,
			MaxReconnects:  10,
			ReconnectWait:  2 * time.Second,
		},
	}

	return &App{
		cfg: cfg,
	}
}

func (a *App) Run() error {
	// Connect to NATS
	natsConn, err := nats.Connect(
		a.cfg.NATS.URL,
		nats.MaxReconnects(a.cfg.NATS.MaxReconnects),
		nats.ReconnectWait(a.cfg.NATS.ReconnectWait),
	)
	if err != nil {
		return err
	}
	a.natsConn = natsConn
	slog.Info("connected to NATS", "url", a.cfg.NATS.URL)

	// Create document connector
	documentConnector := connector.NewNATSDocumentConnector(natsConn, a.cfg.NATS.RequestTimeout)

	// Create auth connector
	authConnector := connector.NewNATSAuthConnector(natsConn)

	// Create and start server
	a.server = server.New(&a.cfg.HTTP, documentConnector, authConnector)

	// Start server in goroutine
	go func() {
		if err := a.server.Listen(&a.cfg.HTTP); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down application...")

	// Shutdown server
	if err := a.server.Shutdown(); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	// Close NATS connection
	a.natsConn.Close()
	slog.Info("NATS connection closed")

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
