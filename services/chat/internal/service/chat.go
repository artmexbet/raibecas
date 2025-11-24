package service

import (
	"context"
	"fmt"

	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

type iVectorStore interface {
	RetrieveVectors(ctx context.Context, vector []float64) ([]domain.Document, error)
}

type iNeuroConnector interface {
	GenerateEmbeddings(ctx context.Context, input string) ([]float64, error)
	Chat(
		ctx context.Context,
		workingContext domain.WorkingContext,
		newPrompt string,
		fn func(domain.ChatResponse) error,
	) error
}

type iChatHistoryStore interface {
	RetrieveChatHistory(ctx context.Context, userID string) ([]domain.Message, error)
}

type Chat struct {
	vectorStore  iVectorStore
	neuro        iNeuroConnector
	historyStore iChatHistoryStore
}

func New(vectorStore iVectorStore, neuro iNeuroConnector, historyStore iChatHistoryStore) *Chat {
	return &Chat{
		vectorStore:  vectorStore,
		neuro:        neuro,
		historyStore: historyStore,
	}
}

func (c *Chat) ProcessInput(ctx context.Context, input, userID string, fn func(response domain.ChatResponse) error) error {
	embedding, err := c.neuro.GenerateEmbeddings(ctx, input)
	if err != nil {
		return fmt.Errorf("could not generate embeddings: %w", err)
	}

	docs, err := c.vectorStore.RetrieveVectors(ctx, embedding)
	if err != nil {
		return fmt.Errorf("could not retrieve vectors: %w", err)
	}

	//DONT RETRIEVE HISTORY FOR NOW //todo: implement saving and retrieving chat history
	//history, err := c.historyStore.RetrieveChatHistory(ctx, userID)
	//if err != nil {
	//	return fmt.Errorf("could not retrieve chat history: %w", err)
	//}

	workingContext := domain.WorkingContext{
		Messages: []domain.Message{}, //history,
		Docs:     docs,
	}
	//todo: save chat history. Maybe wrap fn to do it after each response chunk
	return c.neuro.Chat(ctx, workingContext, input, fn)
}
