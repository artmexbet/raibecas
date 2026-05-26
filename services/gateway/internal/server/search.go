package server

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// semanticSearch handles GET /api/v1/search?q=<query>&limit=<N>
// It sends a NATS request to corpus.search (index-python) and returns the results.
func (s *Server) semanticSearch(c *fiber.Ctx) error {
	var query domain.SearchQuery
	if err := c.QueryParser(&query); err != nil {
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
		})
	}

	if query.Q == "" {
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Query parameter 'q' is required",
		})
	}

	if query.Limit <= 0 {
		query.Limit = 10
	}

	if err := s.validator.Struct(&query); err != nil {
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid query parameters",
			Details: parseValidationErrors(err),
		})
	}

	result, err := s.documentConnector.SemanticSearch(c.UserContext(), query)
	if err != nil {
		slog.Error("semantic search failed", "query", query.Q, "error", err)
		status, errorCode, message := mapConnectorError(err, "Search failed")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.Status(http.StatusOK).JSON(result)
}
