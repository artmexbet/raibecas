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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/users/internal/config"
	"github.com/artmexbet/raibecas/services/users/internal/handler"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
	"github.com/artmexbet/raibecas/services/users/internal/server"
	"github.com/artmexbet/raibecas/services/users/internal/service"
)

type App struct {
	server server.Server
	cfg    config.Config
	pg     *postgres.Postgres
	client *natsw.Client
	tracer *trace.TracerProvider
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Create tracer
	tracer, err := initTracer()
	if err != nil {
		return nil, fmt.Errorf("failed to init tracer: %w", err)
	}

	natsConn, err := nats.Connect(cfg.NATS.GetURL())
	if err != nil {
		return nil, err
	}

	pg, err := postgres.New(context.Background(), cfg.Database)
	if err != nil {
		return nil, err
	}

	client := natsw.NewClient(natsConn,
		natsw.WithLogger(slog.Default()),
		natsw.WithRecover(),
	)

	svc := service.New(pg)
	h := handler.New(svc)

	srv := server.New(client, h)

	return &App{
		server: srv,
		cfg:    cfg,
		pg:     pg,
		client: client,
		tracer: tracer,
	}, nil
}

func (a *App) Run() error {
	err := a.server.Start()
	if err != nil {
		return err
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.tracer.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown tracer", "error", err)
	}

	a.pg.Close()
	a.client.Close()
	return nil
}

func initTracer() (*trace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("users"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
