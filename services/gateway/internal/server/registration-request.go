package server

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// createRegistrationRequest handles POST /api/v1/registration-requests - create new registration request (PUBLIC)
func (s *Server) createRegistrationRequest(c *fiber.Ctx) error {
	var req domain.CreateRegistrationRequestRequest

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
	resp, err := s.userConnector.CreateRegistrationRequest(c.UserContext(), req)
	if err != nil {
		slog.Error("failed to create registration request", "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to create registration request")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// listRegistrationRequests handles GET /api/v1/registration-requests - list registration requests (PROTECTED)
func (s *Server) listRegistrationRequests(c *fiber.Ctx) error {
	var query domain.ListRegistrationRequestsQuery

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
	resp, err := s.userConnector.ListRegistrationRequests(c.UserContext(), query)
	if err != nil {
		slog.Error("failed to list registration requests", "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to retrieve registration requests")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.JSON(resp)
}

// approveRegistrationRequest handles POST /api/v1/registration-requests/:id/approve - approve registration (PROTECTED)
func (s *Server) approveRegistrationRequest(c *fiber.Ctx) error {
	// Get request ID from params
	idParam := c.Params("id")
	requestID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid registration request ID format",
		})
	}

	// Get authenticated user (approver)
	authUser, ok := getAuthUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse optional role from body
	var body struct {
		Role string `json:"role"`
	}
	_ = c.BodyParser(&body)

	// Call users service via NATS
	resp, err := s.userConnector.ApproveRegistrationRequest(c.UserContext(), requestID, authUser.ID, body.Role)
	if err != nil {
		slog.Error("failed to approve registration request", "request_id", requestID, "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to approve registration request")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.JSON(resp)
}

// rejectRegistrationRequest handles POST /api/v1/registration-requests/:id/reject - reject registration (PROTECTED)
func (s *Server) rejectRegistrationRequest(c *fiber.Ctx) error {
	// Get request ID from params
	idParam := c.Params("id")
	requestID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid registration request ID format",
		})
	}

	// Get authenticated user (approver)
	authUser, ok := getAuthUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse optional reason from body
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.BodyParser(&body)

	// Call users service via NATS
	resp, err := s.userConnector.RejectRegistrationRequest(c.UserContext(), requestID, authUser.ID, body.Reason)
	if err != nil {
		slog.Error("failed to reject registration request", "request_id", requestID, "error", err)
		status, errorCode, message := mapConnectorError(err, "Failed to reject registration request")
		return c.Status(status).JSON(domain.ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
	}

	return c.JSON(resp)
}
