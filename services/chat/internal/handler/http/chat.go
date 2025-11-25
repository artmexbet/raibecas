package http

import (
	"bytes"
	"encoding/json"
	"log/slog"

	"github.com/artmexbet/raibecas/services/chat/internal/domain"
	"github.com/artmexbet/raibecas/services/chat/internal/handler/models"
	"github.com/gofiber/fiber/v2"
)

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
