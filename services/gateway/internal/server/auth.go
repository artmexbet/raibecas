package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

func (s *Server) setupAuthRoutes() {
	auth := s.router.Group("/api/v1/auth")
	auth.Post("/login", s.login)
	auth.Post("/refresh", s.refreshToken)
	auth.Post("/validate", s.validateToken)
	auth.Post("/logout", s.logout)
	auth.Post("/logout-all", s.logoutAll)
	auth.Post("/change-password", s.changePassword)
}

// login handles POST /api/v1/auth/login - authenticate user
func (s *Server) login(c *fiber.Ctx) error {
	var req domain.LoginRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Capture client info
	req.UserAgent = string(c.Request().Header.UserAgent())
	req.IPAddress = c.IP()

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	response, err := s.authConnector.Login(c.UserContext(), req)
	if err != nil {
		slog.Error("failed to login", "error", err)
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid credentials",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// refreshToken handles POST /api/v1/auth/refresh - refresh access token
func (s *Server) refreshToken(c *fiber.Ctx) error {
	var req domain.RefreshTokenRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Capture client info
	req.UserAgent = string(c.Request().Header.UserAgent())
	req.IPAddress = c.IP()

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	response, err := s.authConnector.RefreshToken(c.UserContext(), req)
	if err != nil {
		slog.Error("failed to refresh token", "error", err)
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid or expired refresh token",
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// validateToken handles POST /api/v1/auth/validate - validate access token
func (s *Server) validateToken(c *fiber.Ctx) error {
	var req domain.ValidateTokenRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	response, err := s.authConnector.ValidateToken(c.UserContext(), req.Token)
	if err != nil {
		slog.Error("failed to validate token", "error", err)
		return c.Status(http.StatusOK).JSON(domain.ValidateTokenResponse{
			Valid: false,
		})
	}

	return c.Status(http.StatusOK).JSON(response)
}

// logout handles POST /api/v1/auth/logout - logout from current device
func (s *Server) logout(c *fiber.Ctx) error {
	var req domain.LogoutRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	// First validate the token to get user ID
	validateResp, err := s.authConnector.ValidateToken(c.UserContext(), req.Token)
	if err != nil || !validateResp.Valid {
		slog.Error("invalid token for logout", "error", err)
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid token",
		})
	}

	if err := s.authConnector.Logout(c.UserContext(), validateResp.UserID, req.Token); err != nil {
		slog.Error("failed to logout", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to logout",
		})
	}

	return c.Status(http.StatusOK).JSON(domain.SuccessResponse{
		Message: "Logged out successfully",
	})
}

// logoutAll handles POST /api/v1/auth/logout-all - logout from all devices
func (s *Server) logoutAll(c *fiber.Ctx) error {
	var req domain.LogoutAllRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	// First validate the token to get user ID
	validateResp, err := s.authConnector.ValidateToken(c.UserContext(), req.Token)
	if err != nil || !validateResp.Valid {
		slog.Error("invalid token for logout all", "error", err)
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid token",
		})
	}

	if err := s.authConnector.LogoutAll(c.UserContext(), validateResp.UserID, req.Token); err != nil {
		slog.Error("failed to logout all", "error", err)
		return c.Status(http.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to logout from all devices",
		})
	}

	return c.Status(http.StatusOK).JSON(domain.SuccessResponse{
		Message: "Logged out from all devices successfully",
	})
}

// changePassword handles POST /api/v1/auth/change-password - change user password
func (s *Server) changePassword(c *fiber.Ctx) error {
	var req domain.ChangePasswordRequest

	if err := c.BodyParser(&req); err != nil {
		slog.Error("failed to parse request body", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	// First validate the token to get user ID
	validateResp, err := s.authConnector.ValidateToken(c.UserContext(), req.Token)
	if err != nil || !validateResp.Valid {
		slog.Error("invalid token for change password", "error", err)
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid token",
		})
	}

	if err := s.authConnector.ChangePassword(c.UserContext(), validateResp.UserID, req); err != nil {
		slog.Error("failed to change password", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(domain.SuccessResponse{
		Message: "Password changed successfully",
	})
}

// parseValidationErrors extracts validation errors into a map
func parseValidationErrors(err error) map[string]string {
	details := make(map[string]string)
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, e := range validationErrors {
			details[e.Field()] = e.Tag()
		}
	}
	return details
}
