package ingestion

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/domain"
	"github.com/nats-io/nats.go"
)

type mockPipeline struct {
	indexedDocs []domain.Document
	err         error
}

func (m *mockPipeline) Index(_ context.Context, doc domain.Document) error {
	if m.err != nil {
		return m.err
	}
	m.indexedDocs = append(m.indexedDocs, doc)
	return nil
}

func TestConsumer_handleMessage(t *testing.T) {
	tests := []struct {
		name    string
		event   domain.DocumentIndexEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: domain.DocumentIndexEvent{
				DocumentID: "doc123",
				Title:      "Test Document",
				FilePath:   "storage/doc123.txt",
				SourceURI:  "https://example.com/doc123",
				Metadata:   map[string]string{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "missing document_id",
			event: domain.DocumentIndexEvent{
				FilePath: "storage/doc.txt",
			},
			wantErr: true,
		},
		{
			name: "missing file_path",
			event: domain.DocumentIndexEvent{
				DocumentID: "doc456",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPipe := &mockPipeline{}
			consumer := &Consumer{
				cfg: &config.NATS{
					Subject: "test.subject",
					Queue:   "test-queue",
					Durable: "test-durable",
				},
				pipeline: mockPipe,
			}

			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("failed to marshal event: %v", err)
			}

			err = consumer.handleMessage(context.Background(), data)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleMessage() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(mockPipe.indexedDocs) != 1 {
				t.Errorf("expected 1 indexed document, got %d", len(mockPipe.indexedDocs))
			}
		})
	}
}

func TestNewConsumer(t *testing.T) {
	// Этот тест требует запущенного NATS сервера, поэтому пропускаем его в CI
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("NATS not available: %v", err)
	}
	defer nc.Close()

	cfg := &config.NATS{
		URL:     nats.DefaultURL,
		Stream:  "TEST_STREAM",
		Subject: "test.subject",
		Queue:   "test-queue",
		Durable: "test-durable",
	}

	mockPipe := &mockPipeline{}

	consumer, err := NewConsumer(cfg, nc, mockPipe)
	if err != nil {
		t.Fatalf("NewConsumer() error = %v", err)
	}

	if consumer == nil {
		t.Fatal("expected consumer to be non-nil")
	}
}
