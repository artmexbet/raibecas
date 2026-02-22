package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/artmexbet/raibecas/libs/natsw"
	"github.com/artmexbet/raibecas/libs/telemetry"

	"github.com/artmexbet/raibecas/services/auth/internal/config"
	"github.com/artmexbet/raibecas/services/auth/internal/consumer"
	"github.com/artmexbet/raibecas/services/auth/internal/handler"
	natspkg "github.com/artmexbet/raibecas/services/auth/internal/nats"
	"github.com/artmexbet/raibecas/services/auth/internal/postgres"
	"github.com/artmexbet/raibecas/services/auth/internal/service"
	"github.com/artmexbet/raibecas/services/auth/internal/storeredis"
	"github.com/artmexbet/raibecas/services/auth/pkg/jwt"
)

// App represents the App-based auth service server
type App struct {
	cfg            *config.Config
	pool           *pgxpool.Pool
	redis          *redis.Client
	natsClient     *natsw.Client
	subscriber     *natspkg.Subscriber
	userConsumer   *consumer.UserConsumer
	subscriptions  []*natsgo.Subscription
	tracerProvider *sdktrace.TracerProvider
}

// New creates a new App-based server instance
func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	// Initialize OpenTelemetry tracing
	tracerProvider, err := telemetry.InitTracer(telemetry.TracerConfig{
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
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	// Initialize database connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}
	poolConfig.MaxConns = int32(cfg.Database.MaxConns)
	poolConfig.MinConns = int32(cfg.Database.MinConns)
	poolConfig.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Validate database connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	if err := redisotel.InstrumentTracing(
		redisClient,
		redisotel.WithTracerProvider(tracerProvider),
	); err != nil {
		pool.Close()
		redisClient.Close() //nolint:errcheck // safe to ignore error on close during setup failure
		return nil, fmt.Errorf("failed to instrument redis client: %w", err)
	}

	// Initialize App connection
	natsConn, err := natsgo.Connect(
		cfg.NATS.URL,
		natsgo.Name(cfg.NATS.ConnectionName),
		natsgo.MaxReconnects(cfg.NATS.MaxReconnects),
		natsgo.ReconnectWait(cfg.NATS.ReconnectWait),
	)
	if err != nil {
		pool.Close()
		redisClient.Close() //nolint:errcheck // safe to ignore error on close during setup failure
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	natsTracer := tracerProvider.Tracer("nats-client")
	// Create nats wrapper client with middleware
	natsClient := natsw.NewClient(natsConn,
		natsw.WithLogger(slog.Default()),
		natsw.WithRecover(),
		natsw.WithTracer(natsTracer),
		natsw.WithMiddleware(natsw.TraceHandlerMiddleware(natsTracer)),
	)

	pgs := postgres.New(pool)
	// Initialize repositories

	// Initialize stores
	tokenStore := storeredis.NewTokenStoreRedis(redisClient, nil)

	// Initialize JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.Issuer,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
		tokenStore,
	)

	// Initialize services
	authService := service.NewAuthService(pgs, jwtManager)
	regService := service.NewRegistrationService(pgs, pgs)

	// Initialize nats publisher and subscriber
	publisher := natspkg.NewPublisher(natsConn)
	subscriber := natspkg.NewSubscriber(natsConn, regService, publisher)

	// Initialize user consumer for outbox events
	userConsumer := consumer.NewUserConsumer(pgs, slog.Default())

	// Initialize App handlers
	authHandler := handler.NewAuthHandler(authService, publisher)
	regHandler := handler.NewRegistrationHandler(regService, publisher)

	// Setup App subscriptions
	server := &App{
		cfg:            cfg,
		pool:           pool,
		redis:          redisClient,
		natsClient:     natsClient,
		subscriber:     subscriber,
		userConsumer:   userConsumer,
		subscriptions:  make([]*natsgo.Subscription, 0),
		tracerProvider: tracerProvider,
	}

	// Subscribe to request/reply topics
	if err := server.setupSubscriptions(authHandler, regHandler); err != nil {
		return nil, fmt.Errorf("failed to setup subscriptions: %w", err)
	}

	// Subscribe to user registration events from users service
	if err := userConsumer.Subscribe(natsClient); err != nil {
		return nil, fmt.Errorf("failed to subscribe user consumer: %w", err)
	}

	// Start event subscriber (for admin approval/rejection events)
	if err := subscriber.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start event subscriber: %w", err)
	}

	return server, nil
}

// setupSubscriptions sets up App request/reply subscriptions
func (s *App) setupSubscriptions(authHandler *handler.AuthHandler, regHandler *handler.RegistrationHandler) error {
	// Auth service subscriptions using natsw.Client
	sub, err := s.natsClient.Subscribe(natspkg.SubjectAuthRegister, regHandler.HandleRegister)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", natspkg.SubjectAuthRegister, err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsClient.Subscribe(natspkg.SubjectAuthLogin, authHandler.HandleLogin)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", natspkg.SubjectAuthLogin, err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsClient.Subscribe(natspkg.SubjectAuthRefresh, authHandler.HandleRefresh)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", natspkg.SubjectAuthRefresh, err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsClient.Subscribe(natspkg.SubjectAuthValidate, authHandler.HandleValidate)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", natspkg.SubjectAuthValidate, err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsClient.Subscribe(natspkg.SubjectAuthLogout, authHandler.HandleLogout)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", natspkg.SubjectAuthLogout, err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsClient.Subscribe(natspkg.SubjectAuthLogoutAll, authHandler.HandleLogoutAll)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", natspkg.SubjectAuthLogoutAll, err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsClient.Subscribe(natspkg.SubjectAuthChangePassword, authHandler.HandleChangePassword)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", natspkg.SubjectAuthChangePassword, err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	slog.Info("App subscriptions setup complete",
		"topics", []string{
			natspkg.SubjectAuthRegister,
			natspkg.SubjectAuthLogin,
			natspkg.SubjectAuthRefresh,
			natspkg.SubjectAuthValidate,
			natspkg.SubjectAuthLogout,
			natspkg.SubjectAuthLogoutAll,
			natspkg.SubjectAuthChangePassword,
		})

	return nil
}

// Start starts the App server
func (s *App) Start() error {
	slog.Info("Auth service is ready and listening on App topics")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down auth service...")
	return s.Shutdown()
}

// Shutdown gracefully shuts down the server
func (s *App) Shutdown() error {
	// Создаем контекст с таймаутом для shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Unsubscribe from all topics
	for _, sub := range s.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			slog.Error("Error unsubscribing", "error", err)
		}
	}

	// Stop App event subscribers
	if err := s.subscriber.Stop(); err != nil {
		slog.Error("Error stopping App subscribers", "error", err)
	}

	// Close App connection
	s.natsClient.Close()

	// Shutdown tracer provider (flush all pending spans)
	if err := telemetry.Shutdown(ctx, s.tracerProvider); err != nil {
		slog.Error("Error shutting down tracer provider", "error", err)
	}

	// Close Redis connection
	if err := s.redis.Close(); err != nil {
		slog.Error("Error closing Redis connection", "error", err)
	}

	// Close database connection pool
	s.pool.Close()

	slog.Info("Auth service shut down successfully")
	return nil
}
