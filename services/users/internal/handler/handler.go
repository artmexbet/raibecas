package handler

import (
	"errors"
	"log/slog"

	"github.com/artmexbet/raibecas/libs/dto"
	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/mapper"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
	"github.com/artmexbet/raibecas/services/users/internal/service"
)

type Handler struct {
	service ServiceInterface
}

func New(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

// respondError sends error response using easyjson
func (h *Handler) respondError(msg *natsw.Message, errCode dto.ErrorCode) error {
	resp := &dto.ErrorResponse{
		Success: false,
		Error:   string(errCode),
	}
	return msg.RespondEasyJSON(resp)
}

// Users

func (h *Handler) HandleListUsers(msg *natsw.Message) error {
	var req dto.ListUsersRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid list users request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
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
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	resp := &dto.ListUsersResponse{
		Users:      mapper.UsersToDTO(users),
		TotalCount: total,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	return msg.RespondEasyJSON(resp)
}

func (h *Handler) HandleGetUser(msg *natsw.Message) error {
	var req dto.GetUserRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid get user request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	user, err := h.service.GetUserByID(msg.Ctx, req.ID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.DebugContext(msg.Ctx, "user not found", "user_id", req.ID)
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		slog.ErrorContext(msg.Ctx, "failed to get user", "user_id", req.ID, "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	resp := &dto.GetUserResponse{
		User: *mapper.UserToDTO(user),
	}

	return msg.RespondEasyJSON(resp)
}

func (h *Handler) HandleUpdateUser(msg *natsw.Message) error {
	var req dto.UpdateUserRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid update user request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Validate role if provided
	if req.Updates.Role != nil && *req.Updates.Role != "" {
		if !domain.IsValidRole(*req.Updates.Role) {
			slog.DebugContext(msg.Ctx, "invalid role", "role", req.Updates.Role)
			return h.respondError(msg, dto.ErrCodeInvalidRequest)
		}
	}

	user, err := h.service.UpdateUser(msg.Ctx, postgres.UpdateUserParams{
		ID:       req.ID,
		Email:    req.Updates.Email,
		Username: req.Updates.Username,
		FullName: req.Updates.FullName,
		Role:     req.Updates.Role,
		IsActive: req.Updates.IsActive,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.DebugContext(msg.Ctx, "user not found", "user_id", req.ID)
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		slog.ErrorContext(msg.Ctx, "failed to update user", "user_id", req.ID, "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	resp := &dto.UpdateUserResponse{
		User: *mapper.UserToDTO(user),
	}

	return msg.RespondEasyJSON(resp)
}

func (h *Handler) HandleDeleteUser(msg *natsw.Message) error {
	var req dto.DeleteUserRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid delete user request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	err := h.service.DeleteUser(msg.Ctx, req.ID)
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to delete user", "user_id", req.ID, "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	resp := &dto.DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
	}

	return msg.RespondEasyJSON(resp)
}

// Registration Requests

func (h *Handler) HandleCreateRegistration(msg *natsw.Message) error {
	var req dto.CreateRegistrationRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid create registration request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Validate required fields
	if req.Email == "" || req.Username == "" || req.Password == "" {
		slog.DebugContext(msg.Ctx, "missing required registration fields")
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Convert metadata from dto to domain
	metadata := make(domain.Metadata)
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	newReq := &domain.RegistrationRequest{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: req.Password, // Service will hash it
		Metadata:     metadata,
	}

	createdReq, err := h.service.CreateRegistrationRequest(msg.Ctx, newReq)
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to create registration request", "email", req.Email, "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	resp := &dto.CreateRegistrationResponse{
		RequestID: createdReq.ID,
		Status:    string(createdReq.Status),
		Message:   "Registration request submitted successfully. Waiting for admin approval.",
	}

	return msg.RespondEasyJSON(resp)
}

func (h *Handler) HandleListRegistrations(msg *natsw.Message) error {
	var req dto.ListRegistrationsRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid list registrations request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
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

	// Convert dto.RegistrationStatus to domain.RegistrationStatus
	var status domain.RegistrationStatus
	if req.Status != "" {
		status = domain.RegistrationStatus(req.Status)
	}

	reqs, total, err := h.service.ListRegistrationRequests(msg.Ctx, status, limit, offset)
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to list registration requests", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	resp := &dto.ListRegistrationsResponse{
		Requests:   mapper.RegistrationsToDTO(reqs),
		TotalCount: total,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	return msg.RespondEasyJSON(resp)
}

func (h *Handler) HandleApproveRegistration(msg *natsw.Message) error {
	var req dto.ApproveRegistrationRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid approve registration request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Validate role if provided
	if req.Role != "" && !domain.IsValidRole(req.Role) {
		slog.DebugContext(msg.Ctx, "invalid role", "role", req.Role)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	user, err := h.service.ApproveRegistrationRequest(msg.Ctx, req.RequestID, req.ApproverID, req.Role)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			slog.DebugContext(msg.Ctx, "registration request not found", "request_id", req.RequestID)
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		slog.ErrorContext(msg.Ctx, "failed to approve registration request", "request_id", req.RequestID, "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	resp := &dto.ApproveRegistrationResponse{
		Success: true,
		Message: "Registration request approved successfully",
		User:    mapper.UserToDTO(user),
	}

	return msg.RespondEasyJSON(resp)
}

func (h *Handler) HandleRejectRegistration(msg *natsw.Message) error {
	var req dto.RejectRegistrationRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid reject registration request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	err := h.service.RejectRegistrationRequest(msg.Ctx, req.RequestID, req.ApproverID, req.Reason)
	if err != nil {
		slog.ErrorContext(msg.Ctx, "failed to reject registration request", "request_id", req.RequestID, "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	resp := &dto.RejectRegistrationResponse{
		Success: true,
		Message: "Registration request rejected.",
	}

	return msg.RespondEasyJSON(resp)
}
