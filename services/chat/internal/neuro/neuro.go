package neuro

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	ollamaApi "github.com/ollama/ollama/api"
	"utils/pointer"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

type Connector struct {
	client    *ollamaApi.Client
	cfg       *config.Ollama
	baseRoute string
}

// NewConnector creates a new Connector for interacting with the Ollama API.
// cfg is the Ollama configuration containing the API address and model settings.
// Returns a Connector instance or an error if the address is invalid.
func NewConnector(cfg *config.Ollama) (*Connector, error) {
	httpClient := &http.Client{}
	u, err := url.Parse(cfg.GetAddress())
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama address: %w", err)
	}

	client := ollamaApi.NewClient(u, httpClient)
	return &Connector{
		client:    client,
		cfg:       cfg,
		baseRoute: cfg.GetAddress(),
	}, nil
}

// GenerateEmbeddings sends a request to Ollama to generate embeddings for the given input text.
func (e *Connector) GenerateEmbeddings(ctx context.Context, input string) ([]float64, error) {
	reqData := &ollamaApi.EmbeddingRequest{
		Model:  e.cfg.EmbeddingModel,
		Prompt: input,
	}
	resp, err := e.client.Embeddings(ctx, reqData)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	return resp.Embedding, nil
}

// Chat sends a chat request to Ollama with the provided chat history and new prompt.
// - fn is a callback function that processes each chunk of the streaming response.
func (e *Connector) Chat(
	ctx context.Context,
	workingContext domain.WorkingContext,
	newPrompt string,
	fn func(domain.ChatResponse) error,
) error {
	// Prepare chat messages history
	chatHistory := workingContext.Messages
	msgs := make([]ollamaApi.Message, len(chatHistory)+1)
	for i, m := range chatHistory {
		msgs[i] = ollamaApi.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	preparedContext := workingContext.PrepareContext(newPrompt, e.cfg.Context)
	slog.DebugContext(ctx, "Prepared context", "context", preparedContext)

	// Add the new user prompt with prepared context (e.g., including relevant documents)
	msgs[len(chatHistory)] = ollamaApi.Message{
		Role:    "user",
		Content: preparedContext,
	}

	reqData := &ollamaApi.ChatRequest{
		Model:    e.cfg.GenerationModel,
		Messages: msgs,
		Stream:   pointer.To(e.cfg.StreamAnswers),
	}
	err := e.client.Chat(ctx, reqData, func(resp ollamaApi.ChatResponse) error {
		// Map ollamaApi.ChatResponse to domain.ChatResponse for every chunk
		return fn(domain.ChatResponse{
			Done:      resp.Done,
			CreatedAt: resp.CreatedAt,
			Message: &domain.Message{
				Role:    resp.Message.Role,
				Content: resp.Message.Content,
			},
		})
	})

	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	return nil
}
