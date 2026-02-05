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

	users := []struct {
		Username string
		Email    string
		Password string
		FullName string
		Role     string
		IsActive bool
	}{
		{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "password123",
			FullName: "Admin User",
			Role:     "admin",
			IsActive: true,
		},
		{
			Username: "user",
			Email:    "user@example.com",
			Password: "password123",
			FullName: "Regular User",
			Role:     "user",
			IsActive: true,
		},
		{
			Username: "inactive",
			Email:    "inactive@example.com",
			Password: "password123",
			FullName: "Inactive User",
			Role:     "user",
			IsActive: false,
		},
	}

	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("failed to hash password", "user", u.Username, "error", err)
			continue
		}

		user := &domain.User{
			Username:     u.Username,
			Email:        u.Email,
			PasswordHash: string(hash),
			FullName:     u.FullName,
			Role:         u.Role,
			IsActive:     u.IsActive,
		}

		err = pg.CreateUser(ctx, user)
		if err != nil {
			slog.Error("failed to create user", "user", u.Username, "error", err)
		} else {
			slog.Info("user created", "user", u.Username, "id", user.ID)
		}
	}
}
