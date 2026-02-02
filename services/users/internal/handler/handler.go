package handler

import (
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
	"github.com/artmexbet/raibecas/services/users/internal/service"
)

//go:generate easyjson handler.go

type Handler struct {
	service *service.Service
}

func New(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) respondError(msg *natsw.Message, errCode string) error {
	return msg.RespondJSON(map[string]interface{}{
		"success": false,
		"error":   errCode,
	})
}

func (h *Handler) respond(msg *natsw.Message, data interface{}) error {
	return msg.RespondJSON(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// Users

//easyjson:json
type ListUsersRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Search   string `json:"search"`
	IsActive *bool  `json:"is_active"`
}

func (h *Handler) HandleListUsers(msg *natsw.Message) error {
	var req ListUsersRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid list users request", "error", err)
		return h.respondError(msg, "invalid_request")
	}

	// Validate pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}

	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize

	users, total, err := h.service.ListUsers(msg.Ctx, postgres.ListUsersParams{
		Limit:    limit,
		Offset:   offset,
		Search:   req.Search,
		IsActive: req.IsActive,
	})
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to list users", "error", err)
		return h.respondError(msg, "internal_error")
	}

	return h.respond(msg, map[string]interface{}{
		"users":       users,
		"total_count": total,
		"page":        req.Page,
		"page_size":   req.PageSize,
	})
}

//easyjson:json
type GetUserRequest struct {
	ID uuid.UUID `json:"id"`
}

func (h *Handler) HandleGetUser(msg *natsw.Message) error {
	var req GetUserRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid get user request", "error", err)
		return h.respondError(msg, "invalid_request")
	}

	user, err := h.service.GetUserByID(msg.Ctx, req.ID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.DebugContext(msg.Ctx, "user not found", "user_id", req.ID)
			return h.respondError(msg, "not_found")
		}
		slog.ErrorContext(msg.Ctx, "failed to get user", "user_id", req.ID, "error", err)
		return h.respondError(msg, "internal_error")
	}

	return h.respond(msg, map[string]interface{}{
		"user": user,
	})
}

//easyjson:json
type UpdateUserPayload struct {
	Email    *string `json:"email"`
	Username *string `json:"username"`
	FullName *string `json:"full_name"`
	IsActive *bool   `json:"is_active"`
}

//easyjson:json
type UpdateUserRequest struct {
	ID      uuid.UUID         `json:"id"`
	Updates UpdateUserPayload `json:"updates"`
}

func (h *Handler) HandleUpdateUser(msg *natsw.Message) error {
	var req UpdateUserRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid update user request", "error", err)
		return h.respondError(msg, "invalid_request")
	}

	user, err := h.service.UpdateUser(msg.Ctx, postgres.UpdateUserParams{
		ID:       req.ID,
		Email:    req.Updates.Email,
		Username: req.Updates.Username,
		FullName: req.Updates.FullName,
		IsActive: req.Updates.IsActive,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.DebugContext(msg.Ctx, "user not found", "user_id", req.ID)
			return h.respondError(msg, "not_found")
		}
		slog.ErrorContext(msg.Ctx, "failed to update user", "user_id", req.ID, "error", err)
		return h.respondError(msg, "internal_error")
	}

	return h.respond(msg, map[string]interface{}{
		"user": user,
	})
}

//easyjson:json
type DeleteUserRequest struct {
	ID uuid.UUID `json:"id"`
}

func (h *Handler) HandleDeleteUser(msg *natsw.Message) error {
	var req DeleteUserRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid delete user request", "error", err)
		return h.respondError(msg, "invalid_request")
	}

	err := h.service.DeleteUser(msg.Ctx, req.ID)
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to delete user", "user_id", req.ID, "error", err)
		return h.respondError(msg, "internal_error")
	}

	return h.respond(msg, nil)
}

// Registration Requests

//easyjson:json
type CreateRegistrationRequest struct {
	Username string          `json:"username"`
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Metadata domain.Metadata `json:"metadata"`
}

func (h *Handler) HandleCreateRegistration(msg *natsw.Message) error {
	var req CreateRegistrationRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid create registration request", "error", err)
		return h.respondError(msg, "invalid_request")
	}

	// Validate required fields
	if req.Email == "" || req.Username == "" || req.Password == "" {
		slog.DebugContext(msg.Ctx, "missing required registration fields")
		return h.respondError(msg, "invalid_request")
	}

	newReq := &domain.RegistrationRequest{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: req.Password, // Just passing password to service to hash
		Metadata:     req.Metadata,
	}

	createdReq, err := h.service.CreateRegistrationRequest(msg.Ctx, newReq)
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to create registration request", "email", req.Email, "error", err)
		return h.respondError(msg, "internal_error")
	}

	return h.respond(msg, map[string]interface{}{
		"request_id": createdReq.ID,
		"status":     createdReq.Status,
		"message":    "Registration request submitted successfully. Waiting for admin approval.",
	})
}

//easyjson:json
type ListRegistrationsRequest struct {
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
	Status   domain.RegistrationStatus `json:"status"`
}

func (h *Handler) HandleListRegistrations(msg *natsw.Message) error {
	var req ListRegistrationsRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid list registrations request", "error", err)
		return h.respondError(msg, "invalid_request")
	}

	// Validate pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}

	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize

	reqs, total, err := h.service.ListRegistrationRequests(msg.Ctx, req.Status, limit, offset)
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to list registration requests", "error", err)
		return h.respondError(msg, "internal_error")
	}

	return h.respond(msg, map[string]interface{}{
		"requests":    reqs,
		"total_count": total,
		"page":        req.Page,
		"page_size":   req.PageSize,
	})
}

//easyjson:json
type ApproveRegistrationRequest struct {
	RequestID  uuid.UUID `json:"request_id"`
	ApproverID uuid.UUID `json:"approver_id"`
}

func (h *Handler) HandleApproveRegistration(msg *natsw.Message) error {
	var req ApproveRegistrationRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid approve registration request", "error", err)
		return h.respondError(msg, "invalid_request")
	}

	user, err := h.service.ApproveRegistrationRequest(msg.Ctx, req.RequestID, req.ApproverID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.DebugContext(msg.Ctx, "registration request not found", "request_id", req.RequestID)
			return h.respondError(msg, "not_found")
		}
		slog.ErrorContext(msg.Ctx, "failed to approve registration request", "request_id", req.RequestID, "error", err)
		return h.respondError(msg, "internal_error")
	}

	return h.respond(msg, map[string]interface{}{
		"user": user,
	})
}

//easyjson:json
type RejectRegistrationRequest struct {
	RequestID  uuid.UUID `json:"request_id"`
	ApproverID uuid.UUID `json:"approver_id"`
	Reason     string    `json:"reason"`
}

func (h *Handler) HandleRejectRegistration(msg *natsw.Message) error {
	var req RejectRegistrationRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid reject registration request", "error", err)
		return h.respondError(msg, "invalid_request")
	}

	err := h.service.RejectRegistrationRequest(msg.Ctx, req.RequestID, req.ApproverID, req.Reason)
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to reject registration request", "request_id", req.RequestID, "error", err)
		return h.respondError(msg, "internal_error")
	}

	return h.respond(msg, map[string]interface{}{
		"success": true,
		"message": "Registration request rejected.",
	})
}
