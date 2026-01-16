package natsw_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/natsw"
)

type TestEvent struct {
	ID      string    `json:"id"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

func TestClient_Subscribe(t *testing.T) {
	// Skip if NATS server is not available
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
	}
	defer nc.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := natsw.NewClient(nc,
		natsw.WithLogger(logger),
		natsw.WithRecover(),
	)

	received := make(chan *TestEvent, 1)

	// Подписываемся с обработчиком
	sub, err := client.Subscribe("test.events", func(msg *natsw.Message) error {
		var event TestEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			return err
		}

		t.Logf("Received event: %+v", event)
		received <- &event
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Публикуем событие
	testEvent := &TestEvent{
		ID:      "123",
		Message: "test message",
		Time:    time.Now(),
	}

	data, _ := json.Marshal(testEvent)
	err = client.Publish(context.Background(), "test.events", data)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Ждём получения события
	select {
	case event := <-received:
		if event.ID != testEvent.ID {
			t.Errorf("Expected ID %s, got %s", testEvent.ID, event.ID)
		}
		if event.Message != testEvent.Message {
			t.Errorf("Expected message %s, got %s", testEvent.Message, event.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestClient_Middleware(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
	}
	defer nc.Close()

	middlewareCalled := false

	customMiddleware := func(next natsw.HandlerFunc) natsw.HandlerFunc {
		return func(msg *natsw.Message) error {
			middlewareCalled = true
			return next(msg)
		}
	}

	client := natsw.NewClient(nc,
		natsw.WithMiddleware(customMiddleware),
	)

	received := make(chan bool, 1)

	sub, err := client.Subscribe("test.middleware", func(msg *natsw.Message) error {
		received <- true
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Публикуем сообщение
	err = client.Publish(context.Background(), "test.middleware", []byte(`{"test":"data"}`))
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Ждём обработки
	select {
	case <-received:
		if !middlewareCalled {
			t.Error("Middleware was not called")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestClient_RecoverMiddleware(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
	}
	defer nc.Close()

	client := natsw.NewClient(nc,
		natsw.WithRecover(),
	)

	// Подписываемся с handler, который паникует
	sub, err := client.Subscribe("test.panic", func(msg *natsw.Message) error {
		panic("test panic")
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Публикуем сообщение - не должно упасть приложение
	err = client.Publish(context.Background(), "test.panic", []byte(`{"test":"data"}`))
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Даём время на обработку
	time.Sleep(100 * time.Millisecond)

	// Если мы здесь, значит recover сработал
	t.Log("Panic was recovered successfully")
}
