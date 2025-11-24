package main

import (
	"context"
	"log/slog"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/neuro"
	"github.com/qdrant/go-client/qdrant"
)

func main() {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334,
	})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	if _, err := client.HealthCheck(ctx); err != nil {
		panic(err)
	}

	if exists, err := client.CollectionExists(ctx, "books"); err != nil || !exists {
		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: "books",
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     768, //todo:make configurable
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			panic(err)
		}
	}

	documents := []string{
		"По выражению одного из основателей языков программирования Никлауса Вирта «Программы = алгоритмы + структуры данных»[1][2].\n\nПрограммирование основывается на использовании языков программирования и средств программирования. В основном языки программирования основаны на текстовом представлении программ, но иногда программировать можно, используя, например, визуальное программирование или «zero-code» программирование.",
	}

	ne, err := neuro.NewConnector(&config.Ollama{
		Protocol:        "http",
		Host:            "localhost",
		Port:            "11434",
		EmbeddingModel:  "embeddinggemma",
		GenerationModel: "gemma3:4b",
	})
	if err != nil {
		panic(err)
	}

	for i, doc := range documents {
		vec, err := ne.GenerateEmbeddings(ctx, doc)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to generate embedding", slog.String("document", doc), "error", err)
			continue
		}
		res, err := client.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: "books",
			Points: []*qdrant.PointStruct{
				{
					Id:      qdrant.NewIDNum(uint64(i)),
					Payload: qdrant.NewValueMap(map[string]any{"content": doc}),
					Vectors: qdrant.NewVectors(float64ToVectors(vec)...),
				},
			},
		})
		if err != nil {
			slog.ErrorContext(ctx, "Failed to upsert point", slog.String("document", doc), "error", err)
		}

		slog.InfoContext(ctx, "Upserted document", slog.Int("id", i), slog.String("document", doc), slog.Any("result", res))
	}
}

func float64ToVectors(input []float64) []float32 {
	output := make([]float32, len(input))
	for i, v := range input {
		output[i] = float32(v)
	}
	return output
}
