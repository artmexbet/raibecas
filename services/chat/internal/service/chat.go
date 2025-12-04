package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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
	SaveMessage(ctx context.Context, userID string, message domain.Message) error
	ClearChatHistory(ctx context.Context, userID string) error
	GetChatSize(ctx context.Context, userID string) (int, error)
}

type Chat struct {
	vectorStore  iVectorStore
	neuro        iNeuroConnector
	historyStore iChatHistoryStore
}

// New creates a new Chat service with the provided vector store and embedder.
// vectorStore is used to retrieve vectors, and embedder is used to generate embeddings.
func New(vectorStore iVectorStore, neuro iNeuroConnector, historyStore iChatHistoryStore) *Chat {
	return &Chat{
		vectorStore:  vectorStore,
		neuro:        neuro,
		historyStore: historyStore,
	}
}

// ProcessInput generates embeddings for the input text using the embedder,
// retrieves similar vectors from the vector store, and returns an error if any step fails.
// Parameters:
//   - ctx: context for cancellation and deadlines.
//   - input: the input text to process.
//
// Returns:
//   - error: non-nil if embedding generation or vector retrieval fails.
func (c *Chat) ProcessInput(ctx context.Context, input, userID string, fn func(response domain.ChatResponse) error) error {
	embedding, err := c.neuro.GenerateEmbeddings(ctx, input)
	if err != nil {
		return fmt.Errorf("could not generate embeddings: %w", err)
	}

	docs, err := c.vectorStore.RetrieveVectors(ctx, embedding)
	if err != nil {
		return fmt.Errorf("could not retrieve vectors: %w", err)
	}
	slog.DebugContext(ctx, "retrieved documents", "count", len(docs), "docs", docs)

	// Retrieve chat history from Redis
	history, err := c.historyStore.RetrieveChatHistory(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "could not retrieve chat history", "error", err)
		history = []domain.Message{}
	}

	// Add user message to history
	userMessage := domain.Message{
		Role:    "user",
		Content: input,
	}
	err = c.historyStore.SaveMessage(ctx, userID, userMessage)
	if err != nil {
		slog.WarnContext(ctx, "could not save user message", "error", err)
	}

	workingContext := domain.WorkingContext{
		Messages: history,
		Docs:     docs,
	}

	// Process response with message chunking and saving
	assistantContent := strings.Builder{}
	err = c.neuro.Chat(ctx, workingContext, input, func(response domain.ChatResponse) error {
		// Accumulate message chunks
		if response.Message != nil && response.Message.Content != "" {
			assistantContent.WriteString(response.Message.Content)
		}

		// If message is complete, save full message to history
		if response.Done {
			fullMessage := domain.Message{
				Role:    "assistant",
				Content: assistantContent.String(),
			}
			slog.DebugContext(ctx, "Sending chat response chunk")
			err := c.historyStore.SaveMessage(ctx, userID, fullMessage)
			if err != nil {
				slog.WarnContext(ctx, "could not save assistant message", "error", err)
			}
		}

		// Pass response to handler (streaming)
		return fn(response)
	})

	return err
}

// ClearUserChat clears all chat history for a user
func (c *Chat) ClearUserChat(ctx context.Context, userID string) error {
	return c.historyStore.ClearChatHistory(ctx, userID)
}
