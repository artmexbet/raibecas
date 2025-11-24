package domain

import (
	"strings"
	"time"
)

//go:generate easyjson -all models.go

const (
	MaxDocumentLength       = 768 // in characters todo: make configurable
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

func (wc *WorkingContext) PrepareContext(query string) string {
	sBuilder := strings.Builder{}
	sBuilder.Grow(len(wc.Docs)*MaxDocumentLength +
		len(query) + AdditionalCountOfTokens)
	sBuilder.WriteString("Context: ")
	for _, doc := range wc.Docs {
		if content, ok := doc.Metadata["content"].(string); ok {
			sBuilder.WriteString(content)
			sBuilder.WriteString("\n")
		}
	}
	sBuilder.WriteString("Query: ")
	sBuilder.WriteString(query)
	return sBuilder.String()
}
