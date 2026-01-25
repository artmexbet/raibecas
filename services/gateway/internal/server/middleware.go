package server

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UserContextKey is the key for storing user info in context
const UserContextKey = "user"

// AuthUser represents authenticated user data stored in context
type AuthUser struct {
	ID   uuid.UUID
	Role string
	JTI  string
}

// authMiddleware validates access token and fingerprint
func (s *Server) authMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			slog.Warn("missing authorization header")
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Authorization header required",
			})
		}

		// Check Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			slog.Warn("invalid authorization header format")
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Invalid authorization format",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			slog.Warn("empty token")
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Token is required",
			})
		}

		// Get fingerprint from cookie
		fingerprint := getSecureCookie(c, CookieFingerprint)
		if fingerprint == "" {
			slog.Warn("missing fingerprint cookie")
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Authentication fingerprint missing",
			})
		}

		// Validate token via auth service
		validationResp, err := s.authConnector.ValidateToken(c.UserContext(), token, fingerprint)
		if err != nil {
			slog.Error("token validation failed", "error", err)
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Token validation failed",
			})
		}

		if !validationResp.Valid {
			slog.Warn("invalid token")
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
		}

		// Store user info in context
		authUser := &AuthUser{
			ID:   validationResp.UserID,
			Role: validationResp.Role,
			JTI:  validationResp.JTI,
		}
		c.Locals(UserContextKey, authUser)

		slog.Debug("user authenticated", "user_id", authUser.ID, "role", authUser.Role)

		return c.Next()
	}
}

// getAuthUser retrieves authenticated user from context
func getAuthUser(c *fiber.Ctx) (*AuthUser, bool) {
	user, ok := c.Locals(UserContextKey).(*AuthUser)
	return user, ok
}
