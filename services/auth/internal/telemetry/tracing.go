package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracerConfig содержит конфигурацию для tracer
type TracerConfig struct {
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string
	Enabled        bool
	ExportTimeout  time.Duration
	BatchTimeout   time.Duration
	MaxQueueSize   int
	MaxExportBatch int
}

// InitTracer инициализирует OpenTelemetry tracer provider
func InitTracer(cfg TracerConfig) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		slog.Info("OpenTelemetry tracing is disabled")
		return nil, nil
	}

	// Создаем контекст с таймаутом для инициализации exporter
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Настраиваем OTLP exporter
	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		otlptracegrpc.WithTimeout(cfg.ExportTimeout),
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Создаем ресурс с атрибутами сервиса
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Создаем BatchSpanProcessor с настройками для длительного экспорта
	// ВАЖНО: Используем отдельные таймауты, чтобы экспорт не зависел от контекста запроса
	batchProcessor := sdktrace.NewBatchSpanProcessor(
		exporter,
		sdktrace.WithBatchTimeout(cfg.BatchTimeout),         // Время между экспортами
		sdktrace.WithExportTimeout(cfg.ExportTimeout),       // Таймаут для одного экспорта
		sdktrace.WithMaxQueueSize(cfg.MaxQueueSize),         // Размер очереди
		sdktrace.WithMaxExportBatchSize(cfg.MaxExportBatch), // Размер батча
	)

	// Создаем TracerProvider с batch processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(batchProcessor),
	)

	// Устанавливаем глобальный tracer provider
	otel.SetTracerProvider(tp)

	slog.Info("OpenTelemetry tracer initialized",
		"service", cfg.ServiceName,
		"endpoint", cfg.OTLPEndpoint,
		"export_timeout", cfg.ExportTimeout,
		"batch_timeout", cfg.BatchTimeout,
	)

	return tp, nil
}

// Shutdown корректно завершает tracer provider
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) error {
	if tp == nil {
		return nil
	}

	slog.Info("Shutting down tracer provider...")

	// Форсируем экспорт всех оставшихся спанов перед завершением
	if err := tp.ForceFlush(ctx); err != nil {
		slog.Error("Failed to flush tracer provider", "error", err)
	}

	if err := tp.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	slog.Info("Tracer provider shut down successfully")
	return nil
}
