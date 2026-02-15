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
	userConnector     UserServiceConnector
	validator         *validator.Validate
}

func New(cfg *config.HTTPConfig, corsCfg config.CORSConfig, documentConnector DocumentServiceConnector, authConnector AuthServiceConnector, userConnector UserServiceConnector) *Server {
	router := fiber.New()
	logger := slog.Default()
	router.Use(slogfiber.New(logger))

	// CORS configuration for cookie-based authentication
	router.Use(cors.New(cors.Config{
		AllowOrigins:     corsCfg.AllowOrigins,
		AllowCredentials: true, // Required for cookies
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Device-ID",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
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
		userConnector:     userConnector,
		validator:         validator.New(),
	}

	// Setup routes
	s.setupPublicRoutes()
	s.setupCookieAuthRoutes()
	s.setupProtectedRoutes()

	return s
}

// setupPublicRoutes sets up public routes that don't require authentication
func (s *Server) setupPublicRoutes() {
	// Auth routes - login doesn't require authentication
	auth := s.router.Group("/api/v1/auth")
	auth.Post("/login", s.login)

	// Registration requests - creating request is public
	registrationRequests := s.router.Group("/api/v1/registration-requests")
	registrationRequests.Post("/", s.createRegistrationRequest)
}

// setupCookieAuthRoutes sets up routes that work with cookie-based refresh flow
// These endpoints allow authentication via refresh token in cookies
func (s *Server) setupCookieAuthRoutes() {
	// Apply cookie auth middleware - allows both access token headers and refresh token cookies
	cookieProtected := s.router.Group("", s.cookieAuthMiddleware())

	// Auth refresh endpoint - works with cookies when access token is expired
	auth := cookieProtected.Group("/api/v1/auth")
	auth.Post("/refresh", s.refreshToken)
	auth.Post("/validate", s.validateToken)
}

// setupProtectedRoutes sets up protected routes that require authentication
func (s *Server) setupProtectedRoutes() {
	// Apply auth middleware to all protected routes
	protected := s.router.Group("", s.authMiddleware())

	// Auth routes (except login and refresh)
	auth := protected.Group("/api/v1/auth")
	auth.Post("/logout", s.logout)
	auth.Post("/logout-all", s.logoutAll)
	auth.Post("/change-password", s.changePassword)

	// Documents routes
	docs := protected.Group("/api/v1/documents")
	docs.Get("/", s.listDocuments)
	docs.Get("/:id", s.getDocument)
	docs.Post("/", s.createDocument)
	docs.Patch("/:id", s.updateDocument)
	docs.Delete("/:id", s.deleteDocument)

	// Authors routes
	authors := protected.Group("/api/v1/authors")
	authors.Get("/", s.listAuthors)
	authors.Post("/", s.createAuthor)

	// Categories routes
	categories := protected.Group("/api/v1/categories")
	categories.Get("/", s.listCategories)
	categories.Post("/", s.createCategory)

	// Tags routes
	tags := protected.Group("/api/v1/tags")
	tags.Get("/", s.listTags)
	tags.Post("/", s.createTag)

	// Users routes
	users := protected.Group("/api/v1/users")
	users.Get("/", s.listUsers)
	users.Get("/:id", s.getUser)
	users.Patch("/:id", s.updateUser)
	users.Delete("/:id", s.deleteUser)

	// Registration requests (protected actions)
	registrationRequests := protected.Group("/api/v1/registration-requests")
	registrationRequests.Get("/", s.listRegistrationRequests)
	registrationRequests.Post("/:id/approve", s.approveRegistrationRequest)
	registrationRequests.Post("/:id/reject", s.rejectRegistrationRequest)
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
