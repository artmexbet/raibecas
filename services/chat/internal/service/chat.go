package service

import (
	"context"
	"fmt"
)

type iVectorStore interface {
	RetrieveVectors(ctx context.Context, vector []float64) error
}

type iNeuroConnector interface {
	GenerateEmbeddings(ctx context.Context, input string) ([]float64, error)
}

type Chat struct {
	vectorStore iVectorStore
	embedder    iNeuroConnector
}

func New(vectorStore iVectorStore, embedder iNeuroConnector) *Chat {
	return &Chat{
		vectorStore: vectorStore,
		embedder:    embedder,
	}
}

func (c *Chat) ProcessInput(ctx context.Context, input string) error {
	embedding, err := c.embedder.GenerateEmbeddings(ctx, input)
	if err != nil {
		return fmt.Errorf("could not generate embeddings: %w", err)
	}

	err = c.vectorStore.RetrieveVectors(ctx, embedding)
	if err != nil {
		return fmt.Errorf("could not retrieve vectors: %w", err)
	}

	return nil
}
