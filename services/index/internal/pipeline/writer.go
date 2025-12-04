package pipeline

import (
	"context"

	"github.com/artmexbet/raibecas/services/index/internal/domain"
	"github.com/artmexbet/raibecas/services/index/internal/qdrant"
)

type QdrantWriter struct {
	client *qdrant.Client
}

func NewQdrantWriter(client *qdrant.Client) *QdrantWriter {
	return &QdrantWriter{client: client}
}

func (w *QdrantWriter) WriteChunks(ctx context.Context, chunks []domain.Chunk) error {
	points := qdrant.PointsFromChunks(chunks)
	return w.client.UpsertChunks(ctx, points)
}
