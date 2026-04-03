package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

// ChatRequest represents a chat request from NATS
type ChatRequest struct {
	Input     string `json:"input"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
}

// ChatResponseChunk represents a streaming response chunk
type ChatResponseChunk struct {
	Done    bool            `json:"done"`
	Message *domain.Message `json:"message,omitempty"`
}

// Service defines the chat service interface
type Service interface {
	ProcessInput(ctx context.Context, input, userID, sessionID string, fn func(response domain.ChatResponse) error) error
	ClearUserChat(ctx context.Context, userID string) error
}

// Handler handles NATS messages for chat service
type Handler struct {
	client *natsw.Client
	svc    Service
}

// NewHandler creates a new NATS handler
func NewHandler(client *natsw.Client, svc Service) *Handler {
	return &Handler{
		client: client,
		svc:    svc,
	}
}

// Subscribe subscribes to NATS subjects
func (h *Handler) Subscribe() error {
	// Subscribe to chat requests
	_, err := h.client.Subscribe("chat.request", h.handleChatRequest)
	if err != nil {
		return fmt.Errorf("failed to subscribe to chat.request: %w", err)
	}

	slog.Info("NATS handler subscribed to chat.request")
	return nil
}

// handleChatRequest processes chat requests from NATS
func (h *Handler) handleChatRequest(msg *natsw.Message) error {
	var req ChatRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		slog.Error("Failed to unmarshal chat request", "error", err)
		return err
	}

	ctx := context.Background()
	responseSubject := fmt.Sprintf("chat.response.%s", req.UserID)

	slog.Debug("Processing chat request via NATS", "user_id", req.UserID)

	// Stream responses back through NATS
	err := h.svc.ProcessInput(ctx, req.Input, req.UserID, req.SessionID, func(response domain.ChatResponse) error {
		chunk := ChatResponseChunk{
			Done:    response.Done,
			Message: response.Message,
		}

		data, err := json.Marshal(chunk)
		if err != nil {
			slog.Error("Failed to marshal response chunk", "error", err)
			return err
		}

		if err := h.client.Publish(ctx, responseSubject, data); err != nil {
			slog.Error("Failed to publish response chunk", "error", err)
			return err
		}

		return nil
	})

	if err != nil {
		slog.Error("Chat processing error", "error", err, "user_id", req.UserID)
		// Send error response
		errorChunk := ChatResponseChunk{
			Done: true,
			Message: &domain.Message{
				Role:    "system",
				Content: fmt.Sprintf("Error: %v", err),
			},
		}
		data, _ := json.Marshal(errorChunk)
		err = h.client.Publish(ctx, responseSubject, data)
		if err != nil {
			slog.Error("Failed to publish error response", "error", err)
		}
	}

	return nil
}
