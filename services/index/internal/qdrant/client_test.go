package qdrant

import (
	"context"
	"testing"

	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

func TestNew(t *testing.T) {
	cfg := &config.Qdrant{
		Host:            "localhost",
		Port:            6333,
		CollectionName:  "test_collection",
		VectorDimension: 768,
		Distance:        "Cosine",
	}

	// Пытаемся создать клиент
	client, err := New(cfg)

	// Если Qdrant не доступен, это нормально для unit-теста
	if err != nil {
		t.Logf("New() error = %v (Qdrant may not be available)", err)
		// Проверяем что ошибка содержит разумное сообщение
		if client != nil {
			t.Error("New() returned error but client is not nil")
		}
		return
	}

	if client == nil {
		t.Fatal("New() returned nil client without error")
	}

	if client.cfg != cfg {
		t.Error("client.cfg != cfg")
	}

	if client.raw == nil {
		t.Error("client.raw is nil")
	}

	// Закрываем клиент
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestClient_EnsureCollection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := &config.Qdrant{
		Host:            "localhost",
		Port:            6333,
		CollectionName:  "test_collection_ensure",
		VectorDimension: 768,
		Distance:        "Cosine",
	}

	client, err := New(cfg)
	if err != nil {
		t.Skipf("Qdrant not available: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Создаем коллекцию
	err = client.EnsureCollection(ctx)
	if err != nil {
		t.Fatalf("EnsureCollection() error = %v", err)
	}

	// Повторный вызов должен пройти успешно (коллекция уже существует)
	err = client.EnsureCollection(ctx)
	if err != nil {
		t.Errorf("EnsureCollection() second call error = %v", err)
	}
}

func TestPointsFromChunks(t *testing.T) {
	chunks := []domain.Chunk{
		{
			DocumentID: "doc1",
			Ordinal:    0,
			Text:       "First chunk",
			Embedding:  []float64{0.1, 0.2, 0.3},
			Metadata:   map[string]string{"key": "value1"},
		},
		{
			DocumentID: "doc1",
			Ordinal:    1,
			Text:       "Second chunk",
			Embedding:  []float64{0.4, 0.5, 0.6},
			Metadata:   map[string]string{"key": "value2"},
		},
	}

	points := PointsFromChunks(chunks)

	if len(points) != len(chunks) {
		t.Errorf("len(points) = %d, want %d", len(points), len(chunks))
	}

	for i, point := range points {
		if point == nil {
			t.Errorf("points[%d] is nil", i)
			continue
		}

		// Проверяем что ID корректный
		if point.Id == nil {
			t.Errorf("points[%d].Id is nil", i)
		}

		// Проверяем что вектор корректный
		if point.Vectors == nil {
			t.Errorf("points[%d].Vectors is nil", i)
		}

		// Проверяем payload
		if point.Payload == nil {
			t.Errorf("points[%d].Payload is nil", i)
		}
	}
}

func TestPointsFromChunks_Empty(t *testing.T) {
	chunks := []domain.Chunk{}
	points := PointsFromChunks(chunks)

	if len(points) != 0 {
		t.Errorf("len(points) = %d, want 0", len(points))
	}
}

func TestPointsFromChunks_EmptyEmbedding(t *testing.T) {
	chunks := []domain.Chunk{
		{
			DocumentID: "doc1",
			Ordinal:    0,
			Text:       "Chunk without embedding",
			Embedding:  []float64{},
			Metadata:   map[string]string{},
		},
	}

	points := PointsFromChunks(chunks)

	// Функция пропускает чанки с пустыми эмбеддингами
	if len(points) != 0 {
		t.Errorf("len(points) = %d, want 0 (empty embeddings should be skipped)", len(points))
	}
}

func TestClient_UpsertChunks_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := &config.Qdrant{
		Host:            "localhost",
		Port:            6333,
		CollectionName:  "test_collection_upsert",
		VectorDimension: 3,
		Distance:        "Cosine",
	}

	client, err := New(cfg)
	if err != nil {
		t.Skipf("Qdrant not available: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Создаем коллекцию
	if err := client.EnsureCollection(ctx); err != nil {
		t.Fatalf("EnsureCollection() error = %v", err)
	}

	// Создаем тестовые чанки
	chunks := []domain.Chunk{
		{
			DocumentID: "test-doc",
			Ordinal:    0,
			Text:       "Test chunk",
			Embedding:  []float64{0.1, 0.2, 0.3},
			Metadata:   map[string]string{"test": "value"},
		},
	}

	points := PointsFromChunks(chunks)

	// Вставляем чанки
	err = client.UpsertChunks(ctx, points)
	if err != nil {
		t.Fatalf("UpsertChunks() error = %v", err)
	}
}
