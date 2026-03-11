package server

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

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
	response, err := s.documentConnector.CreateDocument(c.UserContext(), req, getUserRole(c))
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
	req.ID = id

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
	response, err := s.documentConnector.UpdateDocument(c.UserContext(), req, getUserRole(c))
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
	if err := s.documentConnector.DeleteDocument(c.UserContext(), id, getUserRole(c)); err != nil {
		slog.Error("failed to delete document", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete document",
		})
	}

	return c.SendStatus(http.StatusNoContent)
}

const (
	maxCoverSize = 5 * 1024 * 1024 // 5 MB
)

var allowedCoverTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
}

// uploadCover handles POST /documents/:id/cover - upload cover image for a document
func (s *Server) uploadCover(c *fiber.Ctx) error {
	idStr := c.Params("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		slog.Error("invalid document ID", "id", idStr, "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid document ID format",
		})
	}

	file, err := c.FormFile("cover")
	if err != nil {
		slog.Error("failed to get cover file", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Cover file is required (field: cover)",
		})
	}

	if file.Size > maxCoverSize {
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Cover image must be smaller than 5 MB",
		})
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedCoverTypes[contentType] {
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Only JPEG, PNG and WebP images are allowed",
		})
	}

	f, err := file.Open()
	if err != nil {
		slog.Error("failed to open cover file", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to process cover file",
		})
	}
	defer f.Close() //nolint:errcheck

	data := make([]byte, file.Size)
	if _, err := f.Read(data); err != nil {
		slog.Error("failed to read cover file", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to read cover file",
		})
	}

	coverURL, err := s.documentConnector.UploadCover(c.UserContext(), id, data, contentType, getUserRole(c))
	if err != nil {
		slog.Error("failed to upload cover", "id", id, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to upload cover",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"cover_url": coverURL,
	})
}
