package ingestion

import "github.com/artmexbet/raibecas/services/index/internal/domain"

type Fetcher interface {
	Fetch(docID string) (domain.Document, error)
}

// IndexRequest представляет запрос на индексацию документа через JSON API
type IndexRequest struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	SourceURI string            `json:"source_uri"`
	Metadata  map[string]string `json:"metadata"`
}
