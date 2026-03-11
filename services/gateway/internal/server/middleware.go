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

// authMiddleware validates access token from Authorization header with fingerprint from cookie
func (s *Server) authMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := s.extractBearerToken(c)
		if !ok {
			slog.Warn("missing authorization header")
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Authorization header required",
			})
		}

		fingerprint := getSecureCookie(c, CookieFingerprint)
		if fingerprint == "" {
			slog.Warn("missing fingerprint cookie")
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Authentication fingerprint missing",
			})
		}

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

// cookieAuthMiddleware validates access token from Authorization header or allows refresh flow via cookies
// This middleware is specifically for endpoints that support cookie-based refresh flow
func (s *Server) cookieAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to authenticate with access token first
		if err := s.tryAuthenticateWithAccessToken(c); err == nil {
			return c.Next()
		}

		// Fall back to refresh token cookie
		if s.hasRefreshToken(c) {
			slog.Debug("request allows cookie-based refresh flow")
			return c.Next()
		}

		slog.Warn("no valid authentication found")
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
	}
}

// tryAuthenticateWithAccessToken attempts to authenticate using Authorization header token
// Returns error if authentication fails or header is missing
func (s *Server) tryAuthenticateWithAccessToken(c *fiber.Ctx) error {
	token, ok := s.extractBearerToken(c)
	if !ok {
		return fiber.NewError(http.StatusUnauthorized, "no bearer token")
	}

	fingerprint := getSecureCookie(c, CookieFingerprint)
	if fingerprint == "" {
		slog.Warn("missing fingerprint cookie")
		return fiber.NewError(http.StatusUnauthorized, "missing fingerprint")
	}

	validationResp, err := s.authConnector.ValidateToken(c.UserContext(), token, fingerprint)
	if err != nil {
		slog.Error("token validation failed", "error", err)
		return fiber.NewError(http.StatusUnauthorized, "validation failed")
	}

	if !validationResp.Valid {
		slog.Debug("access token invalid or expired")
		return fiber.NewError(http.StatusUnauthorized, "invalid token")
	}

	// Store authenticated user in context
	authUser := &AuthUser{
		ID:   validationResp.UserID,
		Role: validationResp.Role,
		JTI:  validationResp.JTI,
	}
	c.Locals(UserContextKey, authUser)
	slog.Debug("user authenticated via access token", "user_id", authUser.ID)

	return nil
}

// extractBearerToken extracts and validates Bearer token from Authorization header
func (s *Server) extractBearerToken(c *fiber.Ctx) (string, bool) {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return "", false
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	return token, token != ""
}

// wsAuthMiddleware validates token for WebSocket connections.
// Browsers cannot set custom headers on WebSocket upgrade requests, so the
// access token is read from the ?token= query parameter (falling back to the
// Authorization header for non-browser clients).
// Fingerprint is NOT required here: the access token is already short-lived and
// transmitted directly — adding a fingerprint check would require JS to read an
// HttpOnly cookie, which is impossible by design.
func (s *Server) wsAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prefer query param, fall back to Authorization header
		token := c.Query("token")
		if token == "" {
			var ok bool
			token, ok = s.extractBearerToken(c)
			if !ok {
				slog.Warn("ws: missing token")
				return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
					"error":   "unauthorized",
					"message": "Token required (pass ?token=<access_token>)",
				})
			}
		}

		// Validate without fingerprint — browsers cannot read HttpOnly cookies in WS upgrade.
		validationResp, err := s.authConnector.ValidateTokenWS(c.UserContext(), token)
		if err != nil {
			slog.Error("ws: token validation failed", "error", err)
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Token validation failed",
			})
		}

		if !validationResp.Valid {
			slog.Warn("ws: invalid token")
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
		}

		authUser := &AuthUser{
			ID:   validationResp.UserID,
			Role: validationResp.Role,
			JTI:  validationResp.JTI,
		}
		c.Locals(UserContextKey, authUser)

		slog.Debug("ws: user authenticated", "user_id", authUser.ID)
		return c.Next()
	}
}

// hasRefreshToken checks if refresh token exists in cookies
func (s *Server) hasRefreshToken(c *fiber.Ctx) bool {
	return getSecureCookie(c, CookieRefreshToken) != ""
}

// getAuthUser retrieves authenticated user from context
func getAuthUser(c *fiber.Ctx) (*AuthUser, bool) {
	user, ok := c.Locals(UserContextKey).(*AuthUser)
	return user, ok
}

// getUserRole extracts user role from fiber context and returns it
func getUserRole(c *fiber.Ctx) string {
	if user, ok := getAuthUser(c); ok {
		return user.Role
	}
	return ""
}

// requireRole returns a middleware that allows access only to users with one of the specified roles.
// Must be used after authMiddleware (which populates UserContextKey).
func requireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		user, ok := getAuthUser(c)
		if !ok {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
		}

		if _, permitted := allowed[user.Role]; !permitted {
			slog.Warn("access denied: insufficient role",
				"user_id", user.ID,
				"user_role", user.Role,
				"required_roles", roles,
			)
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Insufficient permissions",
			})
		}

		return c.Next()
	}
}
