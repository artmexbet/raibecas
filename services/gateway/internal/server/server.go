package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
	slogfiber "github.com/samber/slog-fiber"

	"github.com/artmexbet/raibecas/services/gateway/internal/config"
	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// DocumentServiceConnector defines the interface for interacting with the document service
type DocumentServiceConnector interface {
	// ListDocuments retrieves a list of documents based on query parameters
	ListDocuments(ctx context.Context, query domain.ListDocumentsQuery) (*domain.ListDocumentsResponse, error)

	// GetDocument retrieves a single document by ID
	GetDocument(ctx context.Context, id uuid.UUID) (*domain.DocumentResponse, error)

	// CreateDocument creates a new document
	CreateDocument(ctx context.Context, req domain.CreateDocumentRequest) (*domain.DocumentResponse, error)

	// UpdateDocument updates an existing document
	UpdateDocument(ctx context.Context, id uuid.UUID, req domain.UpdateDocumentRequest) (*domain.DocumentResponse, error)

	// DeleteDocument deletes a document by ID
	DeleteDocument(ctx context.Context, id uuid.UUID) error
}

type Server struct {
	router            *fiber.App
	documentConnector DocumentServiceConnector
	validator         *validator.Validate
}

func New(cfg *config.HTTPConfig, documentConnector DocumentServiceConnector) *Server {
	router := fiber.New()
	logger := slog.Default()
	router.Use(slogfiber.New(logger))
	router.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	})) // Enable CORS middleware
	router.Use(requestid.New())
	router.Use(limiter.New(limiter.Config{Max: cfg.RPS}))
	router.Use(recoverer.New())
	router.Use(healthcheck.New())

	s := &Server{
		router:            router,
		documentConnector: documentConnector,
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
