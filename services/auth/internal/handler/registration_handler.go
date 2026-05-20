package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/artmexbet/raibecas/libs/dto"
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
	tracer     trace.Tracer
}

// NewRegistrationHandler creates a new NATS registration handler
func NewRegistrationHandler(regService RegistrationService, publisher EventPublisher, tracer trace.Tracer) *RegistrationHandler {
	return &RegistrationHandler{
		regService: regService,
		publisher:  publisher,
		tracer:     tracer,
	}
}

// HandleRegister handles registration requests via NATS
func (h *RegistrationHandler) HandleRegister(msg *natsw.Message) error {
	ctx, span := h.tracer.Start(msg.Ctx, "auth.handler.register")
	defer span.End()

	var req RegisterRequest
	if err := msg.UnmarshalData(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request format")
		slog.ErrorContext(ctx, "invalid register request format", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	span.SetAttributes(
		attribute.String("auth.email", req.Email),
		attribute.String("auth.username", req.Username),
	)

	requestID, err := h.regService.CreateRegistrationRequest(ctx, req.ToDomain())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "registration failed")
		slog.ErrorContext(ctx, "registration failed", "email", req.Email, "error", err)
		return h.respondError(msg, mapDomainErrorToCode(err))
	}

	span.SetAttributes(attribute.String("auth.request_id", requestID.String()))

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

func (h *RegistrationHandler) respondError(msg *natsw.Message, errCode dto.ErrorCode) error {
	resp := Response{
		Success: false,
		Error:   string(errCode),
	}

	if err := msg.RespondJSON(resp); err != nil {
		slog.ErrorContext(msg.Ctx, "Failed to send error response", "error", err)
		return err
	}
	return nil
}
