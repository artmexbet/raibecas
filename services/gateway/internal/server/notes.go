package server

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// listNotes handles GET /notes - list user notes with filtering and pagination.
func (s *Server) listNotes(c *fiber.Ctx) error {
	authUser, ok := getAuthUser(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	var query domain.ListNotesQuery

	if err := c.QueryParser(&query); err != nil {
		slog.Error("failed to parse notes query parameters", "error", err)
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

	query.UserID = authUser.ID

	if err := s.validator.Struct(&query); err != nil {
		slog.Error("notes query validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid query parameters",
			Details: parseValidationErrors(err),
		})
	}

	response, err := s.documentConnector.ListNotes(c.UserContext(), query)
	if err != nil {
		slog.Error("failed to list notes", "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to retrieve notes")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// getNote handles GET /notes/:id - get a single note by ID.
func (s *Server) getNote(c *fiber.Ctx) error {
	noteID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		slog.Error("invalid note ID", "id", c.Params("id"), "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid note ID format",
		})
	}

	authUser, ok := getAuthUser(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	response, err := s.documentConnector.GetNote(c.UserContext(), authUser.ID, noteID)
	if err != nil {
		slog.Error("failed to get note", "note_id", noteID, "user_id", authUser.ID, "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to retrieve note")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// createNote handles POST /notes - create a note for the authenticated user.
func (s *Server) createNote(c *fiber.Ctx) error {
	authUser, ok := getAuthUser(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	var req domain.CreateNoteRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse note request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	req.UserID = authUser.ID

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("note request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	response, err := s.documentConnector.CreateNote(c.UserContext(), req)
	if err != nil {
		slog.Error("failed to create note", "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to create note")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.Status(http.StatusCreated).JSON(response)
}

// updateNote handles PUT /notes/:id - update a note for the authenticated user.
func (s *Server) updateNote(c *fiber.Ctx) error {
	noteID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		slog.Error("invalid note ID", "id", c.Params("id"), "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid note ID format",
		})
	}

	authUser, ok := getAuthUser(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	var req domain.UpdateNoteRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse note update request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	req.ID = noteID
	req.UserID = authUser.ID

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("note update request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	response, err := s.documentConnector.UpdateNote(c.UserContext(), req)
	if err != nil {
		slog.Error("failed to update note", "note_id", noteID, "user_id", authUser.ID, "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to update note")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// deleteNote handles DELETE /notes/:id - remove a note for the authenticated user.
func (s *Server) deleteNote(c *fiber.Ctx) error {
	noteID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		slog.Error("invalid note ID", "id", c.Params("id"), "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid note ID format",
		})
	}

	authUser, ok := getAuthUser(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	if err := s.documentConnector.DeleteNote(c.UserContext(), authUser.ID, noteID); err != nil {
		slog.Error("failed to delete note", "note_id", noteID, "user_id", authUser.ID, "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to delete note")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.SendStatus(http.StatusNoContent)
}
