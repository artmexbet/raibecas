package server

import (
	"context"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// AuthServiceConnector defines the interface for communicating with the auth service
type AuthServiceConnector interface {
	// Login authenticates a user and returns tokens
	Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error)

	// RefreshToken refreshes an access token using a refresh token
	RefreshToken(ctx context.Context, req domain.RefreshTokenRequest) (*domain.RefreshTokenResponse, error)

	// ValidateToken validates an access token
	ValidateToken(ctx context.Context, token string) (*domain.ValidateTokenResponse, error)

	// Logout logs out a user from the current device
	Logout(ctx context.Context, userID uuid.UUID, token string) error

	// LogoutAll logs out a user from all devices
	LogoutAll(ctx context.Context, userID uuid.UUID, token string) error

	// ChangePassword changes a user's password
	ChangePassword(ctx context.Context, userID uuid.UUID, req domain.ChangePasswordRequest) error
}
