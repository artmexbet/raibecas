package domain

// SearchChunk represents a single text chunk found during semantic search.
type SearchChunk struct {
	Text    string  `json:"text"`
	Score   float32 `json:"score"`
	Ordinal int     `json:"ordinal"`
}

// SearchResult represents a single document found during semantic search,
// potentially containing multiple relevant chunks.
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
