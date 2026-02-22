package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres/queries"
)

// TagRepository implements repository.TagRepository using PostgreSQL
type TagRepository struct {
	queries *queries.Queries
}

// NewTagRepository creates a new PostgreSQL tag repository
func NewTagRepository(queries *queries.Queries) *TagRepository {
	return &TagRepository{queries: queries}
}

// AddToDocument adds a tag to a document
func (r *TagRepository) AddToDocument(ctx context.Context, documentID uuid.UUID, tagID int) error {
	if err := r.queries.AddDocumentTag(ctx, queries.AddDocumentTagParams{
		DocumentID: documentID,
		TagID:      int32(tagID),
	}); err != nil {
		return fmt.Errorf("add tag to document: %w", err)
	}
	return nil
}

// RemoveFromDocument removes a tag from a document
func (r *TagRepository) RemoveFromDocument(ctx context.Context, documentID uuid.UUID, tagID int) error {
	if err := r.queries.RemoveDocumentTag(ctx, queries.RemoveDocumentTagParams{
		DocumentID: documentID,
		TagID:      int32(tagID),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("remove tag from document: %w", err)
	}
	return nil
}

// ClearDocument removes all tags from a document
func (r *TagRepository) ClearDocument(ctx context.Context, documentID uuid.UUID) error {
	if err := r.queries.ClearDocumentTags(ctx, documentID); err != nil {
		return fmt.Errorf("clear document tags: %w", err)
	}
	return nil
}

// GetByDocumentID retrieves all tags for a document
func (r *TagRepository) GetByDocumentID(ctx context.Context, documentID uuid.UUID) ([]domain.Tag, error) {
	tags, err := r.queries.GetDocumentTags(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("get document tags: %w", err)
	}

	result := make([]domain.Tag, len(tags))
	for i, tag := range tags {
		result[i] = domain.Tag{
			ID:        int(tag.ID),
			Title:     tag.Title,
			CreatedAt: tag.CreatedAt,
		}
	}

	return result, nil
}
