package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ChatHTTPConnector manages HTTP requests to the Chat service REST API.
type ChatHTTPConnector struct {
	baseURL    string
	httpClient *http.Client
}

// ChatSession represents a chat session returned by the chat service.
type ChatSession struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id"`
	Title     string        `json:"title"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
	Messages  []ChatMessage `json:"messages"`
}

// ChatMessage represents a single message in a chat session.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewChatHTTPConnector creates a new ChatHTTPConnector.
func NewChatHTTPConnector(baseURL string) *ChatHTTPConnector {
	return &ChatHTTPConnector{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetUserSessions fetches all chat sessions for a user.
func (c *ChatHTTPConnector) GetUserSessions(ctx context.Context, userID string) ([]ChatSession, error) {
	reqURL := fmt.Sprintf("%s/api/v1/chat/%s/sessions", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("GetUserSessions request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GetUserSessions do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // safe to ignore

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GetUserSessions: status %d, body: %s", resp.StatusCode, body)
	}

	var sessions []ChatSession
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("GetUserSessions decode: %w", err)
	}
	return sessions, nil
}

// CreateSession creates a new chat session for a user.
func (c *ChatHTTPConnector) CreateSession(ctx context.Context, userID, title string) (string, error) {
	reqURL := fmt.Sprintf("%s/api/v1/chat/%s/sessions", c.baseURL, userID)

	body, err := json.Marshal(map[string]string{"title": title})
	if err != nil {
		return "", fmt.Errorf("CreateSession marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("CreateSession request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("CreateSession do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // safe to ignore

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("CreateSession: status %d, body: %s", resp.StatusCode, b)
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("CreateSession decode: %w", err)
	}
	return result.SessionID, nil
}

// SearchResponse mirrors the chat service search response.
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// SearchResult represents a single document in search results.
type SearchResult struct {
	DocumentID string            `json:"document_id"`
	Title      string            `json:"title"`
	Score      float32           `json:"score"`
	Chunks     []SearchChunk     `json:"chunks"`
	Metadata   map[string]string `json:"metadata"`
}

// SearchChunk represents a text chunk within a search result.
type SearchChunk struct {
	Text    string  `json:"text"`
	Score   float32 `json:"score"`
	Ordinal int     `json:"ordinal"`
}

// Search performs a semantic search via the chat service.
func (c *ChatHTTPConnector) Search(ctx context.Context, query string, limit int) (*SearchResponse, error) {
	reqURL := fmt.Sprintf("%s/api/v1/search?q=%s&limit=%d", c.baseURL, url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Search request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Search do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // safe to ignore

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Search: status %d, body: %s", resp.StatusCode, body)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("Search decode: %w", err)
	}
	return &result, nil
}
