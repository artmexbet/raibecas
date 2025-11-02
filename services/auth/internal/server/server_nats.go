package server

import (
	"auth/internal/postgres"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auth/internal/config"
	"auth/internal/handler"
	"auth/internal/nats"
	"auth/internal/repository"
	"auth/internal/service"
	"auth/internal/storeredis"
	"auth/pkg/jwt"

	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// NATS represents the NATS-based auth service server
type NATS struct {
	cfg           *config.Config
	pool          *pgxpool.Pool
	redis         *redis.Client
	natsConn      *natsgo.Conn
	subscriber    *nats.Subscriber
	subscriptions []*natsgo.Subscription
}

// NewNATS creates a new NATS-based server instance
func NewNATS(cfg *config.Config) (*NATS, error) {
	ctx := context.Background()

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
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize NATS connection
	natsConn, err := natsgo.Connect(
		cfg.NATS.URL,
		natsgo.Name(cfg.NATS.ConnectionName),
		natsgo.MaxReconnects(cfg.NATS.MaxReconnects),
		natsgo.ReconnectWait(cfg.NATS.ReconnectWait),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	pgs := postgres.New(pool)
	// Initialize repositories
	regRepo := repository.NewRegistrationRepository(pool)

	// Initialize stores
	tokenStore := storeredis.NewTokenStore(redisClient)

	// Initialize JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.Issuer,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
	)

	// Initialize services
	authService := service.NewAuthService(pgs, tokenStore, jwtManager)
	regService := service.NewRegistrationService(regRepo, pgs)

	// Initialize NATS publisher and subscriber
	publisher := nats.NewPublisher(natsConn)
	subscriber := nats.NewSubscriber(natsConn, regService, publisher)

	// Initialize NATS handlers
	authHandler := handler.NewAuthHandler(authService, publisher)
	regHandler := handler.NewRegistrationHandler(regService, publisher)

	// Setup NATS subscriptions
	server := &NATS{
		cfg:           cfg,
		pool:          pool,
		redis:         redisClient,
		natsConn:      natsConn,
		subscriber:    subscriber,
		subscriptions: make([]*natsgo.Subscription, 0),
	}

	// Subscribe to request/reply topics
	if err := server.setupSubscriptions(authHandler, regHandler); err != nil {
		return nil, fmt.Errorf("failed to setup subscriptions: %w", err)
	}

	// Start event subscriber (for admin approval/rejection events)
	if err := subscriber.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start event subscriber: %w", err)
	}

	return server, nil
}

// setupSubscriptions sets up NATS request/reply subscriptions
func (s *NATS) setupSubscriptions(authHandler *handler.AuthHandler, regHandler *handler.RegistrationHandler) error {
	var err error

	// Auth service subscriptions
	sub, err := s.natsConn.Subscribe("auth.register", regHandler.HandleRegister)
	if err != nil {
		return fmt.Errorf("failed to subscribe to auth.register: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsConn.Subscribe("auth.login", authHandler.HandleLogin)
	if err != nil {
		return fmt.Errorf("failed to subscribe to auth.login: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsConn.Subscribe("auth.refresh", authHandler.HandleRefresh)
	if err != nil {
		return fmt.Errorf("failed to subscribe to auth.refresh: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsConn.Subscribe("auth.validate", authHandler.HandleValidate)
	if err != nil {
		return fmt.Errorf("failed to subscribe to auth.validate: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsConn.Subscribe("auth.logout", authHandler.HandleLogout)
	if err != nil {
		return fmt.Errorf("failed to subscribe to auth.logout: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsConn.Subscribe("auth.logout_all", authHandler.HandleLogoutAll)
	if err != nil {
		return fmt.Errorf("failed to subscribe to auth.logout_all: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	sub, err = s.natsConn.Subscribe("auth.change_password", authHandler.HandleChangePassword)
	if err != nil {
		return fmt.Errorf("failed to subscribe to auth.change_password: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	log.Println("NATS subscriptions setup complete:")
	log.Println("  - auth.register")
	log.Println("  - auth.login")
	log.Println("  - auth.refresh")
	log.Println("  - auth.validate")
	log.Println("  - auth.logout")
	log.Println("  - auth.logout_all")
	log.Println("  - auth.change_password")

	return nil
}

// Start starts the NATS server
func (s *NATS) Start() error {
	log.Printf("Auth service is ready and listening on NATS topics")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down auth service...")
	return s.Shutdown()
}

// Shutdown gracefully shuts down the server
func (s *NATS) Shutdown() error {
	// Unsubscribe from all topics
	for _, sub := range s.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Error unsubscribing: %v", err)
		}
	}

	// Stop NATS event subscribers
	if err := s.subscriber.Stop(); err != nil {
		log.Printf("Error stopping NATS subscribers: %v", err)
	}

	// Close NATS connection
	s.natsConn.Close()

	// Close Redis connection
	if err := s.redis.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}

	// Close database connection pool
	s.pool.Close()

	log.Println("Auth service shut down successfully")
	return nil
}
