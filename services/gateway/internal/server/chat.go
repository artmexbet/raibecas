package server

import (
	"context"
	"log/slog"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// handleWebSocketChat handles WebSocket connections for chat streaming
// It bridges the client connection directly to the chat service via WebSocket
func (s *Server) handleWebSocketChat(c *websocket.Conn) {
	userID := c.Params("userID")
	if userID == "" {
		slog.Error("WebSocket connection without userID")
		c.Close()
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
		c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","error":"failed to connect to chat service"}`))
		c.Close()
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
