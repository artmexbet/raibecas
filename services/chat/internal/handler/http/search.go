package http

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

// searchHandler handles GET /api/v1/search?q=<query>&limit=<N>
func (h *Handler) searchHandler(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "query parameter 'q' is required",
		})
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = min(parsed, 50) // cap at 50
		}
	}

	slog.DebugContext(c.UserContext(), "semantic search request",
		slog.String("query", query),
		slog.Int("limit", limit),
	)

	result, err := h.svc.Search(c.UserContext(), query, limit)
	if err != nil {
		slog.ErrorContext(c.UserContext(), "semantic search failed",
			slog.String("query", query),
			slog.String("error", err.Error()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "search failed",
		})
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// Compile-time check that *domain.SearchResponse is used (prevents unused import).
var _ = (*domain.SearchResponse)(nil)
