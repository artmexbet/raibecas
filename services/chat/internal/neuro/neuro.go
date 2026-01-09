package neuro

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	ollamaApi "github.com/ollama/ollama/api"

	"github.com/artmexbet/raibecas/libs/utils/pointer"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

const (
	roleSystem = "system"
	roleUser   = "user"

	contextContentKey = "chunk_text"
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
	docs := workingContext.Docs

	slog.DebugContext(ctx, "Chat request", "history_length", len(chatHistory), "new_prompt", newPrompt, "docs_count", len(docs))

	// Construct messages for the chat request
	msgs := make([]ollamaApi.Message, 0, len(docs)+len(chatHistory)+2)
	msgs = append(msgs, ollamaApi.Message{
		Role:    roleSystem,
		Content: e.cfg.Context.BasePrompt,
	})

	for _, doc := range docs {
		if content, err := e.prepareDoc(doc); err == nil {
			msgs = append(msgs, ollamaApi.Message{
				Role:    roleSystem,
				Content: content, // Add context document content
			})
		} else if errors.Is(err, domain.ErrDocumentWithoutContent) {
			slog.WarnContext(ctx, "document without content", "doc_id", doc.ID)
		}
	}

	for _, m := range chatHistory {
		msgs = append(msgs, ollamaApi.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	preparedContext := workingContext.PrepareContext(newPrompt, e.cfg.Context)

	// Add the new user prompt with prepared context (e.g., including relevant documents)
	msgs = append(msgs, ollamaApi.Message{
		Role:    roleUser,
		Content: preparedContext,
	})

	reqData := &ollamaApi.ChatRequest{
		Model:    e.cfg.GenerationModel,
		Messages: msgs,
		Stream:   pointer.To(e.cfg.StreamAnswers),
		Options: map[string]interface{}{
			"temperature": e.cfg.Temperature, // Controls randomness in generation
		},
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

func (e *Connector) prepareDoc(doc domain.Document) (string, error) {
	content, ok := doc.Metadata[contextContentKey].(string)
	if !ok {
		return "", domain.ErrDocumentWithoutContent
	}
	sBuilder := strings.Builder{}
	// Preallocate memory to reduce allocations
	sBuilder.Grow(len(doc.Metadata)*256 + len(content) + 64)
	sBuilder.WriteString("Context document:\n")
	for key, value := range doc.Metadata {
		if key == "content" {
			continue
		}
		sBuilder.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}
	sBuilder.WriteString("Content:\n")
	sBuilder.WriteString(content)
	return sBuilder.String(), nil

}
