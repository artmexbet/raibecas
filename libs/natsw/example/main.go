package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/natsw"
)

// UserEvent - пример события
type UserEvent struct {
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// Настройка логера
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Подключение к NATS
	nc, err := nats.Connect(
		nats.DefaultURL,
		nats.Name("example-service"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Создание клиента с middleware
	client := natsw.NewClient(nc,
		natsw.WithLogger(logger),
		natsw.WithRecover(),
		natsw.WithMiddleware(authMiddleware()),
	)

	// Пример 1: Подписка на события
	_, err = client.Subscribe("user.events", handleUserEvent)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	slog.Info("Subscribed to user.events")

	// Пример 2: Публикация событий (симуляция)
	go publishEvents(client)

	// Ждём сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down...")
}

// handleUserEvent обрабатывает события пользователей
func handleUserEvent(msg *natsw.Message) error {
	var event UserEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return err
	}

	slog.InfoContext(msg.Ctx, "User event received",
		"user_id", event.UserID,
		"action", event.Action,
		"timestamp", event.Timestamp,
	)

	return nil
}

// publishEvents периодически публикует события
func publishEvents(client *natsw.Client) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	counter := 0
	for range ticker.C {
		counter++

		ctx := context.Background()

		event := &UserEvent{
			UserID:    "user-123",
			Action:    "click",
			Timestamp: time.Now(),
		}

		data, _ := json.Marshal(event)
		err := client.Publish(ctx, "user.events", data)
		if err != nil {
			slog.Error("Failed to publish event", "error", err)
		} else {
			slog.Info("Event published", "counter", counter)
		}
	}
}

// authMiddleware - пример кастомного middleware
func authMiddleware() natsw.Middleware {
	return func(next natsw.HandlerFunc) natsw.HandlerFunc {
		return func(msg *natsw.Message) error {
			// Извлекаем user_id из headers и добавляем в контекст
			if msg.Header != nil {
				if userID := msg.Header.Get("X-User-Id"); userID != "" {
					msg.Ctx = context.WithValue(msg.Ctx, "user_id", userID)
				}
			}
			return next(msg)
		}
	}
}
