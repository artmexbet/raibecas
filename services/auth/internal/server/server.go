package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"auth/internal/config"
	"auth/internal/handler"
	"auth/internal/middleware"
	"auth/internal/nats"
	"auth/internal/repository"
	"auth/internal/service"
	"auth/internal/storeredis"
	"auth/pkg/jwt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	natsgo "github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// Server represents the auth service server
type Server struct {
	cfg        *config.Config
	app        *fiber.App
	pool       *pgxpool.Pool
	redis      *redis.Client
	natsConn   *natsgo.Conn
	subscriber *nats.Subscriber
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
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
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
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

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		ErrorHandler: errorHandler,
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	// Initialize repositories
	userRepo := repository.NewUserRepository(pool)
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
	authService := service.NewAuthService(userRepo, tokenStore, jwtManager)
	regService := service.NewRegistrationService(regRepo, userRepo)

	// Initialize NATS publisher and subscriber
	publisher := nats.NewPublisher(natsConn)
	subscriber := nats.NewSubscriber(natsConn, regService, publisher)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, publisher)
	regHandler := handler.NewRegistrationHandler(regService, publisher)

	// Setup routes
	setupRoutes(app, authHandler, regHandler, authService)

	return &Server{
		cfg:        cfg,
		app:        app,
		pool:       pool,
		redis:      redisClient,
		natsConn:   natsConn,
		subscriber: subscriber,
	}, nil
}

// Start starts the server
func (s *Server) Start() error {
	// Start NATS subscribers
	if err := s.subscriber.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start NATS subscribers: %w", err)
	}

	log.Printf("Starting auth service on port %s", s.cfg.Server.Port)
	return s.app.Listen(":" + s.cfg.Server.Port)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	log.Println("Shutting down auth service...")

	// Stop NATS subscribers
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

	// Shutdown Fiber app
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
	defer cancel()

	return s.app.ShutdownWithContext(ctx)
}

// setupRoutes sets up the HTTP routes
func setupRoutes(app *fiber.App, authHandler *handler.AuthHandler, regHandler *handler.RegistrationHandler, authService *service.AuthService) {
	api := app.Group("/api/v1")

	// Public routes
	api.Post("/register", regHandler.Register)
	api.Post("/login", authHandler.Login)
	api.Post("/refresh", authHandler.Refresh)
	api.Post("/validate", authHandler.Validate)

	// Protected routes (require authentication)
	protected := api.Group("/", middleware.AuthMiddleware(authService))
	protected.Post("/logout", authHandler.Logout)
	protected.Post("/logout-all", authHandler.LogoutAll)
	protected.Post("/change-password", authHandler.ChangePassword)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"service": "auth",
		})
	})
}

// errorHandler handles errors globally
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal server error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}
