package server

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/artmexbet/raibecas/libs/dto"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// listAuthors handles GET /authors - list all authors
func (s *Server) listAuthors(c *fiber.Ctx) error {
	// Call document service via connector
	response, err := s.documentConnector.ListAuthors(c.UserContext())
	if err != nil {
		slog.Error("failed to list authors", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve authors",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// createAuthor handles POST /authors - create a new author
func (s *Server) createAuthor(c *fiber.Ctx) error {
	var req domain.CreateAuthorRequest

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

	// Get user role from context
	userRole := getUserRole(c)

	// Call document service via connector
	response, err := s.documentConnector.CreateAuthor(c.UserContext(), req, userRole)
	if err != nil {
		slog.Error("failed to create author", "error", err)

		// Check if it's an unauthorized error
		if err.Error() == string(dto.ErrCodeUnauthorized) {
			return c.Status(http.StatusForbidden).JSON(domain.ErrorResponse{
				Error:   "forbidden",
				Message: "Insufficient permissions to create author",
			})
		}

		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create author",
		})
	}

	return c.Status(http.StatusCreated).JSON(response)
}

// listCategories handles GET /categories - list all categories
func (s *Server) listCategories(c *fiber.Ctx) error {
	// Call document service via connector
	response, err := s.documentConnector.ListCategories(c.UserContext())
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve categories",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// createCategory handles POST /categories - create a new category
func (s *Server) createCategory(c *fiber.Ctx) error {
	var req domain.CreateCategoryRequest

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

	// Get user role from context
	userRole := getUserRole(c)

	// Call document service via connector
	response, err := s.documentConnector.CreateCategory(c.UserContext(), req, userRole)
	if err != nil {
		slog.Error("failed to create category", "error", err)

		// Check if it's an unauthorized error
		if err.Error() == string(dto.ErrCodeUnauthorized) {
			return c.Status(http.StatusForbidden).JSON(domain.ErrorResponse{
				Error:   "forbidden",
				Message: "Insufficient permissions to create category",
			})
		}

		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create category",
		})
	}

	return c.Status(http.StatusCreated).JSON(response)
}

// listDocumentTypes handles GET /document-types - list all document types.
func (s *Server) listDocumentTypes(c *fiber.Ctx) error {
	response, err := s.documentConnector.ListDocumentTypes(c.UserContext())
	if err != nil {
		slog.Error("failed to list document types", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve document types",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// listAuthorshipTypes handles GET /authorship-types - list all authorship types.
func (s *Server) listAuthorshipTypes(c *fiber.Ctx) error {
	response, err := s.documentConnector.ListAuthorshipTypes(c.UserContext())
	if err != nil {
		slog.Error("failed to list authorship types", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve authorship types",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// listTags handles GET /tags - list all tags
func (s *Server) listTags(c *fiber.Ctx) error {
	// Call document service via connector
	response, err := s.documentConnector.ListTags(c.UserContext())
	if err != nil {
		slog.Error("failed to list tags", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve tags",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// createTag handles POST /tags - create a new tag
func (s *Server) createTag(c *fiber.Ctx) error {
	var req domain.CreateTagRequest

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

	// Get user role from context
	userRole := getUserRole(c)

	// Call document service via connector
	response, err := s.documentConnector.CreateTag(c.UserContext(), req, userRole)
	if err != nil {
		slog.Error("failed to create tag", "error", err)

		// Check if it's an unauthorized error
		if err.Error() == string(dto.ErrCodeUnauthorized) {
			return c.Status(http.StatusForbidden).JSON(domain.ErrorResponse{
				Error:   "forbidden",
				Message: "Insufficient permissions to create tag",
			})
		}

		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create tag",
		})
	}

	return c.Status(http.StatusCreated).JSON(response)
}
