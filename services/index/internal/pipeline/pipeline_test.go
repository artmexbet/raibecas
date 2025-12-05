package pipeline_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/domain"
	"github.com/artmexbet/raibecas/services/index/internal/pipeline"
)

// Mock implementations
type mockEmbedder struct {
	embeddings []float64
	err        error
}

func (m *mockEmbedder) Embedding(ctx context.Context, text string) ([]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.embeddings != nil {
		return m.embeddings, nil
	}
	// Return dummy embedding
	return []float64{0.1, 0.2, 0.3}, nil
}

type mockChunkWriter struct {
	chunks []domain.Chunk
	err    error
}

func (m *mockChunkWriter) WriteChunks(ctx context.Context, chunks []domain.Chunk) error {
	if m.err != nil {
		return m.err
	}
	m.chunks = append(m.chunks, chunks...)
	return nil
}

type mockStorageReader struct {
	content string
	err     error
}

func (m *mockStorageReader) Get(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return io.NopCloser(strings.NewReader(m.content)), nil
}

func TestPipeline_Index_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Pipeline: config.Pipeline{
			ChunkSize:    50,
			ChunkOverlap: 10,
			MaxChunks:    10,
		},
	}

	content := "This is a test document. It has multiple sentences. We will chunk it."
	embedder := &mockEmbedder{}
	writer := &mockChunkWriter{}
	storage := &mockStorageReader{content: content}

	pipe := pipeline.New(cfg, embedder, writer, storage)

	doc := domain.Document{
		ID:       "test-doc-1",
		Title:    "Test Document",
		FilePath: "storage/test-doc-1.txt",
		Metadata: map[string]string{"author": "Test Author"},
	}

	// Act
	err := pipe.Index(context.Background(), doc)

	// Assert
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if len(writer.chunks) == 0 {
		t.Error("Expected chunks to be written, got 0")
	}

	// Verify chunk structure
	for _, chunk := range writer.chunks {
		if chunk.DocumentID != doc.ID {
			t.Errorf("Chunk DocumentID = %s, want %s", chunk.DocumentID, doc.ID)
		}
		if len(chunk.Embedding) == 0 {
			t.Error("Chunk embedding is empty")
		}
		if chunk.Text == "" {
			t.Error("Chunk text is empty")
		}
		if chunk.Metadata["author"] != "Test Author" {
			t.Error("Chunk metadata not preserved")
		}
	}
}

func TestPipeline_Index_StorageError(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Pipeline: config.Pipeline{
			ChunkSize:    50,
			ChunkOverlap: 10,
			MaxChunks:    10,
		},
	}

	embedder := &mockEmbedder{}
	writer := &mockChunkWriter{}
	storage := &mockStorageReader{err: errors.New("storage error")}

	pipe := pipeline.New(cfg, embedder, writer, storage)

	doc := domain.Document{
		ID:       "test-doc-2",
		FilePath: "storage/test-doc-2.txt",
	}

	// Act
	err := pipe.Index(context.Background(), doc)

	// Assert
	if err == nil {
		t.Error("Expected error from storage, got nil")
	}

	if len(writer.chunks) != 0 {
		t.Errorf("Expected 0 chunks written, got %d", len(writer.chunks))
	}
}

func TestPipeline_Index_EmbeddingError(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Pipeline: config.Pipeline{
			ChunkSize:    50,
			ChunkOverlap: 10,
			MaxChunks:    10,
		},
	}

	content := "This is a test document for embedding error."
	embedder := &mockEmbedder{err: errors.New("embedding error")}
	writer := &mockChunkWriter{}
	storage := &mockStorageReader{content: content}

	pipe := pipeline.New(cfg, embedder, writer, storage)

	doc := domain.Document{
		ID:       "test-doc-3",
		FilePath: "storage/test-doc-3.txt",
	}

	// Act
	err := pipe.Index(context.Background(), doc)

	// Assert
	if err == nil {
		t.Error("Expected embedding error, got nil")
	}

	if len(writer.chunks) != 0 {
		t.Errorf("Expected 0 chunks written, got %d", len(writer.chunks))
	}
}

func TestPipeline_Index_EmptyDocument(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Pipeline: config.Pipeline{
			ChunkSize:    50,
			ChunkOverlap: 10,
			MaxChunks:    10,
		},
	}

	embedder := &mockEmbedder{}
	writer := &mockChunkWriter{}
	storage := &mockStorageReader{content: ""} // Empty content

	pipe := pipeline.New(cfg, embedder, writer, storage)

	doc := domain.Document{
		ID:       "test-doc-4",
		FilePath: "storage/test-doc-4.txt",
	}

	// Act
	err := pipe.Index(context.Background(), doc)

	// Assert
	if err == nil {
		t.Error("Expected error for empty document, got nil")
	}
}

func TestPipeline_Index_LargeDocument(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Pipeline: config.Pipeline{
			ChunkSize:    100,
			ChunkOverlap: 20,
			MaxChunks:    5, // Limit chunks
		},
	}

	// Create large content
	content := strings.Repeat("This is a sentence. ", 100) // ~2000 chars

	embedder := &mockEmbedder{}
	writer := &mockChunkWriter{}
	storage := &mockStorageReader{content: content}

	pipe := pipeline.New(cfg, embedder, writer, storage)

	doc := domain.Document{
		ID:       "test-doc-5",
		FilePath: "storage/test-doc-5.txt",
	}

	// Act
	err := pipe.Index(context.Background(), doc)

	// Assert
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	// Verify max chunks limit is respected
	if len(writer.chunks) > cfg.Pipeline.MaxChunks {
		t.Errorf("Expected max %d chunks, got %d", cfg.Pipeline.MaxChunks, len(writer.chunks))
	}

	// Verify all chunks have increasing ordinals
	for i, chunk := range writer.chunks {
		if chunk.Ordinal != i {
			t.Errorf("Chunk %d has ordinal %d, want %d", i, chunk.Ordinal, i)
		}
	}
}

func TestPipeline_Index_WriterError(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Pipeline: config.Pipeline{
			ChunkSize:    50,
			ChunkOverlap: 10,
			MaxChunks:    10,
		},
	}

	content := "This is a test document for writer error."
	embedder := &mockEmbedder{}
	writer := &mockChunkWriter{err: errors.New("writer error")}
	storage := &mockStorageReader{content: content}

	pipe := pipeline.New(cfg, embedder, writer, storage)

	doc := domain.Document{
		ID:       "test-doc-6",
		FilePath: "storage/test-doc-6.txt",
	}

	// Act
	err := pipe.Index(context.Background(), doc)

	// Assert
	if err == nil {
		t.Error("Expected writer error, got nil")
	}
}
