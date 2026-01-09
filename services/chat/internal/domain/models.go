package domain

import (
	"strings"
	"time"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
)

//go:generate easyjson -all models.go

const (
	AdditionalCountOfTokens = 512 // in tokens
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Done      bool      `json:"done"`
	Message   *Message  `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type Document struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata"`
}

// WorkingContext holds the context for a chat session, including chat history and retrieved documents.
type WorkingContext struct {
	Messages []Message  `json:"messages"`  // Chat history. May be empty.
	Docs     []Document `json:"documents"` // Retrieved documents. May be empty.
}

func (wc *WorkingContext) PrepareContext(query string, cfg config.ContextGeneration) string {
	sBuilder := strings.Builder{}
	// Preallocate memory to reduce allocations
	sBuilder.Grow(
		len(cfg.ContextPrompt) +
			len(query) +
			len(cfg.QueryPrompt) +
			AdditionalCountOfTokens,
	)
	// Build the context string
	sBuilder.WriteString(cfg.ContextPrompt)
	sBuilder.WriteString(cfg.QueryPrompt)
	sBuilder.WriteString(query)
	return sBuilder.String()
}
