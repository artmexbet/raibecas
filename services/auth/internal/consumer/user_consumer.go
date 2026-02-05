package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	UpdateUser(ctx context.Context, userID uuid.UUID, username, email string, role domain.UserRole, isActive bool) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	ExistsUserByEmail(ctx context.Context, email string) (bool, error)
	ExistsUserByUsername(ctx context.Context, username string) (bool, error)
}

// UserRegisteredEvent represents the event payload from users service
type UserRegisteredEvent struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	IsActive     bool   `json:"is_active"`
}

// UserUpdatedEvent represents the user update event payload from users service
type UserUpdatedEvent struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

// UserConsumer handles user registration events
type UserConsumer struct {
	userRepo UserRepository
	logger   *slog.Logger
}

// NewUserConsumer creates a new user consumer
func NewUserConsumer(userRepo UserRepository, logger *slog.Logger) *UserConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	return &UserConsumer{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Subscribe subscribes to user registration events
func (c *UserConsumer) Subscribe(client *natsw.Client) error {
	_, err := client.Subscribe("users.user.registered", c.handleUserRegistered)
	if err != nil {
		return fmt.Errorf("failed to subscribe to users.user.registered: %w", err)
	}

	_, err = client.Subscribe("users.user.updated", c.handleUserUpdated)
	if err != nil {
		return fmt.Errorf("failed to subscribe to users.user.updated: %w", err)
	}

	c.logger.Info("subscribed to user events (registered, updated)")
	return nil
}

// handleUserRegistered processes user.registered events
func (c *UserConsumer) handleUserRegistered(msg *natsw.Message) error {
	var event UserRegisteredEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		c.logger.Error("failed to unmarshal user registered event", "error", err)
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	c.logger.Info("received user registered event",
		"user_id", event.UserID,
		"email", event.Email,
		"username", event.Username,
		"role", event.Role,
	)

	// Parse user ID
	userID, err := uuid.Parse(event.UserID)
	if err != nil {
		c.logger.Error("invalid user ID", "user_id", event.UserID, "error", err)
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user already exists (idempotency)
	exists, err := c.userRepo.ExistsUserByEmail(msg.Ctx, event.Email)
	if err != nil {
		c.logger.Error("failed to check user existence by email", "error", err)
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		c.logger.Warn("user already exists, skipping", "email", event.Email)
		return nil // Acknowledge the message - this is expected for idempotency
	}

	exists, err = c.userRepo.ExistsUserByUsername(msg.Ctx, event.Username)
	if err != nil {
		c.logger.Error("failed to check user existence by username", "error", err)
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		c.logger.Warn("user already exists, skipping", "username", event.Username)
		return nil // Acknowledge the message - this is expected for idempotency
	}

	// Create user
	user := &domain.User{
		ID:           userID,
		Username:     event.Username,
		Email:        event.Email,
		PasswordHash: event.PasswordHash,
		Role:         domain.UserRole(event.Role),
		IsActive:     event.IsActive,
	}

	if err := c.userRepo.CreateUser(msg.Ctx, user); err != nil {
		c.logger.Error("failed to create user", "error", err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	c.logger.Info("user created successfully",
		"user_id", userID,
		"email", event.Email,
		"username", event.Username,
	)

	return nil
}

// handleUserUpdated processes user.updated events
func (c *UserConsumer) handleUserUpdated(msg *natsw.Message) error {
	var event UserUpdatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		c.logger.Error("failed to unmarshal user updated event", "error", err)
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	c.logger.Info("received user updated event",
		"user_id", event.UserID,
		"email", event.Email,
		"username", event.Username,
		"role", event.Role,
	)

	// Parse user ID
	userID, err := uuid.Parse(event.UserID)
	if err != nil {
		c.logger.Error("invalid user ID", "user_id", event.UserID, "error", err)
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user exists
	existingUser, err := c.userRepo.GetUserByID(msg.Ctx, userID)
	if err != nil {
		c.logger.Error("user not found for update", "user_id", userID, "error", err)
		return fmt.Errorf("user not found: %w", err)
	}

	c.logger.Debug("updating user in auth service",
		"user_id", userID,
		"old_username", existingUser.Username,
		"new_username", event.Username,
		"old_email", existingUser.Email,
		"new_email", event.Email,
		"old_role", existingUser.Role,
		"new_role", event.Role,
	)

	// Update user
	if err := c.userRepo.UpdateUser(
		msg.Ctx,
		userID,
		event.Username,
		event.Email,
		domain.UserRole(event.Role),
		event.IsActive,
	); err != nil {
		c.logger.Error("failed to update user", "user_id", userID, "error", err)
		return fmt.Errorf("failed to update user: %w", err)
	}

	c.logger.Info("user updated successfully",
		"user_id", userID,
		"username", event.Username,
		"email", event.Email,
		"role", event.Role,
		"is_active", event.IsActive,
	)

	return nil
}
