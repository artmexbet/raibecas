package server

import (
	"fmt"
	"log/slog"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	slogfiber "github.com/samber/slog-fiber"

	"github.com/artmexbet/raibecas/services/gateway/internal/config"
)

const serviceName = "gateway"

type Server struct {
	router            *fiber.App
	documentConnector DocumentServiceConnector
	authConnector     AuthServiceConnector
	validator         *validator.Validate
}

func New(cfg *config.HTTPConfig, documentConnector DocumentServiceConnector, authConnector AuthServiceConnector) *Server {
	router := fiber.New()
	logger := slog.Default()
	router.Use(slogfiber.New(logger))

	// CORS configuration for cookie-based authentication
	router.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000", // TODO: Configure specific origins in production
		AllowCredentials: true,                    // Required for cookies
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Device-ID",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
	}))

	router.Use(requestid.New())
	router.Use(limiter.New(limiter.Config{Max: cfg.RPS}))
	router.Use(recoverer.New())
	router.Use(healthcheck.New())

	// Init http metrics
	prometheus := fiberprometheus.New(serviceName)
	prometheus.RegisterAt(router, "/metrics")
	prometheus.SetSkipPaths([]string{"/livez", "/readyz"})
	prometheus.SetIgnoreStatusCodes([]int{401, 403, 404})
	router.Use(prometheus.Middleware)

	router.Use(
		otelfiber.Middleware(otelfiber.WithoutMetrics(true)),
	)

	s := &Server{
		router:            router,
		documentConnector: documentConnector,
		authConnector:     authConnector,
		validator:         validator.New(),
	}

	// Setup routes
	s.setupDocumentRoutes()
	s.setupUsersRoutes()
	s.setupAuthRoutes()
	s.setupRegistrationRequestRoutes()

	return s
}

func (s *Server) Listen(cfg *config.HTTPConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	slog.Info("starting server", "address", addr)
	return s.router.Listen(addr)
}

func (s *Server) Shutdown() error {
	slog.Info("shutting down server")
	return s.router.Shutdown()
}
