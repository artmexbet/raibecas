package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/natsw"
	"github.com/artmexbet/raibecas/libs/telemetry"

	"github.com/artmexbet/raibecas/services/documents/internal/config"
	natsPublisher "github.com/artmexbet/raibecas/services/documents/internal/nats"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres/queries"
	"github.com/artmexbet/raibecas/services/documents/internal/server"
	"github.com/artmexbet/raibecas/services/documents/internal/service"
	"github.com/artmexbet/raibecas/services/documents/internal/storage"
	"github.com/artmexbet/raibecas/services/documents/migrations"
)

// App represents the application
type App struct {
	cfg        *config.Config
	logger     *slog.Logger
	queries    *queries.Queries
	dbPool     *pgxpool.Pool
	storage    *storage.MinIOStorage
	natsConn   *nats.Conn
	natsClient *natsw.Client
	server     *server.Server
	shutdown   func(context.Context) error
}

// New creates a new application instance
func New(ctx context.Context) (*App, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	app := &App{
		cfg:    cfg,
		logger: logger,
	}

	// Initialize telemetry
	if cfg.Telemetry.Enabled {
		tp, err := telemetry.InitTracer(telemetry.TracerConfig{
			ServiceName:    cfg.Telemetry.ServiceName,
			ServiceVersion: cfg.Telemetry.ServiceVersion,
			OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
			Enabled:        cfg.Telemetry.Enabled,
		})
		if err != nil {
			logger.Warn("failed to initialize telemetry", "error", err)
		} else if tp != nil {
			app.shutdown = func(ctx context.Context) error {
				return tp.Shutdown(ctx)
			}
			logger.Info("telemetry initialized",
				"service", cfg.Telemetry.ServiceName,
				"endpoint", cfg.Telemetry.OTLPEndpoint,
			)
		}
	}

	if err := migrations.Up(cfg.Database.DSN()); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Initialize database
	q, pool, err := postgres.NewQueries(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	app.queries = q
	app.dbPool = pool
	logger.Info("connected to database",
		"host", cfg.Database.Host,
		"max_conns", cfg.Database.MaxConns,
	)

	// Initialize MinIO storage
	minioStorage, err := storage.NewMinIOStorage(cfg.MinIO, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize minio storage: %w", err)
	}
	app.storage = minioStorage

	// Ensure MinIO bucket exists
	if err := minioStorage.EnsureBucket(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure minio bucket: %w", err)
	}
	logger.Info("minio storage initialized", "bucket", cfg.MinIO.Bucket)

	// Initialize NATS
	natsConn, err := nats.Connect(
		cfg.NATS.URL,
		nats.Name(cfg.NATS.ConnectionName),
		nats.MaxReconnects(cfg.NATS.MaxReconnects),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}
	app.natsConn = natsConn
	logger.Info("connected to nats", "url", cfg.NATS.URL)

	// Create NATS client with middleware
	natsClient := natsw.NewClient(natsConn,
		natsw.WithLogger(logger),
		natsw.WithRecover(),
	)
	app.natsClient = natsClient

	// Initialize publisher
	publisher := natsPublisher.NewPublisher(natsClient, logger)

	// Initialize repositories
	docRepo := postgres.NewDocumentRepository(q)
	bookmarkRepo := postgres.NewBookmarkRepository(pool)
	versionRepo := postgres.NewVersionRepository(q)
	tagRepo := postgres.NewTagRepository(q)
	metadataRepo := postgres.NewMetadataRepository(q)

	// Initialize service with repositories
	docService := service.NewDocumentService(
		docRepo,
		bookmarkRepo,
		versionRepo,
		tagRepo,
		metadataRepo,
		minioStorage,
		publisher,
		logger,
	)

	// Initialize handlers
	docHandler := server.NewDocumentHandler(docService, logger)
	metadataHandler := server.NewMetadataHandler(docService, logger)

	// Initialize server
	srv := server.New(natsClient, docHandler, metadataHandler)
	app.server = srv

	return app, nil
}

// Run starts the application and blocks until shutdown signal
func (a *App) Run() error {
	// Start server (register subscriptions)
	if err := a.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	a.logger.Info("documents service started",
		"service", a.cfg.Telemetry.ServiceName,
		"nats_url", a.cfg.NATS.URL,
	)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	a.logger.Info("shutting down gracefully...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Cleanup with timeout
	if a.shutdown != nil {
		if err := a.shutdown(shutdownCtx); err != nil {
			a.logger.Error("failed to shutdown telemetry", "error", err)
		} else {
			a.logger.Info("telemetry shutdown complete")
		}
	}

	if a.natsConn != nil {
		a.natsConn.Close()
		a.logger.Info("nats connection closed")
	}

	if a.dbPool != nil {
		a.dbPool.Close()
		a.logger.Info("database connection pool closed")
	}

	a.logger.Info("shutdown complete")
	return nil
}
