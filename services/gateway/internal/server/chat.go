package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// handleWebSocketChat handles WebSocket connections for chat streaming
// It bridges the client connection directly to the chat service via WebSocket
func (s *Server) handleWebSocketChat(c *websocket.Conn) {
	userID := c.Params("userID")
	if userID == "" {
		slog.Error("WebSocket connection without userID")
		c.Close() //nolint:errcheck // safe to ignore
		return
	}

	slog.Info("WebSocket chat connection established", "user_id", userID)

	// Create context for this connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to chat service and bridge connections
	err := s.chatConnector.Connect(ctx, c, userID)
	if err != nil {
		slog.Error("Failed to connect to chat service", "user_id", userID, "error", err)
		c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","error":"failed to connect to chat service"}`)) //nolint:errcheck // safe to ignore
		c.Close()                                                                                                     //nolint:errcheck // safe to ignore
		return
	}
}

// WebSocketUpgradeHandler upgrades HTTP connection to WebSocket
func (s *Server) WebSocketUpgradeHandler(c *fiber.Ctx) error {
	// Check authentication
	// TODO: Implement auth check for WebSocket

	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// getChatSessions proxies GET /api/v1/chat/:userID/sessions to the chat service
func (s *Server) getChatSessions(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if userID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "userID is required"})
	}

	sessions, err := s.chatHTTPConnector.GetUserSessions(c.UserContext(), userID)
	if err != nil {
		slog.Error("failed to get chat sessions", "user_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get chat sessions"})
	}

	return c.Status(http.StatusOK).JSON(sessions)
}

// createChatSession proxies POST /api/v1/chat/:userID/sessions to the chat service
func (s *Server) createChatSession(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if userID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "userID is required"})
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	sessionID, err := s.chatHTTPConnector.CreateSession(c.UserContext(), userID, body.Title)
	if err != nil {
		slog.Error("failed to create chat session", "user_id", userID, "error", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create chat session"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"session_id": sessionID})
}
