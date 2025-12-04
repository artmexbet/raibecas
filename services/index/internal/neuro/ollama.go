package neuro

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	ollamaApi "github.com/ollama/ollama/api"

	"github.com/artmexbet/raibecas/services/index/internal/config"
)

type Connector struct {
	client *ollamaApi.Client
	cfg    *config.Ollama
}

func NewConnector(cfg *config.Ollama) (*Connector, error) {
	httpClient := &http.Client{Timeout: cfg.Timeout}
	parsed, err := url.Parse(cfg.Address())
	if err != nil {
		return nil, fmt.Errorf("invalid ollama url: %w", err)
	}
	client := ollamaApi.NewClient(parsed, httpClient)
	return &Connector{client: client, cfg: cfg}, nil
}

func (c *Connector) Embedding(ctx context.Context, text string) ([]float64, error) {
	req := &ollamaApi.EmbeddingRequest{Model: c.cfg.EmbeddingModel, Prompt: text}
	resp, err := c.client.Embeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate embeddings: %w", err)
	}
	return resp.Embedding, nil
}

func (c *Connector) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := c.client.Heartbeat(ctx)
	return err
}
