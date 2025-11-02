package handler

import (
	"time"

	"auth/internal/nats"
	"auth/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService *service.AuthService
	publisher   *nats.Publisher
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, publisher *nats.Publisher) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		publisher:   publisher,
	}
}

// LoginRequest represents a login request body
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response body
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Login handles user login
// POST /login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get client metadata
	deviceID := c.Get("X-Device-ID", "")
	userAgent := c.Get("User-Agent", "")
	ipAddress := c.IP()

	loginReq := service.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		DeviceID:  deviceID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	tokens, userID, err := h.authService.Login(c.Context(), loginReq)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Publish login event
	_ = h.publisher.PublishUserLogin(nats.UserLoginEvent{
		UserID:    userID,
		DeviceID:  deviceID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		Timestamp: time.Now(),
	})

	return c.JSON(LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
	})
}

// LogoutRequest represents a logout request body
type LogoutRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

// Logout handles user logout
// POST /logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Get user ID from token/context (middleware should set this)
	userID := c.Locals("user_id").(uuid.UUID)

	if err := h.authService.Logout(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to logout",
		})
	}

	// Publish logout event
	_ = h.publisher.PublishUserLogout(nats.UserLogoutEvent{
		UserID:    userID,
		Timestamp: time.Now(),
	})

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// LogoutAll handles user logout from all devices
// POST /logout-all
func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	// Get user ID from token/context (middleware should set this)
	userID := c.Locals("user_id").(uuid.UUID)

	if err := h.authService.LogoutAll(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to logout from all devices",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Logged out from all devices successfully",
	})
}

// ValidateRequest represents a token validation request
type ValidateRequest struct {
	Token string `json:"token" validate:"required"`
}

// ValidateResponse represents a token validation response
type ValidateResponse struct {
	Valid  bool      `json:"valid"`
	UserID uuid.UUID `json:"user_id,omitempty"`
	Role   string    `json:"role,omitempty"`
}

// Validate handles token validation
// POST /validate
func (h *AuthHandler) Validate(c *fiber.Ctx) error {
	var req ValidateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	claims, err := h.authService.ValidateAccessToken(c.Context(), req.Token)
	if err != nil {
		return c.JSON(ValidateResponse{
			Valid: false,
		})
	}

	return c.JSON(ValidateResponse{
		Valid:  true,
		UserID: claims.UserID,
		Role:   claims.Role,
	})
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// RefreshRequest represents a token refresh request body
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Refresh handles token refresh
// POST /refresh
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get client metadata
	deviceID := c.Get("X-Device-ID", "")
	userAgent := c.Get("User-Agent", "")
	ipAddress := c.IP()

	refreshReq := service.RefreshRequest{
		RefreshToken: req.RefreshToken,
		DeviceID:     deviceID,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
	}

	tokens, _, err := h.authService.RefreshTokens(c.Context(), refreshReq)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired refresh token",
		})
	}

	return c.JSON(LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
	})
}

// ChangePassword handles password change
// POST /change-password
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	// Get user ID from token/context (middleware should set this)
	userID := c.Locals("user_id").(uuid.UUID)

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	changeReq := service.ChangePasswordRequest{
		UserID:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	if err := h.authService.ChangePassword(c.Context(), changeReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Publish password reset event
	_ = h.publisher.PublishPasswordReset(nats.PasswordResetEvent{
		UserID:    userID,
		Method:    "self-service",
		Timestamp: time.Now(),
	})

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}
