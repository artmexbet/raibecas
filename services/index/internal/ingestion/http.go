package ingestion

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	recoverer "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

type iPipeline interface {
	Index(ctx context.Context, doc domain.Document) error
}

type HTTPIngestor struct {
	app  *fiber.App
	pipe iPipeline
}

func NewHTTPIngestor(pipe iPipeline) *HTTPIngestor {
	app := fiber.New()
	app.Use(requestid.New())
	app.Use(recoverer.New())
	app.Use(logger.New())

	ingestor := &HTTPIngestor{app: app, pipe: pipe}
	app.Post("/api/v1/index", ingestor.index())

	return ingestor
}

func (i *HTTPIngestor) index() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			ID       string            `json:"id"`
			Content  string            `json:"content"`
			Title    string            `json:"title"`
			Metadata map[string]string `json:"metadata"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fiber.ErrBadRequest
		}
		if req.ID == "" || req.Content == "" {
			return fiber.NewError(fiber.StatusBadRequest, "id and content required")
		}
		doc := domain.Document{ID: req.ID, Content: req.Content, Title: req.Title, Metadata: req.Metadata}
		if err := i.pipe.Index(c.UserContext(), doc); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusAccepted)
	}
}

func (i *HTTPIngestor) Start(addr string) error {
	return i.app.Listen(addr)
}
