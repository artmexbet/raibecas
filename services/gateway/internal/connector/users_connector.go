package connector

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

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

// usersResponse represents a generic NATS response from users service
type usersResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// ListUsers retrieves a list of users based on query parameters
func (c *NATSUserConnector) ListUsers(ctx context.Context, query domain.ListUsersQuery) (*domain.ListUsersResponse, error) {
	reqData, err := query.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list users request: %w", err)
	}

	msg := nats.NewMsg(SubjectUsersList)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list users request: %w", err)
	}

	var response usersResponse
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list users response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("list users failed: %s", response.Error)
	}

	var listResp domain.ListUsersResponse
	if err := json.Unmarshal(response.Data, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list users data: %w", err)
	}

	return &listResp, nil
}

// GetUser retrieves a single user by ID
func (c *NATSUserConnector) GetUser(ctx context.Context, id uuid.UUID) (*domain.GetUserResponse, error) {
	req := domain.GetUserRequest{ID: id}
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get user request: %w", err)
	}

	msg := nats.NewMsg(SubjectUsersGet)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send get user request: %w", err)
	}

	var response usersResponse
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get user response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("get user failed: %s", response.Error)
	}

	var getResp domain.GetUserResponse
	if err := json.Unmarshal(response.Data, &getResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get user data: %w", err)
	}

	return &getResp, nil
}

// UpdateUser updates an existing user
func (c *NATSUserConnector) UpdateUser(ctx context.Context, id uuid.UUID, req domain.UpdateUserRequest) (*domain.UpdateUserResponse, error) {
	reqPayload := UpdateUserRequestWrapper{
		ID: id,
		Updates: UpdateUserUpdates{
			Email:    req.Email,
			Username: req.Username,
			FullName: req.FullName,
			IsActive: req.IsActive,
		},
	}

	reqData, err := reqPayload.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update user request: %w", err)
	}

	msg := nats.NewMsg(SubjectUsersUpdate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send update user request: %w", err)
	}

	var response usersResponse
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update user response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("update user failed: %s", response.Error)
	}

	var updateResp domain.UpdateUserResponse
	if err := json.Unmarshal(response.Data, &updateResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update user data: %w", err)
	}

	return &updateResp, nil
}

// DeleteUser deletes a user by ID
func (c *NATSUserConnector) DeleteUser(ctx context.Context, id uuid.UUID) error {
	req := DeleteUserRequest{ID: id}
	reqData, err := req.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal delete user request: %w", err)
	}

	msg := nats.NewMsg(SubjectUsersDelete)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send delete user request: %w", err)
	}

	var response usersResponse
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal delete user response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("delete user failed: %s", response.Error)
	}

	return nil
}

// CreateRegistrationRequest creates a new registration request
func (c *NATSUserConnector) CreateRegistrationRequest(ctx context.Context, req domain.CreateRegistrationRequestRequest) (*domain.CreateRegistrationRequestResponse, error) {
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create registration request: %w", err)
	}

	msg := nats.NewMsg(SubjectRegistrationRequestCreate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create registration request: %w", err)
	}

	var response usersResponse
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create registration response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("create registration request failed: %s", response.Error)
	}

	var createResp domain.CreateRegistrationRequestResponse
	if err := json.Unmarshal(response.Data, &createResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create registration data: %w", err)
	}

	return &createResp, nil
}

// ListRegistrationRequests retrieves a list of registration requests
func (c *NATSUserConnector) ListRegistrationRequests(ctx context.Context, query domain.ListRegistrationRequestsQuery) (*domain.ListRegistrationRequestsResponse, error) {
	reqData, err := query.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list registration requests: %w", err)
	}

	msg := nats.NewMsg(SubjectRegistrationRequestList)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list registration requests: %w", err)
	}

	var response usersResponse
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list registration response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("list registration requests failed: %s", response.Error)
	}

	var listResp domain.ListRegistrationRequestsResponse
	if err := json.Unmarshal(response.Data, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list registration data: %w", err)
	}

	return &listResp, nil
}

// ApproveRegistrationRequest approves a registration request
func (c *NATSUserConnector) ApproveRegistrationRequest(ctx context.Context, requestID, approverID uuid.UUID) (*domain.ApproveRegistrationRequestResponse, error) {
	req := ApproveRegistrationRequest{
		RequestID:  requestID,
		ApproverID: approverID,
	}

	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal approve request: %w", err)
	}

	msg := nats.NewMsg(SubjectRegistrationRequestApprove)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send approve request: %w", err)
	}

	var response usersResponse
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approve response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("approve registration request failed: %s", response.Error)
	}

	var approveResp domain.ApproveRegistrationRequestResponse
	if err := json.Unmarshal(response.Data, &approveResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approve data: %w", err)
	}

	return &approveResp, nil
}

// RejectRegistrationRequest rejects a registration request
func (c *NATSUserConnector) RejectRegistrationRequest(ctx context.Context, requestID, approverID uuid.UUID, reason string) (*domain.RejectRegistrationRequestResponse, error) {
	req := RejectRegistrationRequest{
		RequestID:  requestID,
		ApproverID: approverID,
		Reason:     reason,
	}

	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reject request: %w", err)
	}

	msg := nats.NewMsg(SubjectRegistrationRequestReject)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send reject request: %w", err)
	}

	var response usersResponse
	if err := json.Unmarshal(respMsg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reject response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("reject registration request failed: %s", response.Error)
	}

	var rejectResp domain.RejectRegistrationRequestResponse
	if err := json.Unmarshal(response.Data, &rejectResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reject data: %w", err)
	}

	return &rejectResp, nil
}
