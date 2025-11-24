package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"
	"github.com/artmexbet/raibecas/services/chat/internal/handler/models"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	recover2 "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type iService interface {
	ProcessInput(ctx context.Context, input, userID string, fn func(response domain.ChatResponse) error) error
}

// Handler represents the HTTP handler.
// It is used to test the service.
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
	h.router.Post("/api/v1/chat", func(c *fiber.Ctx) error { //todo: move to constants
		slog.Debug("Received chat request", slog.String("request_id", c.Get(fiber.HeaderXRequestID)))

		// Parse request body
		var req models.ChatRequest
		err := req.UnmarshalJSON(c.Body())
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		// Set headers for streaming response
		c.Set("Content-Type", "application/x-ndjson")
		c.Set("Transfer-Encoding", "chunked")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Cache-Control", "no-cache")

		slog.Debug("Processing chat input", slog.String("request_id", c.Get(fiber.HeaderXRequestID)))
		buf := bytes.NewBuffer(nil)
		// Stream chunks as they arrive
		err = h.svc.ProcessInput(c.UserContext(), req.Input, req.UserID, func(response domain.ChatResponse) error {
			slog.DebugContext(c.UserContext(), "Sending chat response chunk",
				slog.String("chunk", response.Message.Content),
			)
			// Marshal the response to JSON
			data, err := json.Marshal(response)
			if err != nil {
				return err
			}
			buf.WriteString(string(data) + "\n")

			// Flush the response writer to send data immediately
			return nil
		})

		if err != nil {
			return err
		}
		return c.SendStream(buf)
	})
}

func (h *Handler) Shutdown(ctx context.Context) error {
	return h.router.ShutdownWithContext(ctx)
}

func (h *Handler) Run() error {
	return h.router.Listen(h.cfg.GetAddress())
}
