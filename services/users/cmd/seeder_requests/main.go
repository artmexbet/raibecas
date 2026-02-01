package main

import (
	"context"
	"log/slog"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/artmexbet/raibecas/services/users/internal/config"
	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	pg, err := postgres.New(context.Background(), cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	ctx := context.Background()

	requests := []struct {
		Username string
		Email    string
		Password string
		Status   domain.RegistrationStatus
		Metadata map[string]interface{}
	}{
		{
			Username: "new_candidate_1",
			Email:    "candidate1@example.com",
			Password: "password123",
			Status:   domain.RegistrationStatusPending,
			Metadata: map[string]interface{}{"note": "First candidate"},
		},
		{
			Username: "new_candidate_2",
			Email:    "candidate2@example.com",
			Password: "password123",
			Status:   domain.RegistrationStatusPending,
			Metadata: map[string]interface{}{"note": "Second candidate", "department": "IT"},
		},
		{
			Username: "rejected_candidate",
			Email:    "rejected@example.com",
			Password: "password123",
			Status:   domain.RegistrationStatusRejected,
			Metadata: map[string]interface{}{"reason": "Spam"},
		},
	}

	for _, r := range requests {
		hash, err := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("failed to hash password", "username", r.Username, "error", err)
			continue
		}

		req := &domain.RegistrationRequest{
			Username:     r.Username,
			Email:        r.Email,
			PasswordHash: string(hash),
			Status:       r.Status,
			Metadata:     r.Metadata,
		}

		err = pg.CreateRegistrationRequest(ctx, req)
		if err != nil {
			slog.Error("failed to create registration request", "username", r.Username, "error", err)
		} else {
			slog.Info("registration request created", "username", r.Username, "id", req.ID, "status", req.Status)
		}
	}
}
