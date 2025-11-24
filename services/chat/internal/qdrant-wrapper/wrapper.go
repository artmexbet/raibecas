package qdrantWrapper

import (
	"context"
	"fmt"
	"log/slog"

	"utils/pointer"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"

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

func (q *QdrantWrapper) CheckConnection(ctx context.Context) error {
	exists, err := q.client.CollectionExists(ctx, q.cfg.CollectionName)
	if err != nil {
		return fmt.Errorf("error checking collection existence: %w", err)
	}
	if !exists {
		slog.InfoContext(ctx, "connection does not exist, creating new collection",
			slog.String("collection", q.cfg.CollectionName))
		return q.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: q.cfg.CollectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     768, //todo:make configurable
				Distance: qdrant.Distance_Cosine,
			}),
		})
	}
	return nil
}

func (q *QdrantWrapper) RetrieveVectors(ctx context.Context, vector []float64) ([]domain.Document, error) {
	convertedVector := make([]float32, len(vector))
	for i, v := range vector {
		convertedVector[i] = float32(v)
	}
	vector = nil // to avoid accidental usage

	result, err := q.client.Query(
		ctx,
		&qdrant.QueryPoints{
			CollectionName: q.cfg.CollectionName,
			Query:          qdrant.NewQuery(convertedVector...),
			WithPayload:    qdrant.NewWithPayload(q.cfg.RetrievePayload),
			Limit:          pointer.To(q.cfg.CountOfResults),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve the vectors: %w", err)
	}

	response := make([]domain.Document, len(result))
	for i, v := range result {
		response[i] = domain.Document{
			ID:       v.Id.String(),
			Metadata: make(map[string]interface{}),
		}
		for key, value := range v.Payload {
			response[i].Metadata[key] = value //todo: check the type of value. May be need to convert
		}
	}

	return response, nil
}
