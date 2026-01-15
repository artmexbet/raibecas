package server

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	slogfiber "github.com/samber/slog-fiber"

	"github.com/artmexbet/raibecas/services/gateway/internal/config"
)

type Server struct {
	router *fiber.App
}

func New(cfg *config.HTTPConfig) *Server {
	router := fiber.New()
	logger := slog.Default()
	router.Use(slogfiber.New(logger))
	router.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	})) // Enable CORS middleware
	router.Use(requestid.New())
	router.Use(limiter.New(limiter.Config{Max: cfg.RPS}))
	router.Use(recoverer.New())
	router.Use(healthcheck.New())

	return &Server{
		router: router,
	}
}
