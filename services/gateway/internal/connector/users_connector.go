package connector

import (
	"context"
	"fmt"

	"github.com/artmexbet/raibecas/libs/utils/pointer"
	"github.com/google/uuid"
	"github.com/mailru/easyjson"
	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/dto"
	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// NATS subjects for users service communication
const (
	SubjectUsersList   = "users.list"
	SubjectUsersGet    = "users.get"
	SubjectUsersUpdate = "users.update"
	SubjectUsersDelete = "users.delete"

	// Registration requests
	SubjectRegistrationRequestCreate  = "users.registration.create"
	SubjectRegistrationRequestList    = "users.registration.list"
	SubjectRegistrationRequestApprove = "users.registration.approve"
	SubjectRegistrationRequestReject  = "users.registration.reject"
)

// NATSUserConnector implements server.UserServiceConnector using NATS for communication
type NATSUserConnector struct {
	client *natsw.Client
}

// NewNATSUserConnector creates a new NATS-based users service connector
func NewNATSUserConnector(client *natsw.Client) *NATSUserConnector {
	return &NATSUserConnector{
		client: client,
	}
}

// ListUsers retrieves a list of users based on query parameters
func (c *NATSUserConnector) ListUsers(ctx context.Context, query domain.ListUsersQuery) (*domain.ListUsersResponse, error) {
	// Convert domain query to dto request
	req := &dto.ListUsersRequest{
		Page:     query.Page,
		PageSize: query.PageSize,
		Search:   query.Search,
		IsActive: query.IsActive,
	}

	reqData, err := easyjson.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list users request: %w", err)
	}

	msg := nats.NewMsg(SubjectUsersList)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list users request: %w", err)
	}

	var listResp dto.ListUsersResponse
	if err := easyjson.Unmarshal(respMsg.Data, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list users response: %w", err)
	}

	// Convert dto response to domain response
	users := make([]domain.User, len(listResp.Users))
	for i, u := range listResp.Users {
		users[i] = domain.User{
			ID:           u.ID,
			Email:        u.Email,
			Username:     u.Username,
			FullName:     u.FullName,
			RegisteredAt: u.CreatedAt,
			LastLoginAt:  u.LastLoginAt,
			IsActive:     u.IsActive,
		}
	}

	return &domain.ListUsersResponse{
		Users:      users,
		TotalCount: listResp.TotalCount,
		Page:       listResp.Page,
		PageSize:   listResp.PageSize,
	}, nil
}

// GetUser retrieves a single user by ID
func (c *NATSUserConnector) GetUser(ctx context.Context, id uuid.UUID) (*domain.GetUserResponse, error) {
	req := &dto.GetUserRequest{ID: id}
	reqData, err := easyjson.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get user request: %w", err)
	}

	msg := nats.NewMsg(SubjectUsersGet)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send get user request: %w", err)
	}

	var getResp dto.GetUserResponse
	if err := easyjson.Unmarshal(respMsg.Data, &getResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get user response: %w", err)
	}

	// Convert dto to domain
	return &domain.GetUserResponse{
		User: domain.User{
			ID:           getResp.User.ID,
			Email:        getResp.User.Email,
			Username:     getResp.User.Username,
			FullName:     getResp.User.FullName,
			RegisteredAt: getResp.User.CreatedAt,
			LastLoginAt:  getResp.User.LastLoginAt,
			IsActive:     getResp.User.IsActive,
		},
	}, nil
}

// UpdateUser updates an existing user
func (c *NATSUserConnector) UpdateUser(ctx context.Context, id uuid.UUID, req domain.UpdateUserRequest) (*domain.UpdateUserResponse, error) {
	dtoReq := &dto.UpdateUserRequest{
		ID: id,
		Updates: dto.UpdateUserPayload{
			Email:    req.Email,
			Username: req.Username,
			FullName: req.FullName,
			IsActive: req.IsActive,
			Role:     pointer.To(string(req.Role)),
		},
	}

	reqData, err := easyjson.Marshal(dtoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update user request: %w", err)
	}

	msg := nats.NewMsg(SubjectUsersUpdate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send update user request: %w", err)
	}

	var updateResp dto.UpdateUserResponse
	if err := easyjson.Unmarshal(respMsg.Data, &updateResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update user response: %w", err)
	}

	// Convert dto to domain
	return &domain.UpdateUserResponse{
		User: domain.User{
			ID:           updateResp.User.ID,
			Email:        updateResp.User.Email,
			Username:     updateResp.User.Username,
			FullName:     updateResp.User.FullName,
			RegisteredAt: updateResp.User.CreatedAt,
			LastLoginAt:  updateResp.User.LastLoginAt,
			IsActive:     updateResp.User.IsActive,
		},
	}, nil
}

// DeleteUser deletes a user by ID
func (c *NATSUserConnector) DeleteUser(ctx context.Context, id uuid.UUID) error {
	req := &dto.DeleteUserRequest{ID: id}
	reqData, err := easyjson.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal delete user request: %w", err)
	}

	msg := nats.NewMsg(SubjectUsersDelete)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send delete user request: %w", err)
	}

	var delResp dto.DeleteUserResponse
	if err := easyjson.Unmarshal(respMsg.Data, &delResp); err != nil {
		return fmt.Errorf("failed to unmarshal delete user response: %w", err)
	}

	if !delResp.Success {
		return fmt.Errorf("delete user failed")
	}

	return nil
}

// CreateRegistrationRequest creates a new registration request
func (c *NATSUserConnector) CreateRegistrationRequest(ctx context.Context, req domain.CreateRegistrationRequestRequest) (*domain.CreateRegistrationRequestResponse, error) {
	dtoReq := &dto.CreateRegistrationRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Metadata: req.Metadata,
	}

	reqData, err := easyjson.Marshal(dtoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create registration request: %w", err)
	}

	msg := nats.NewMsg(SubjectRegistrationRequestCreate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create registration request: %w", err)
	}

	var createResp dto.CreateRegistrationResponse
	if err := easyjson.Unmarshal(respMsg.Data, &createResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create registration response: %w", err)
	}

	return &domain.CreateRegistrationRequestResponse{
		RequestID: createResp.RequestID,
		Status:    createResp.Status,
		Message:   createResp.Message,
	}, nil
}

// ListRegistrationRequests retrieves a list of registration requests
func (c *NATSUserConnector) ListRegistrationRequests(ctx context.Context, query domain.ListRegistrationRequestsQuery) (*domain.ListRegistrationRequestsResponse, error) {
	dtoReq := &dto.ListRegistrationsRequest{
		Page:     query.Page,
		PageSize: query.PageSize,
		Status:   dto.RegistrationStatus(query.Status),
	}

	reqData, err := easyjson.Marshal(dtoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list registration requests: %w", err)
	}

	msg := nats.NewMsg(SubjectRegistrationRequestList)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list registration requests: %w", err)
	}

	var listResp dto.ListRegistrationsResponse
	if err := easyjson.Unmarshal(respMsg.Data, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list registration response: %w", err)
	}

	// Convert dto to domain
	requests := make([]domain.RegistrationRequest, len(listResp.Requests))
	for i, r := range listResp.Requests {
		requests[i] = domain.RegistrationRequest{
			ID:         r.ID,
			Username:   r.Username,
			Email:      r.Email,
			Status:     domain.RegistrationStatus(r.Status),
			Metadata:   r.Metadata,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			ApprovedBy: r.ApprovedBy,
			ApprovedAt: r.ApprovedAt,
		}
	}

	return &domain.ListRegistrationRequestsResponse{
		Requests:   requests,
		TotalCount: listResp.TotalCount,
		Page:       listResp.Page,
		PageSize:   listResp.PageSize,
	}, nil
}

// ApproveRegistrationRequest approves a registration request
func (c *NATSUserConnector) ApproveRegistrationRequest(ctx context.Context, requestID, approverID uuid.UUID, role string) (*domain.ApproveRegistrationRequestResponse, error) {
	dtoReq := &dto.ApproveRegistrationRequest{
		RequestID:  requestID,
		ApproverID: approverID,
		Role:       role,
	}

	reqData, err := easyjson.Marshal(dtoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal approve request: %w", err)
	}

	msg := nats.NewMsg(SubjectRegistrationRequestApprove)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send approve request: %w", err)
	}

	var approveResp dto.ApproveRegistrationResponse
	if err := easyjson.Unmarshal(respMsg.Data, &approveResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approve response: %w", err)
	}

	// Convert user if present
	var user *domain.User
	if approveResp.User != nil {
		user = &domain.User{
			ID:           approveResp.User.ID,
			Email:        approveResp.User.Email,
			Username:     approveResp.User.Username,
			FullName:     approveResp.User.FullName,
			RegisteredAt: approveResp.User.CreatedAt,
			LastLoginAt:  approveResp.User.LastLoginAt,
			IsActive:     approveResp.User.IsActive,
		}
	}

	return &domain.ApproveRegistrationRequestResponse{
		Success: approveResp.Success,
		Message: approveResp.Message,
		User:    user,
	}, nil
}

// RejectRegistrationRequest rejects a registration request
func (c *NATSUserConnector) RejectRegistrationRequest(ctx context.Context, requestID, approverID uuid.UUID, reason string) (*domain.RejectRegistrationRequestResponse, error) {
	dtoReq := &dto.RejectRegistrationRequest{
		RequestID:  requestID,
		ApproverID: approverID,
		Reason:     reason,
	}

	reqData, err := easyjson.Marshal(dtoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reject request: %w", err)
	}

	msg := nats.NewMsg(SubjectRegistrationRequestReject)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send reject request: %w", err)
	}

	var rejectResp dto.RejectRegistrationResponse
	if err := easyjson.Unmarshal(respMsg.Data, &rejectResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reject response: %w", err)
	}

	return &domain.RejectRegistrationRequestResponse{
		Success: rejectResp.Success,
		Message: rejectResp.Message,
	}, nil
}
