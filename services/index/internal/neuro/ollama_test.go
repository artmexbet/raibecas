package neuro

import (
	"context"
	"testing"

	"github.com/artmexbet/raibecas/services/index/internal/config"
)

func TestNewConnector(t *testing.T) {
	cfg := &config.Ollama{
		Protocol:       "http",
		Host:           "localhost",
		Port:           "11434",
		EmbeddingModel: "mxbai-embed-large",
		Timeout:        30,
	}

	connector, err := NewConnector(cfg)
	if err != nil {
		t.Fatalf("NewConnector() error = %v", err)
	}

	if connector == nil {
		t.Fatal("NewConnector() returned nil")
	}

	if connector.cfg != cfg {
		t.Errorf("connector.cfg != cfg")
	}

	if connector.client == nil {
		t.Error("connector.client is nil")
	}
}

func TestNewConnector_InvalidURL(t *testing.T) {
	cfg := &config.Ollama{
		Protocol:       "ht!tp", // invalid protocol
		Host:           "localhost",
		Port:           "11434",
		EmbeddingModel: "mxbai-embed-large",
		Timeout:        30,
	}

	connector, err := NewConnector(cfg)

	// В зависимости от реализации может не быть ошибки при создании,
	// но проверяем что функция не паникует
	if err != nil {
		t.Logf("NewConnector() with invalid URL returned error (expected): %v", err)
	}
	if connector != nil && connector.client == nil {
		t.Error("connector created but client is nil")
	}
}

func TestConnector_Embedding_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := &config.Ollama{
		Protocol:       "http",
		Host:           "localhost",
		Port:           "11434",
		EmbeddingModel: "mxbai-embed-large",
		Timeout:        30,
	}

	connector, err := NewConnector(cfg)
	if err != nil {
		t.Skipf("NewConnector() error = %v (Ollama not available)", err)
	}

	ctx := context.Background()

	// Проверяем ping сначала
	if err := connector.Ping(ctx); err != nil {
		t.Skipf("Ollama not available: %v", err)
	}

	// Генерируем эмбеддинг
	embedding, err := connector.Embedding(ctx, "test text")
	if err != nil {
		t.Fatalf("Embedding() error = %v", err)
	}

	if len(embedding) == 0 {
		t.Error("Embedding() returned empty vector")
	}

	// Проверяем что эмбеддинг имеет разумную размерность (обычно 768 или 1024)
	if len(embedding) < 100 || len(embedding) > 5000 {
		t.Errorf("Embedding() vector size = %d, expected between 100 and 5000", len(embedding))
	}
}

func TestConnector_Ping_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := &config.Ollama{
		Protocol:       "http",
		Host:           "localhost",
		Port:           "11434",
		EmbeddingModel: "mxbai-embed-large",
		Timeout:        30,
	}

	connector, err := NewConnector(cfg)
	if err != nil {
		t.Skipf("NewConnector() error = %v", err)
	}

	ctx := context.Background()
	err = connector.Ping(ctx)

	// Если Ollama не доступен, тест должен быть пропущен, а не провален
	if err != nil {
		t.Skipf("Ollama not available: %v", err)
	}
}

func TestConnector_Embedding_EmptyText(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := &config.Ollama{
		Protocol:       "http",
		Host:           "localhost",
		Port:           "11434",
		EmbeddingModel: "mxbai-embed-large",
		Timeout:        30,
	}

	connector, err := NewConnector(cfg)
	if err != nil {
		t.Skipf("NewConnector() error = %v", err)
	}

	ctx := context.Background()

	if err := connector.Ping(ctx); err != nil {
		t.Skipf("Ollama not available: %v", err)
	}

	// Проверяем эмбеддинг пустого текста
	embedding, err := connector.Embedding(ctx, "")
	if err != nil {
		t.Logf("Embedding(\"\") error = %v (may be expected)", err)
		return
	}

	if len(embedding) == 0 {
		t.Error("Embedding(\"\") returned empty vector")
	}
}
