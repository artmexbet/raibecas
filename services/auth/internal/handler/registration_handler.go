package handler

import (
	"time"

	"auth/internal/nats"
	"auth/internal/service"

	"github.com/gofiber/fiber/v2"
)

// RegistrationHandler handles registration HTTP requests
type RegistrationHandler struct {
	regService *service.RegistrationService
	publisher  *nats.Publisher
}

// NewRegistrationHandler creates a new registration handler
func NewRegistrationHandler(regService *service.RegistrationService, publisher *nats.Publisher) *RegistrationHandler {
	return &RegistrationHandler{
		regService: regService,
		publisher:  publisher,
	}
}

// RegisterRequest represents a registration request body
type RegisterRequest struct {
	Username string         `json:"username" validate:"required,min=3,max=50"`
	Email    string         `json:"email" validate:"required,email"`
	Password string         `json:"password" validate:"required,min=8"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// RegisterResponse represents a registration response body
type RegisterResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// Register handles user registration
// POST /register
func (h *RegistrationHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	registerReq := service.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Metadata: req.Metadata,
	}

	requestID, err := h.regService.CreateRegistrationRequest(c.Context(), registerReq)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Publish registration requested event
	_ = h.publisher.PublishRegistrationRequested(nats.RegistrationRequestedEvent{
		RequestID: requestID,
		Username:  req.Username,
		Email:     req.Email,
		Metadata:  req.Metadata,
		Timestamp: time.Now(),
	})

	return c.Status(fiber.StatusAccepted).JSON(RegisterResponse{
		RequestID: requestID.String(),
		Status:    "pending",
		Message:   "Registration request submitted successfully. Waiting for admin approval.",
	})
}
