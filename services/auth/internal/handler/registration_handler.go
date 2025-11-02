package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"auth/internal/domain"
	"auth/internal/service"

	natspkg "github.com/nats-io/nats.go"
)

// RegistrationHandler handles registration NATS requests
type RegistrationHandler struct {
	regService *service.RegistrationService
	publisher  IEventPublisher
}

// NewRegistrationHandler creates a new NATS registration handler
func NewRegistrationHandler(regService *service.RegistrationService, publisher IEventPublisher) *RegistrationHandler {
	return &RegistrationHandler{
		regService: regService,
		publisher:  publisher,
	}
}

// HandleRegister handles registration requests via NATS
func (h *RegistrationHandler) HandleRegister(msg *natspkg.Msg) {
	var req RegisterRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	ctx := context.Background()
	regReq := req.ToDomain()

	requestID, err := h.regService.CreateRegistrationRequest(ctx, regReq)
	if err != nil {
		h.respondError(msg, err.Error())
		return
	}

	// Publish registration requested event
	_ = h.publisher.PublishRegistrationRequested(domain.RegistrationRequestedEvent{
		RequestID: requestID,
		Username:  req.Username,
		Email:     req.Email,
		Timestamp: time.Now(),
	})

	response := RegisterResponse{
		RequestID: requestID,
		Status:    "pending",
		Message:   "Registration request submitted successfully. Waiting for admin approval.",
	}

	h.respond(msg, response, nil)
}

// Helper methods
func (h *RegistrationHandler) respond(msg *natspkg.Msg, data any, err error) {
	resp := Response{
		Success: err == nil,
		Data:    data,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	response, err := json.Marshal(resp)
	if err != nil {
		slog.Error("Failed to marshal response", "error", err)
		return
	}

	if err := msg.Respond(response); err != nil {
		slog.Error("Failed to send response", "error", err)
	}
}

func (h *RegistrationHandler) respondError(msg *natspkg.Msg, errorMsg string) {
	resp := Response{
		Success: false,
		Error:   errorMsg,
	}

	response, err := json.Marshal(resp)
	if err != nil {
		slog.Error("Failed to marshal error response", "error", err)
		return
	}

	if err := msg.Respond(response); err != nil {
		slog.Error("Failed to send error response", "error", err)
	}
}
