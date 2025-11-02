package nats

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"auth/internal/nats"
	"auth/internal/service"

	"github.com/google/uuid"
	natspkg "github.com/nats-io/nats.go"
)

// RegistrationHandler handles registration NATS requests
type RegistrationHandler struct {
	regService *service.RegistrationService
	publisher  *nats.Publisher
}

// NewRegistrationHandler creates a new NATS registration handler
func NewRegistrationHandler(regService *service.RegistrationService, publisher *nats.Publisher) *RegistrationHandler {
	return &RegistrationHandler{
		regService: regService,
		publisher:  publisher,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string                 `json:"username"`
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterResponse represents a registration response
type RegisterResponse struct {
	RequestID uuid.UUID `json:"request_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

// HandleRegister handles registration requests via NATS
func (h *RegistrationHandler) HandleRegister(msg *natspkg.Msg) {
	var req RegisterRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	ctx := context.Background()
	regReq := service.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Metadata: req.Metadata,
	}

	requestID, err := h.regService.CreateRegistrationRequest(ctx, regReq)
	if err != nil {
		h.respondError(msg, err.Error())
		return
	}

	// Publish registration requested event
	_ = h.publisher.PublishRegistrationRequested(nats.RegistrationRequestedEvent{
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

	h.respond(msg, response)
}

// Helper methods
func (h *RegistrationHandler) respond(msg *natspkg.Msg, data interface{}) {
	response, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	if err := msg.Respond(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func (h *RegistrationHandler) respondError(msg *natspkg.Msg, errorMsg string) {
	h.respond(msg, ErrorResponse{Error: errorMsg})
}
