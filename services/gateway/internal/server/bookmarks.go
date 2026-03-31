package server

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// listBookmarks handles GET /bookmarks - list user bookmarks with filtering and pagination.
func (s *Server) listBookmarks(c *fiber.Ctx) error {
	var query domain.ListBookmarksQuery

	if err := c.QueryParser(&query); err != nil {
		slog.Error("failed to parse bookmarks query parameters", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid query parameters",
		})
	}

	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 16
	}

	if authUser, ok := getAuthUser(c); ok {
		query.UserID = authUser.ID
	}

	if err := s.validator.Struct(&query); err != nil {
		slog.Error("bookmarks query validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid query parameters",
			Details: parseValidationErrors(err),
		})
	}

	response, err := s.documentConnector.ListBookmarks(c.UserContext(), query)
	if err != nil {
		slog.Error("failed to list bookmarks", "error", err)
		status, errorCode, message := mapBookmarkConnectorError(err, "Failed to retrieve bookmarks")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// createBookmark handles POST /bookmarks - save a bookmark for the authenticated user.
func (s *Server) createBookmark(c *fiber.Ctx) error {
	var req domain.CreateBookmarkRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse bookmark request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if authUser, ok := getAuthUser(c); ok {
		req.UserID = authUser.ID
	}

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("bookmark request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	response, err := s.documentConnector.CreateBookmark(c.UserContext(), req)
	if err != nil {
		slog.Error("failed to create bookmark", "error", err)
		status, errorCode, message := mapBookmarkConnectorError(err, "Failed to save bookmark")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.Status(http.StatusCreated).JSON(response)
}

// deleteBookmark handles DELETE /bookmarks/:id - remove a bookmark for the authenticated user.
func (s *Server) deleteBookmark(c *fiber.Ctx) error {
	bookmarkID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		slog.Error("invalid bookmark ID", "id", c.Params("id"), "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid bookmark ID format",
		})
	}

	authUser, ok := getAuthUser(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	if err := s.documentConnector.DeleteBookmark(c.UserContext(), authUser.ID, bookmarkID); err != nil {
		slog.Error("failed to delete bookmark", "bookmark_id", bookmarkID, "user_id", authUser.ID, "error", err)
		status, errorCode, message := mapBookmarkConnectorError(err, "Failed to delete bookmark")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.SendStatus(http.StatusNoContent)
}

func mapBookmarkConnectorError(err error, fallbackMessage string) (status int, errorCode string, message string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	switch {
	case strings.Contains(err.Error(), "invalid_request"):
		return http.StatusBadRequest, "invalid_request", fallbackMessage
	case strings.Contains(err.Error(), "not_found"):
		return http.StatusNotFound, "not_found", fallbackMessage
	default:
		return http.StatusInternalServerError, "internal_error", fallbackMessage
	}
}
