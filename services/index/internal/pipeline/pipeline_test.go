package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

type embedderStub struct {
	//values map[int][]float64
	errOrd int
}

func (e *embedderStub) Embedding(_ context.Context, text string) ([]float64, error) {
	if e.errOrd >= 0 {
		e.errOrd--
		if e.errOrd < 0 {
			return nil, errors.New("embedder error")
		}
	}
	return []float64{float64(len(text))}, nil
}

type writerStub struct {
	chunks []domain.Chunk
	err    error
}

func (w *writerStub) WriteChunks(_ context.Context, chunks []domain.Chunk) error {
	w.chunks = append(w.chunks, chunks...)
	return w.err
}

func TestPipelineIndex(t *testing.T) {
	cfg := &config.Config{}
	cfg.Pipeline.ChunkSize = 5
	cfg.Pipeline.ChunkOverlap = 0

	emb := &embedderStub{errOrd: 999}
	writer := &writerStub{}

	p := New(cfg, emb, writer)

	doc := domain.Document{ID: "doc", Content: "hello world"}
	if err := p.Index(context.Background(), doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.chunks) == 0 {
		t.Fatal("expected chunks written")
	}
}
