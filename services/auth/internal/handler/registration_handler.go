package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

type RegistrationService interface {
	CreateRegistrationRequest(context.Context, domain.RegisterRequest) (uuid.UUID, error)
}

// RegistrationHandler handles registration NATS requests
type RegistrationHandler struct {
	regService RegistrationService
	publisher  EventPublisher
}

// NewRegistrationHandler creates a new NATS registration handler
func NewRegistrationHandler(regService RegistrationService, publisher EventPublisher) *RegistrationHandler {
	return &RegistrationHandler{
		regService: regService,
		publisher:  publisher,
	}
}

// HandleRegister handles registration requests via NATS
func (h *RegistrationHandler) HandleRegister(msg *natsw.Message) error {
	var req RegisterRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	regReq := req.ToDomain()

	requestID, err := h.regService.CreateRegistrationRequest(ctx, regReq)
	if err != nil {
		return h.respondError(msg, err.Error())
	}

	// Publish registration requested event
	_ = h.publisher.PublishRegistrationRequested(ctx, domain.RegistrationRequestedEvent{
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

	return h.respond(msg, response)
}

// Helper methods
func (h *RegistrationHandler) respond(msg *natsw.Message, data any) error {
	resp := Response{
		Success: true,
		Data:    data,
	}

	return msg.RespondJSON(resp)
}

func (h *RegistrationHandler) respondError(msg *natsw.Message, errorMsg string) error {
	resp := Response{
		Success: false,
		Error:   errorMsg,
	}

	if err := msg.RespondJSON(resp); err != nil {
		slog.ErrorContext(msg.Ctx, "Failed to send error response", "error", err)
		return err
	}
	return nil
}
