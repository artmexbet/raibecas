package server

import (
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

func (s *Server) setupDocumentRoutes() {
	documents := s.router.Group("/documents")
	documents.Get("/", s.listDocuments)
	documents.Post("/", s.createDocument)
	documents.Get("/:id", s.getDocument)
	documents.Put("/:id", s.updateDocument)
	documents.Delete("/:id", s.deleteDocument)
}

// listDocuments handles GET /documents - list documents with filtering and pagination
func (s *Server) listDocuments(c *fiber.Ctx) error {
	var query domain.ListDocumentsQuery

	// Parse query parameters
	if err := c.QueryParser(&query); err != nil {
		slog.Error("failed to parse query parameters", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
		})
	}

	// Set defaults
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 20
	}

	// Validate query
	if err := s.validator.Struct(&query); err != nil {
		slog.Error("query validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid query parameters",
			Details: parseValidationErrors(err),
		})
	}

	// Call document service via connector
	response, err := s.documentConnector.ListDocuments(c.UserContext(), query)
	if err != nil {
		slog.Error("failed to list documents", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve documents",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// createDocument handles POST /documents - create a new document
func (s *Server) createDocument(c *fiber.Ctx) error {
	var req domain.CreateDocumentRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Validate request
	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	// Call document service via connector
	response, err := s.documentConnector.CreateDocument(c.UserContext(), req)
	if err != nil {
		slog.Error("failed to create document", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create document",
		})
	}

	return c.Status(http.StatusCreated).JSON(response)
}

// getDocument handles GET /documents/:id - get a single document by ID
func (s *Server) getDocument(c *fiber.Ctx) error {
	idStr := c.Params("id")

	// Parse UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("invalid document ID", "id", idStr, "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid document ID format",
		})
	}

	// Call document service via connector
	response, err := s.documentConnector.GetDocument(c.UserContext(), id)
	if err != nil {
		slog.Error("failed to get document", "id", id, "error", err)
		return c.Status(http.StatusNotFound).JSON(domain.ErrorResponse{
			Error:   "not_found",
			Message: "Document not found",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// updateDocument handles PUT /documents/:id - update an existing document
func (s *Server) updateDocument(c *fiber.Ctx) error {
	idStr := c.Params("id")

	// Parse UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("invalid document ID", "id", idStr, "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid document ID format",
		})
	}

	var req domain.UpdateDocumentRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Validate request
	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	// Call document service via connector
	response, err := s.documentConnector.UpdateDocument(c.UserContext(), id, req)
	if err != nil {
		slog.Error("failed to update document", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update document",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// deleteDocument handles DELETE /documents/:id - delete a document by ID
func (s *Server) deleteDocument(c *fiber.Ctx) error {
	idStr := c.Params("id")

	// Parse UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("invalid document ID", "id", idStr, "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid document ID format",
		})
	}

	// Call document service via connector
	if err := s.documentConnector.DeleteDocument(c.UserContext(), id); err != nil {
		slog.Error("failed to delete document", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete document",
		})
	}

	return c.SendStatus(http.StatusNoContent)
}

// parseValidationErrors extracts validation errors into a map
func parseValidationErrors(err error) map[string]string {
	details := make(map[string]string)
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			details[e.Field()] = e.Tag()
		}
	}
	return details
}
