package qdrant

import (
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

func PointsFromChunks(chunks []domain.Chunk) []*qdrant.PointStruct {
	result := make([]*qdrant.PointStruct, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk.Embedding) == 0 {
			continue
		}
		vector := make([]float32, len(chunk.Embedding))
		for i, v := range chunk.Embedding {
			vector[i] = float32(v)
		}
		payload := map[string]*qdrant.Value{
			"document_id": qdrant.NewValueString(chunk.DocumentID),
			"ordinal":     qdrant.NewValueInt(int64(chunk.Ordinal)),
			"text":        qdrant.NewValueString(chunk.Text),
		}
		for k, v := range chunk.Metadata {
			payload[k] = qdrant.NewValueString(v)
		}
		result = append(result, &qdrant.PointStruct{
			Id:      qdrant.NewID(uuid.NewString()),
			Vectors: qdrant.NewVectors(vector...),
			Payload: payload,
		})
	}
	return result
}
