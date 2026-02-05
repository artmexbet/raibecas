package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
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

// InitTracer инициализирует OpenTelemetry tracer provider и устанавливает глобальные пропагаторы
func InitTracer(cfg TracerConfig) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		slog.Info("OpenTelemetry tracing is disabled")
		return nil, nil
	}

	// Создаем контекст с таймаутом для инициализации exporter
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Настраиваем OTLP exporter
	exporterOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		otlptracehttp.WithInsecure(), // Используем insecure connection для локальной разработки
		otlptracehttp.WithTimeout(cfg.ExportTimeout),
	}

	exporter, err := otlptracehttp.New(ctx, exporterOpts...)
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
	batchProcessor := sdktrace.NewBatchSpanProcessor(
		exporter,
		sdktrace.WithBatchTimeout(cfg.BatchTimeout),
		sdktrace.WithExportTimeout(cfg.ExportTimeout),
		sdktrace.WithMaxQueueSize(cfg.MaxQueueSize),
		sdktrace.WithMaxExportBatchSize(cfg.MaxExportBatch),
	)

	// Создаем TracerProvider с batch processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(batchProcessor),
	)

	// Устанавливаем глобальный tracer provider
	otel.SetTracerProvider(tp)

	// Устанавливаем глобальный text map propagator для trace context propagation
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

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

	return tp.Shutdown(ctx)
}
