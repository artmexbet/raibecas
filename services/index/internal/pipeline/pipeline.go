package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/artmexbet/raibecas/services/index/internal/chunker"
	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

type embedder interface {
	Embedding(ctx context.Context, text string) ([]float64, error)
}

type chunkWriter interface {
	WriteChunks(ctx context.Context, chunks []domain.Chunk) error
}

type Pipeline struct {
	cfg        config.Pipeline
	chunkerCfg chunker.Config
	embedder   embedder
	writer     chunkWriter
}

func New(cfg *config.Config, emb embedder, writer chunkWriter) *Pipeline {
	c := chunker.Config{ChunkSize: cfg.Pipeline.ChunkSize, ChunkOverlap: cfg.Pipeline.ChunkOverlap, MaxChunks: cfg.Pipeline.MaxChunks}
	return &Pipeline{cfg: cfg.Pipeline, chunkerCfg: c, embedder: emb, writer: writer}
}

func (p *Pipeline) Index(ctx context.Context, doc domain.Document) error {
	chunks := chunker.SplitText(p.chunkerCfg, doc.Content)
	if len(chunks) == 0 {
		return fmt.Errorf("document %s has no chunks", doc.ID)
	}

	result := make([]domain.Chunk, 0, len(chunks))
	for _, ch := range chunks {
		embedding, err := p.embedder.Embedding(ctx, ch.Text)
		if err != nil {
			return fmt.Errorf("embed chunk %d: %w", ch.Ordinal, err)
		}
		result = append(result, domain.Chunk{
			DocumentID: doc.ID,
			Ordinal:    ch.Ordinal,
			Text:       ch.Text,
			Embedding:  embedding,
			Metadata:   doc.Metadata,
		})
	}

	slog.InfoContext(ctx, "pipeline finished", slog.String("document_id", doc.ID), slog.Int("chunks", len(result)))
	return p.writer.WriteChunks(ctx, result)
}
