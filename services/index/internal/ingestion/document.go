package ingestion

import "github.com/artmexbet/raibecas/services/index/internal/domain"

type Fetcher interface {
	Fetch(docID string) (domain.Document, error)
}
