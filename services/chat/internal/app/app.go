package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/handler/http"
	"github.com/artmexbet/raibecas/services/chat/internal/neuro"
	qdrantWrapper "github.com/artmexbet/raibecas/services/chat/internal/qdrant-wrapper"
	_redis "github.com/artmexbet/raibecas/services/chat/internal/redis"
	"github.com/artmexbet/raibecas/services/chat/internal/service"
	"github.com/redis/go-redis/v9"

	"github.com/qdrant/go-client/qdrant"
)

type App struct {
}

func New() *App {
	return &App{}
}

func (a *App) Run() error {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}
	slog.InfoContext(ctx, "Loaded config", slog.Any("config", cfg))

	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.Qdrant.Host,
		Port: cfg.Qdrant.Port,
	})
	if err != nil {
		return fmt.Errorf("error creating qdrant redisClient: %v", err)
	}
	if _, err := qdrantClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("error connecting to qdrant: %v", err)
	}

	slog.InfoContext(ctx, "Connected to qdrant")

	qdrantWrap := qdrantWrapper.New(&cfg.Qdrant, qdrantClient)
	err = qdrantWrap.CheckConnection(ctx)
	if err != nil {
		return fmt.Errorf("error checking qdrant connection: %v", err)
	}

	ollama, err := neuro.NewConnector(&cfg.Ollama)
	if err != nil {
		return fmt.Errorf("error creating ollama connector: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.GetAddress(), DB: cfg.Redis.DB})
	status := redisClient.Ping(context.Background())
	if status.Err() != nil {
		return fmt.Errorf("error connecting to redis: %v", status.Err())
	}
	slog.Info("Connected to redis")

	historyStore := _redis.New(&cfg.Redis, redisClient)

	svc := service.New(qdrantWrap, ollama, historyStore)

	ch := make(chan os.Signal, 1)
	defer close(ch)
	signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)

	var api *http.Handler
	if cfg.UseHTTP {
		api = http.New(&cfg.HTTP, svc)
		api.RegisterRoutes()
		go func() {
			if err := api.Run(); err != nil {
				slog.Error("HTTP server error", "err", err)
				ch <- syscall.SIGTERM
			}
		}()
	}

	//wait for shutdown signal
	<-ch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	slog.Info("Shutting down the service...")
	if api != nil {
		if err := api.Shutdown(ctx); err != nil {
			slog.Error("Error during HTTP server shutdown", "err", err)
		}
	}

	//redisClient.Shutdown(ctx)
	err = qdrantClient.Close()
	if err != nil {
		slog.Error("Error during Qdrant client shutdown", "err", err)
	}
	slog.Info("Service stopped gracefully")

	return nil
}
