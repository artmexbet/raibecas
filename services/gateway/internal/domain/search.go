package domain

// SearchQuery represents the query parameters for a semantic search request.
type SearchQuery struct {
	Q     string `query:"q" validate:"required,min=1"`
	Limit int    `query:"limit" validate:"omitempty,min=1,max=50"`
}

// SearchChunk represents a single text chunk found during semantic search.
type SearchChunk struct {
	Text    string  `json:"text"`
	Score   float32 `json:"score"`
	Ordinal int     `json:"ordinal"`
}

// SearchResult represents a single document found during semantic search.
type SearchResult struct {
	DocumentID string            `json:"document_id"`
	Title      string            `json:"title"`
	Score      float32           `json:"score"`
	Chunks     []SearchChunk     `json:"chunks"`
	Metadata   map[string]string `json:"metadata"`
}

// SearchResponse is the top-level response for a semantic search query.
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}
