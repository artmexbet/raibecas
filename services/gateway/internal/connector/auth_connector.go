package connector

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// NATS subjects for auth service communication
const (
	SubjectAuthLogin          = "auth.login"
	SubjectAuthRefresh        = "auth.refresh"
	SubjectAuthValidate       = "auth.validate"
	SubjectAuthLogout         = "auth.logout"
	SubjectAuthLogoutAll      = "auth.logout_all"
	SubjectAuthChangePassword = "auth.change_password"
)

// NATSAuthConnector implements server.AuthServiceConnector using NATS for communication
type NATSAuthConnector struct {
	conn *nats.Conn
}

// NewNATSAuthConnector creates a new NATS-based auth service connector
func NewNATSAuthConnector(conn *nats.Conn) *NATSAuthConnector {
	return &NATSAuthConnector{
		conn: conn,
	}
}

// authResponse represents a generic NATS response from auth service
type authResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Login authenticates a user and returns tokens
func (c *NATSAuthConnector) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectAuthLogin, reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send login request: %w", err)
	}

	var response authResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("login failed: %s", response.Error)
	}

	var loginResp domain.LoginResponse
	if err := json.Unmarshal(response.Data, &loginResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal login data: %w", err)
	}

	return &loginResp, nil
}

// RefreshToken refreshes an access token using a refresh token
func (c *NATSAuthConnector) RefreshToken(ctx context.Context, req domain.RefreshTokenRequest) (*domain.RefreshTokenResponse, error) {
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectAuthRefresh, reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send refresh request: %w", err)
	}

	var response authResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("refresh failed: %s", response.Error)
	}

	var refreshResp domain.RefreshTokenResponse
	if err := json.Unmarshal(response.Data, &refreshResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh data: %w", err)
	}

	return &refreshResp, nil
}

// ValidateToken validates an access token
func (c *NATSAuthConnector) ValidateToken(ctx context.Context, token string) (*domain.ValidateTokenResponse, error) {
	req := domain.ValidateTokenRequest{Token: token}
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validate request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectAuthValidate, reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send validate request: %w", err)
	}

	var response authResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validate response: %w", err)
	}

	if !response.Success {
		return &domain.ValidateTokenResponse{Valid: false}, nil
	}

	var validateResp domain.ValidateTokenResponse
	if err := json.Unmarshal(response.Data, &validateResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validate data: %w", err)
	}

	return &validateResp, nil
}

// Logout logs out a user from the current device
func (c *NATSAuthConnector) Logout(ctx context.Context, userID uuid.UUID, token string) error {
	type logoutRequest struct {
		UserID uuid.UUID `json:"user_id"`
		Token  string    `json:"token"`
	}

	req := logoutRequest{
		UserID: userID,
		Token:  token,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal logout request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectAuthLogout, reqData)
	if err != nil {
		return fmt.Errorf("failed to send logout request: %w", err)
	}

	var response authResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal logout response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("logout failed: %s", response.Error)
	}

	return nil
}

// LogoutAll logs out a user from all devices
func (c *NATSAuthConnector) LogoutAll(ctx context.Context, userID uuid.UUID, token string) error {
	type logoutAllRequest struct {
		UserID uuid.UUID `json:"user_id"`
		Token  string    `json:"token"`
	}

	req := logoutAllRequest{
		UserID: userID,
		Token:  token,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal logout all request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectAuthLogoutAll, reqData)
	if err != nil {
		return fmt.Errorf("failed to send logout all request: %w", err)
	}

	var response authResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal logout all response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("logout all failed: %s", response.Error)
	}

	return nil
}

// ChangePassword changes a user's password
func (c *NATSAuthConnector) ChangePassword(ctx context.Context, userID uuid.UUID, req domain.ChangePasswordRequest) error {
	type changePasswordRequest struct {
		UserID      uuid.UUID `json:"user_id"`
		Token       string    `json:"token"`
		OldPassword string    `json:"old_password"`
		NewPassword string    `json:"new_password"`
	}

	changeReq := changePasswordRequest{
		UserID:      userID,
		Token:       req.Token,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	reqData, err := json.Marshal(changeReq)
	if err != nil {
		return fmt.Errorf("failed to marshal change password request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectAuthChangePassword, reqData)
	if err != nil {
		return fmt.Errorf("failed to send change password request: %w", err)
	}

	var response authResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal change password response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("change password failed: %s", response.Error)
	}

	return nil
}
