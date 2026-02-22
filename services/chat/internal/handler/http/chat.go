package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"github.com/artmexbet/raibecas/services/chat/internal/domain"
	"github.com/artmexbet/raibecas/services/chat/internal/handler/models"
)

// chatHandler handles HTTP chat requests (for testing)
func (h *Handler) chatHandler(c *fiber.Ctx) error {
	slog.Debug("Received chat request", slog.String("request_id", c.Get(fiber.HeaderXRequestID)))

	// Parse request body
	var req models.ChatRequest
	err := req.UnmarshalJSON(c.Body())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Set headers for streaming response
	c.Set("Content-Type", "application/x-ndjson")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Cache-Control", "no-cache")

	slog.Debug("Processing chat input", slog.String("request_id", c.Get(fiber.HeaderXRequestID)))
	buf := bytes.NewBuffer(nil)
	// Stream chunks as they arrive
	err = h.svc.ProcessInput(c.UserContext(), req.Input, req.UserID, func(response domain.ChatResponse) error {
		slog.DebugContext(c.UserContext(), "Sending chat response chunk",
			slog.String("chunk", response.Message.Content),
		)
		// Marshal the response to JSON
		data, err := json.Marshal(response)
		if err != nil {
			return err
		}
		buf.WriteString(string(data) + "\n")

		// Flush the response writer to send data immediately
		return nil
	})

	if err != nil {
		return err
	}
	return c.SendStream(buf)
}

// deleteChatHandler clears chat history
func (h *Handler) deleteChatHandler(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "userID is required"})
	}

	slog.Debug("Clearing chat history", slog.String("user_id", userID))
	err := h.svc.ClearUserChat(c.UserContext(), userID)
	if err != nil {
		slog.Error("Could not clear chat history", slog.String("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not clear chat history"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "chat history cleared"})
}

// wsChatHandler handles WebSocket chat connections
func (h *Handler) wsChatHandler(c *websocket.Conn) {
	// Get userID from query param or header
	userID := c.Query("userID")
	if userID == "" {
		slog.Error("WebSocket connection without userID")
		c.Close() //nolint:errcheck // safe to ignore error
		return
	}

	slog.Info("WebSocket chat connection established", "user_id", userID)

	// Handle messages
	for {
		msgType, data, err := c.ReadMessage()
		if err != nil {
			slog.Debug("WebSocket read error", "user_id", userID, "error", err)
			return
		}

		// Parse the message
		var req models.ChatRequest
		if err := req.UnmarshalJSON(data); err != nil {
			slog.Error("Failed to unmarshal request", "error", err)
			err = c.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid request"}`))
			if err != nil {
				slog.Error("Failed to write error message", "error", err)
			}
			continue
		}

		// Process the chat input and stream responses
		ctx := context.Background()
		err = h.svc.ProcessInput(ctx, req.Input, userID, func(response domain.ChatResponse) error {
			respData, err := json.Marshal(response)
			if err != nil {
				return err
			}
			return c.WriteMessage(msgType, respData)
		})

		if err != nil {
			slog.Error("Chat processing error", "error", err)
			err = c.WriteMessage(websocket.TextMessage, []byte(`{"error":"processing failed"}`))
			if err != nil {
				slog.Error("Failed to write error message", "error", err)
			}
		}
	}
}

// WSUpgradeHandler upgrades HTTP to WebSocket
func (h *Handler) WSUpgradeHandler(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// getUserSessionsHandler returns all chat sessions for a user
func (h *Handler) getUserSessionsHandler(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "userID is required"})
	}

	sessions, err := h.svc.GetUserSessions(c.UserContext(), userID)
	if err != nil {
		slog.Error("Could not get user sessions", slog.String("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not get sessions"})
	}

	return c.Status(fiber.StatusOK).JSON(sessions)
}

// createSessionHandler creates a new chat session for a user
func (h *Handler) createSessionHandler(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "userID is required"})
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	sessionID, err := h.svc.CreateSession(c.UserContext(), userID, body.Title)
	if err != nil {
		slog.Error("Could not create session", slog.String("error", err.Error()))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create session"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"session_id": sessionID})
}
