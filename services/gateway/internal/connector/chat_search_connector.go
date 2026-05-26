package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

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
