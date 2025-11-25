package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	recover2 "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

type iService interface {
	ProcessInput(ctx context.Context, input, userID string, fn func(response domain.ChatResponse) error) error
	ClearUserChat(ctx context.Context, userID string) error
}

// Handler represents the HTTP handler.
// It used by only for testing the service.
type Handler struct {
	router *fiber.App
	svc    iService
	cfg    *config.HTTP
}

func New(cfg *config.HTTP, svc iService) *Handler {
	router := fiber.New()

	router.Use(cors.New(
		cors.Config{
			AllowOrigins: "*",
		},
	))
	router.Use(logger.New())
	router.Use(recover2.New())
	router.Use(healthcheck.New())
	router.Use(requestid.New())

	return &Handler{
		router: router,
		svc:    svc,
		cfg:    cfg,
	}
}

func (h *Handler) RegisterRoutes() {
	h.router.Post("/api/v1/chat", h.chatHandler)

	// Clear chat history endpoint
	h.router.Delete("/api/v1/chat/:userID", h.deleteChatHandler)
}

func (h *Handler) Shutdown(ctx context.Context) error {
	return h.router.ShutdownWithContext(ctx)
}

func (h *Handler) Run() error {
	return h.router.Listen(h.cfg.GetAddress())
}
