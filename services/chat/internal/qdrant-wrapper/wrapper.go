package qdrantWrapper

import (
	"context"
	"fmt"

	"utils/pointer"

	"github.com/artmexbet/raibecas/services/chat/internal/config"

	"github.com/qdrant/go-client/qdrant"
)

type QdrantWrapper struct {
	client *qdrant.Client
	cfg    *config.Qdrant
}

func New(cfg *config.Qdrant, client *qdrant.Client) *QdrantWrapper {
	return &QdrantWrapper{
		client: client,
		cfg:    cfg,
	}
}

func (q *QdrantWrapper) RetrieveVectors(ctx context.Context, vector []float64) error {
	convertedVector := make([]float32, len(vector))
	for i, v := range vector {
		convertedVector[i] = float32(v)
	}
	vector = nil // to avoid accidental usage

	_, err := q.client.Query( //todo: use the result, map to domain model and then return
		ctx,
		&qdrant.QueryPoints{
			CollectionName: q.cfg.CollectionName,
			Query:          qdrant.NewQuery(convertedVector...),
			WithPayload:    qdrant.NewWithPayload(q.cfg.RetrievePayload),
			Limit:          pointer.To(q.cfg.CountOfResults),
		},
	)
	if err != nil {
		return fmt.Errorf("cannot retrieve the vectors: %w", err)
	}
	return nil
}
