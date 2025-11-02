package middleware

import (
	"strings"

	"auth/internal/service"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Extract token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		token := parts[1]

		// Validate token
		claims, err := authService.ValidateAccessToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Set user info in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_role", claims.Role)

		return c.Next()
	}
}
