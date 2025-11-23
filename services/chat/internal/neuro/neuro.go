package neuro

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"utils/pointer"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"

	ollamaApi "github.com/ollama/ollama/api"
)

type Connector struct {
	client    *ollamaApi.Client
	cfg       *config.Ollama
	baseRoute string
}

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

// GenerateEmbeddings send a request to Ollama to generate embeddings for the given input text.
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
	chatHistory []domain.Message,
	newPrompt string,
	fn func(domain.ChatResponse) error,
) error {
	msgs := make([]ollamaApi.Message, len(chatHistory)+1)
	for i, m := range chatHistory {
		msgs[i] = ollamaApi.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	msgs[len(chatHistory)] = ollamaApi.Message{
		Role:    "user",
		Content: newPrompt,
	}

	reqData := &ollamaApi.ChatRequest{
		Model:    e.cfg.EmbeddingModel,
		Messages: msgs,
		Stream:   pointer.To(true),
	}
	err := e.client.Chat(ctx, reqData, func(resp ollamaApi.ChatResponse) error {
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
