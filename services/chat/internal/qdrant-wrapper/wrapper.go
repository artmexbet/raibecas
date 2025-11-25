package qdrantWrapper

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/qdrant/go-client/qdrant"
	"utils/pointer"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

type QdrantWrapper struct {
	client *qdrant.Client
	cfg    *config.Qdrant
}

// New creates a new QdrantWrapper instance with the provided configuration and Qdrant client.
// cfg is the Qdrant configuration, and client is the Qdrant client used for communication.
func New(cfg *config.Qdrant, client *qdrant.Client) *QdrantWrapper {
	return &QdrantWrapper{
		client: client,
		cfg:    cfg,
	}
}

// RetrieveVectors queries the Qdrant vector database for similar vectors based on the input vector.
// It takes a context for cancellation and a slice of float64 representing the query vector.
// Returns an error if the retrieval fails.
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
				Size:     q.cfg.VectorDimension,
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
			response[i].Metadata[key] = value.GetStringValue()
		}
	}

	return response, nil
}
