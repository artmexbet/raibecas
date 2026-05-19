package server

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// listUsers handles GET /api/v1/users - list all users with pagination and filtering
func (s *Server) listUsers(c *fiber.Ctx) error {
	var query domain.ListUsersQuery

	// Parse query parameters
	if err := c.QueryParser(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters",
		})
	}

	// Set defaults if not provided
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 10
	}

	// Validate query
	if err := s.validator.Struct(query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid query parameters",
			Details: parseValidationErrors(err),
		})
	}

	// Call users service via NATS
	resp, err := s.userConnector.ListUsers(c.UserContext(), query)
	if err != nil {
		slog.Error("failed to list users", "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to retrieve users")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.JSON(resp)
}

// getUser handles GET /api/v1/users/:id - get a single user by ID
func (s *Server) getUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid user ID format",
		})
	}

	// Call users service via NATS
	resp, err := s.userConnector.GetUser(c.UserContext(), id)
	if err != nil {
		slog.Error("failed to get user", "user_id", id, "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to retrieve user")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.JSON(resp)
}

// updateUser handles PATCH /api/v1/users/:id - update user information
func (s *Server) updateUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid user ID format",
		})
	}

	var req domain.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	// Call users service via NATS
	resp, err := s.userConnector.UpdateUser(c.UserContext(), id, req)
	if err != nil {
		slog.Error("failed to update user", "user_id", id, "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to update user")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.JSON(resp)
}

// deleteUser handles DELETE /api/v1/users/:id - delete a user
func (s *Server) deleteUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid user ID format",
		})
	}

	// Call users service via NATS
	if err := s.userConnector.DeleteUser(c.UserContext(), id); err != nil {
		slog.Error("failed to delete user", "user_id", id, "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to delete user")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.JSON(domain.SuccessResponse{
		Message: "User deleted successfully",
	})
}
