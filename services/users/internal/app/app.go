package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/artmexbet/raibecas/libs/natsw"
	"github.com/artmexbet/raibecas/libs/telemetry"

	"github.com/artmexbet/raibecas/services/users/internal/config"
	"github.com/artmexbet/raibecas/services/users/internal/handler"
	"github.com/artmexbet/raibecas/services/users/internal/middleware"
	natspublisher "github.com/artmexbet/raibecas/services/users/internal/nats"
	"github.com/artmexbet/raibecas/services/users/internal/outbox"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
	"github.com/artmexbet/raibecas/services/users/internal/server"
	"github.com/artmexbet/raibecas/services/users/internal/service"
)

type App struct {
	server          server.Server
	cfg             config.Config
	pg              *postgres.Postgres
	client          *natsw.Client
	tracer          *trace.TracerProvider
	metrics         *middleware.Metrics
	outboxProcessor *outbox.Processor
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create tracer using unified telemetry package
	tracer, err := telemetry.InitTracer(telemetry.TracerConfig{
		ServiceName:    "users",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "localhost:4318",
		Enabled:        true,
		ExportTimeout:  30 * time.Second,
		BatchTimeout:   5 * time.Second,
		MaxQueueSize:   2048,
		MaxExportBatch: 512,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	natsConn, err := nats.Connect(cfg.NATS.GetURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	// Create context for database initialization
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pg, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		natsConn.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	metrics := middleware.NewMetrics(prometheus.DefaultRegisterer)

	natsTracer := tracer.Tracer("nats-client")
	client := natsw.NewClient(natsConn,
		natsw.WithLogger(slog.Default()),
		natsw.WithRecover(),
		natsw.WithMiddleware(metrics.Middleware),
		natsw.WithTracer(natsTracer),
		natsw.WithMiddleware(natsw.TraceHandlerMiddleware(natsTracer)),
	)

	svc := service.New(pg, metrics)
	h := handler.New(svc)

	srv := server.New(client, h)

	// Create outbox processor
	publisher := natspublisher.NewPublisher(client)
	outboxProcessor := outbox.NewProcessor(pg, publisher, slog.Default())

	return &App{
		server:          srv,
		cfg:             cfg,
		pg:              pg,
		client:          client,
		tracer:          tracer,
		metrics:         metrics,
		outboxProcessor: outboxProcessor,
	}, nil
}

func (a *App) Run() error {
	err := a.server.Start()
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	metricsSrv := a.startMetricsServer()

	metricsCtx, cancelMetrics := context.WithCancel(context.Background())
	go a.runMetricCollectors(metricsCtx)

	// Start outbox processor
	outboxCtx, cancelOutbox := context.WithCancel(context.Background())
	go func() {
		if err := a.outboxProcessor.Start(outboxCtx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("outbox processor error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	cancelMetrics()
	cancelOutbox()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := metricsSrv.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown metrics server", "error", err)
	}

	if err := telemetry.Shutdown(ctx, a.tracer); err != nil {
		slog.Error("failed to shutdown tracer", "error", err)
	}

	a.pg.Close()
	a.client.Close()
	return nil
}

func (a *App) startMetricsServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.cfg.Metrics.Port),
		Handler: mux,
	}

	go func() {
		slog.Info("starting metrics server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	return srv
}

func (a *App) runMetricCollectors(ctx context.Context) {
	a.updateUserCountMetrics(ctx)

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.updateUserCountMetrics(ctx)
		}
	}
}

func (a *App) updateUserCountMetrics(ctx context.Context) {
	count, err := a.pg.CountTotalUsers(ctx)
	if err != nil {
		slog.Error("failed to update user count metrics", "error", err)
		return
	}
	a.metrics.UsersTotal.Set(float64(count))
}
