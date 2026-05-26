package service

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	qdrantWrapper "github.com/artmexbet/raibecas/services/chat/internal/qdrant-wrapper"

	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

// searchVectorStore is the interface for vector search with scores.
type searchVectorStore interface {
	SearchWithScores(ctx context.Context, vector []float64, limit uint64) ([]qdrantWrapper.ScoredDocument, error)
}

// Search performs semantic search: generates an embedding for the query,
// searches Qdrant for similar chunks, groups them by document_id, and returns
// ranked results.
func (c *Chat) Search(ctx context.Context, query string, limit int) (*domain.SearchResponse, error) {
	ctx, span := c.tracer.Start(ctx, "chat.service.search",
		trace.WithAttributes(
			attribute.String("search.query", query),
			attribute.Int("search.limit", limit),
		),
	)
	defer span.End()

	if limit <= 0 {
		limit = 10
	}

	// Request more chunks from Qdrant than the final document limit,
	// because multiple chunks may belong to the same document.
	qdrantLimit := uint64(limit * 3) //nolint:mnd // fetch 3x chunks to group by document

	embedding, err := c.neuro.GenerateEmbeddings(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "embedding generation failed")
		return nil, fmt.Errorf("could not generate embeddings: %w", err)
	}

	scoredDocs, err := c.vectorStore.(searchVectorStore).SearchWithScores(ctx, embedding, qdrantLimit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "vector search failed")
		return nil, fmt.Errorf("could not search vectors: %w", err)
	}

	slog.DebugContext(ctx, "search: retrieved scored documents", "count", len(scoredDocs))

	results := groupByDocument(scoredDocs, limit)
	span.SetAttributes(attribute.Int("search.results_count", len(results)))

	return &domain.SearchResponse{
		Query:   query,
		Results: results,
		Total:   len(results),
	}, nil
}

// groupByDocument groups scored chunks by document_id, picks the best score
// per document, and returns up to limit results sorted by score descending.
func groupByDocument(docs []qdrantWrapper.ScoredDocument, limit int) []domain.SearchResult {
	type docEntry struct {
		result domain.SearchResult
		best   float32
	}

	byDoc := make(map[string]*docEntry)

	for _, d := range docs {
		docID, _ := d.Metadata["document_id"].(string)
		if docID == "" {
			continue
		}

		chunkText, _ := d.Metadata["chunk_text"].(string)
		ordinal, _ := d.Metadata["ordinal"].(string)
		ord, _ := strconv.Atoi(ordinal)

		chunk := domain.SearchChunk{
			Text:    chunkText,
			Score:   d.Score,
			Ordinal: ord,
		}

		entry, exists := byDoc[docID]
		if !exists {
			title, _ := d.Metadata["title"].(string)
			metadata := extractSearchMetadata(d.Metadata)

			entry = &docEntry{
				result: domain.SearchResult{
					DocumentID: docID,
					Title:      title,
					Score:      d.Score,
					Chunks:     []domain.SearchChunk{chunk},
					Metadata:   metadata,
				},
				best: d.Score,
			}
			byDoc[docID] = entry
		} else {
			entry.result.Chunks = append(entry.result.Chunks, chunk)
			if d.Score > entry.best {
				entry.best = d.Score
				entry.result.Score = d.Score
			}
		}
	}

	results := make([]domain.SearchResult, 0, len(byDoc))
	for _, entry := range byDoc {
		// Sort chunks within each document by score descending
		slices.SortFunc(entry.result.Chunks, func(a, b domain.SearchChunk) int {
			return cmp.Compare(b.Score, a.Score) // descending
		})
		results = append(results, entry.result)
	}

	// Sort documents by best score descending
	slices.SortFunc(results, func(a, b domain.SearchResult) int {
		return cmp.Compare(b.Score, a.Score) // descending
	})

	// Trim to limit
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// extractSearchMetadata picks relevant metadata fields for the search response,
// excluding internal fields like chunk_text and ordinal.
func extractSearchMetadata(m map[string]any) map[string]string {
	keys := []string{
		"document_type", "publication_date", "description",
		"participant_names", "participant_roles", "tag_titles",
		"category_id", "document_type_id", "version",
	}

	result := make(map[string]string, len(keys))
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			result[k] = v
		}
	}
	return result
}
