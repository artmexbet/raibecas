package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

// TestRedisSetup creates a test Redis client
func setupTestRedis(t *testing.T) (*Redis, func()) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   2, // Use separate DB for tests
	})

	// Check connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping Redis tests: %v", err)
	}

	cfg := &config.Redis{
		Host:       "localhost",
		Port:       "6379",
		DB:         1,
		ChatTTL:    3600, // 1 hour for tests
		MessageTTL: 3600,
	}

	r := New(cfg, client)

	cleanup := func() {
		ctx := context.Background()
		client.FlushDB(ctx)
		_ = client.Close()
	}

	return r, cleanup
}

func TestSaveMessage(t *testing.T) {
	r, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test_user_1"

	// Save first message
	msg1 := domain.Message{
		Role:    "user",
		Content: "Hello",
	}

	err := r.SaveMessage(ctx, userID, msg1)
	if err != nil {
		t.Fatalf("Failed to save message: %v", err)
	}

	// Retrieve and verify
	history, err := r.RetrieveChatHistory(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to retrieve history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 message, got %d", len(history))
	}

	if history[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got '%s'", history[0].Content)
	}

	// Save second message
	msg2 := domain.Message{
		Role:    "assistant",
		Content: "Hi there!",
	}

	err = r.SaveMessage(ctx, userID, msg2)
	if err != nil {
		t.Fatalf("Failed to save second message: %v", err)
	}

	history, err = r.RetrieveChatHistory(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to retrieve history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(history))
	}
}

func TestClearChatHistory(t *testing.T) {
	r, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test_user_3"

	// Save some messages
	msg1 := domain.Message{Role: "user", Content: "Message 1"}
	msg2 := domain.Message{Role: "assistant", Content: "Message 2"}

	_ = r.SaveMessage(ctx, userID, msg1)
	_ = r.SaveMessage(ctx, userID, msg2)

	// Clear history
	err := r.ClearChatHistory(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to clear chat history: %v", err)
	}

	// Verify history is empty
	history, err := r.RetrieveChatHistory(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to retrieve history: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d messages", len(history))
	}

	// Verify temp key is deleted
	tempKey := r.getTemporaryMessageKey(userID)
	val, err := r.client.Get(ctx, tempKey).Result()
	if !errors.Is(err, redis.Nil) {
		t.Errorf("Expected temp key to be deleted, got: %s", val)
	}
}

func TestGetChatSize(t *testing.T) {
	r, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test_user_4"

	// Empty chat
	size, err := r.GetChatSize(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get chat size: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}

	// Add messages
	for i := 0; i < 5; i++ {
		msg := domain.Message{Role: "user", Content: "Message"}
		_ = r.SaveMessage(ctx, userID, msg)
	}

	size, err = r.GetChatSize(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get chat size: %v", err)
	}
	if size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}
}

func TestMultipleUsers(t *testing.T) {
	r, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	// Add messages for different users
	for user := 1; user <= 3; user++ {
		userID := "user_" + string(rune(48+user))
		for msg := 1; msg <= 3; msg++ {
			message := domain.Message{
				Role:    "user",
				Content: "Message from user",
			}
			_ = r.SaveMessage(ctx, userID, message)
		}
	}

	// Verify each user has independent history
	for user := 1; user <= 3; user++ {
		userID := "user_" + string(rune(48+user))
		history, err := r.RetrieveChatHistory(ctx, userID)
		if err != nil {
			t.Fatalf("Failed to retrieve history for %s: %v", userID, err)
		}

		if len(history) != 3 {
			t.Errorf("Expected 3 messages for %s, got %d", userID, len(history))
		}
	}
}
