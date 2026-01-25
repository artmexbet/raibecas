package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

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
	req.DeviceID = c.Get("X-Device-ID", "")

	if err := s.validator.Struct(&req); err != nil {
		slog.Error("request validation failed", "error", err)
		return c.Status(http.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Details: parseValidationErrors(err),
		})
	}

	// Call auth service
	authResp, err := s.authConnector.Login(c.UserContext(), req)
	if err != nil {
		slog.Error("failed to login", "error", err)
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid credentials",
		})
	}

	// Store refresh token and metadata in HttpOnly cookies
	setSecureCookie(c, CookieRefreshToken, authResp.RefreshToken, RefreshTokenMaxAge)
	setSecureCookie(c, CookieTokenID, authResp.TokenID, RefreshTokenMaxAge)
	setSecureCookie(c, CookieFingerprint, authResp.Fingerprint, RefreshTokenMaxAge)

	// Validate token to get user info
	userInfo, err := s.authConnector.ValidateToken(c.UserContext(), authResp.AccessToken, authResp.Fingerprint)
	if err != nil {
		slog.Error("failed to validate token for user info", "error", err)
	}

	// Return only public data to client
	publicResp := domain.LoginResponse{
		AccessToken: authResp.AccessToken,
		ExpiresIn:   authResp.ExpiresIn,
		TokenType:   "Bearer",
	}

	// Add user info if available
	if userInfo != nil && userInfo.Valid {
		publicResp.User = &domain.UserInfo{
			ID:   userInfo.UserID,
			Role: userInfo.Role,
		}
	}

	return c.Status(http.StatusOK).JSON(publicResp)
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

	// Get refresh token and metadata from cookies
	refreshToken := getSecureCookie(c, CookieRefreshToken)
	tokenID := getSecureCookie(c, CookieTokenID)
	fingerprint := getSecureCookie(c, CookieFingerprint)

	if refreshToken == "" || tokenID == "" || fingerprint == "" {
		slog.Error("missing refresh token or metadata in cookies")
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Refresh token not found",
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

	// Build internal request to auth service
	authReq := domain.AuthServiceRefreshRequest{
		RefreshToken: refreshToken,
		TokenID:      tokenID,
		Fingerprint:  fingerprint,
		DeviceID:     req.DeviceID,
		UserAgent:    req.UserAgent,
		IPAddress:    req.IPAddress,
	}

	authResp, err := s.authConnector.RefreshToken(c.UserContext(), authReq)
	if err != nil {
		slog.Error("failed to refresh token", "error", err)
		// Clear cookies on failure
		clearSecureCookie(c, CookieRefreshToken)
		clearSecureCookie(c, CookieTokenID)
		clearSecureCookie(c, CookieFingerprint)
		return c.Status(http.StatusUnauthorized).JSON(domain.ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid or expired refresh token",
		})
	}

	// Update cookies with new tokens
	setSecureCookie(c, CookieRefreshToken, authResp.RefreshToken, RefreshTokenMaxAge)
	setSecureCookie(c, CookieTokenID, authResp.TokenID, RefreshTokenMaxAge)
	setSecureCookie(c, CookieFingerprint, authResp.Fingerprint, RefreshTokenMaxAge)

	// Validate token to get user info
	userInfo, err := s.authConnector.ValidateToken(c.UserContext(), authResp.AccessToken, authResp.Fingerprint)
	if err != nil {
		slog.Error("failed to validate token for user info", "error", err)
	}

	// Return only public data to client
	publicResp := domain.LoginResponse{
		AccessToken: authResp.AccessToken,
		ExpiresIn:   authResp.ExpiresIn,
		TokenType:   "Bearer",
	}

	// Add user info if available
	if userInfo != nil && userInfo.Valid {
		publicResp.User = &domain.UserInfo{
			ID:   userInfo.UserID,
			Role: userInfo.Role,
		}
	}

	return c.Status(http.StatusOK).JSON(publicResp)
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

	// Get fingerprint from cookie (optional for validation)
	fingerprint := getSecureCookie(c, CookieFingerprint)

	response, err := s.authConnector.ValidateToken(c.UserContext(), req.Token, fingerprint)
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

	// Get fingerprint from cookie
	fingerprint := getSecureCookie(c, CookieFingerprint)

	// First validate the token to get user ID
	validateResp, err := s.authConnector.ValidateToken(c.UserContext(), req.Token, fingerprint)
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

	// Clear cookies
	clearSecureCookie(c, CookieRefreshToken)
	clearSecureCookie(c, CookieTokenID)
	clearSecureCookie(c, CookieFingerprint)

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

	// Get fingerprint from cookie
	fingerprint := getSecureCookie(c, CookieFingerprint)

	// First validate the token to get user ID
	validateResp, err := s.authConnector.ValidateToken(c.UserContext(), req.Token, fingerprint)
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

	// Clear cookies
	clearSecureCookie(c, CookieRefreshToken)
	clearSecureCookie(c, CookieTokenID)
	clearSecureCookie(c, CookieFingerprint)

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

	// Get fingerprint from cookie
	fingerprint := getSecureCookie(c, CookieFingerprint)

	// First validate the token to get user ID
	validateResp, err := s.authConnector.ValidateToken(c.UserContext(), req.Token, fingerprint)
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
