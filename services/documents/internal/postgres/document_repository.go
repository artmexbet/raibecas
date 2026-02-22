package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres/queries"
)

// DocumentRepository implements repository.DocumentRepository using PostgreSQL
type DocumentRepository struct {
	queries *queries.Queries
}

// NewDocumentRepository creates a new PostgreSQL document repository
func NewDocumentRepository(queries *queries.Queries) *DocumentRepository {
	return &DocumentRepository{queries: queries}
}

// Create creates a new document
func (r *DocumentRepository) Create(ctx context.Context, doc *domain.Document) error {
	created, err := r.queries.CreateDocument(ctx, queries.CreateDocumentParams{
		Title:           doc.Title,
		Description:     doc.Description,
		AuthorID:        doc.AuthorID,
		CategoryID:      int32(doc.CategoryID),
		PublicationDate: timeToDate(doc.PublicationDate),
		ContentPath:     doc.ContentPath,
		CurrentVersion:  int32(doc.CurrentVersion),
	})
	if err != nil {
		return fmt.Errorf("create document: %w", err)
	}

	doc.ID = created.ID
	doc.CreatedAt = created.CreatedAt
	doc.UpdatedAt = created.UpdatedAt
	return nil
}

// GetByID retrieves a document by ID
func (r *DocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	row, err := r.queries.GetDocumentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get document by id: %w", err)
	}

	// Convert row to domain with embedded entities
	doc := &domain.Document{
		ID:              row.ID,
		Title:           row.Title,
		Description:     row.Description,
		AuthorID:        row.AuthorID,
		CategoryID:      int(row.CategoryID),
		PublicationDate: row.PublicationDate.Time,
		ContentPath:     row.ContentPath,
		CurrentVersion:  int(row.CurrentVersion),
		Indexed:         row.Indexed,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	// Add author if present
	if row.Author.ID != uuid.Nil {
		doc.Author = &domain.Author{
			ID:        row.Author.ID,
			Name:      row.Author.Name,
			Bio:       row.Author.Bio,
			CreatedAt: row.Author.CreatedAt,
			UpdatedAt: row.Author.UpdatedAt,
		}
	}

	// Add category if present
	if row.Category.ID != 0 {
		doc.Category = &domain.Category{
			ID:          int(row.Category.ID),
			Title:       row.Category.Title,
			Description: row.Category.Description,
			CreatedAt:   row.Category.CreatedAt,
		}
	}

	// Load tags
	tags, err := r.queries.GetDocumentTags(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document tags: %w", err)
	}

	doc.Tags = make([]domain.Tag, len(tags))
	for i, tag := range tags {
		doc.Tags[i] = domain.Tag{
			ID:        int(tag.ID),
			Title:     tag.Title,
			CreatedAt: tag.CreatedAt,
		}
	}

	return doc, nil
}

// List retrieves documents with filters
func (r *DocumentRepository) List(ctx context.Context, params domain.ListDocumentsParams) ([]domain.Document, error) {
	rows, err := r.queries.ListDocuments(ctx, queries.ListDocumentsParams{
		Limit:      int32(params.Limit),
		Offset:     int32(params.Offset),
		AuthorID:   params.AuthorID,
		CategoryID: params.CategoryID,
		Search:     convertStringToPtr(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	result := make([]domain.Document, len(rows))
	for i, row := range rows {
		doc := domain.Document{
			ID:              row.ID,
			Title:           row.Title,
			Description:     row.Description,
			AuthorID:        row.AuthorID,
			CategoryID:      int(row.CategoryID),
			PublicationDate: row.PublicationDate.Time,
			ContentPath:     row.ContentPath,
			CurrentVersion:  int(row.CurrentVersion),
			Indexed:         row.Indexed,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}

		// Add author if present
		if row.Author.ID != uuid.Nil {
			doc.Author = &domain.Author{
				ID:        row.Author.ID,
				Name:      row.Author.Name,
				Bio:       row.Author.Bio,
				CreatedAt: row.Author.CreatedAt,
				UpdatedAt: row.Author.UpdatedAt,
			}
		}

		// Add category if present
		if row.Category.ID != 0 {
			doc.Category = &domain.Category{
				ID:          int(row.Category.ID),
				Title:       row.Category.Title,
				Description: row.Category.Description,
				CreatedAt:   row.Category.CreatedAt,
			}
		}

		// Load tags for this document
		tags, err := r.queries.GetDocumentTags(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("get document tags for %s: %w", row.ID, err)
		}

		doc.Tags = make([]domain.Tag, len(tags))
		for j, tag := range tags {
			doc.Tags[j] = domain.Tag{
				ID:        int(tag.ID),
				Title:     tag.Title,
				CreatedAt: tag.CreatedAt,
			}
		}

		result[i] = doc
	}
	return result, nil
}

// Count counts documents
func (r *DocumentRepository) Count(ctx context.Context, params domain.ListDocumentsParams) (int, error) {
	count, err := r.queries.CountDocuments(ctx, queries.CountDocumentsParams{
		AuthorID:   params.AuthorID,
		CategoryID: params.CategoryID,
		Search:     convertStringToPtr(params.Search),
	})
	if err != nil {
		return 0, fmt.Errorf("count documents: %w", err)
	}
	return int(count), nil
}

// Update updates a document
func (r *DocumentRepository) Update(ctx context.Context, doc *domain.Document) error {
	categoryID := int32(doc.CategoryID)
	currentVersion := int32(doc.CurrentVersion)

	updated, err := r.queries.UpdateDocument(ctx, queries.UpdateDocumentParams{
		ID:              doc.ID,
		Title:           &doc.Title,
		Description:     doc.Description,
		AuthorID:        &doc.AuthorID,
		CategoryID:      &categoryID,
		PublicationDate: timeToDate(doc.PublicationDate),
		ContentPath:     &doc.ContentPath,
		CurrentVersion:  &currentVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("update document: %w", err)
	}
	doc.UpdatedAt = updated.UpdatedAt
	return nil
}

// Delete deletes a document
func (r *DocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteDocument(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete document: %w", err)
	}
	return nil
}

// UpdateIndexedStatus updates indexed status
func (r *DocumentRepository) UpdateIndexedStatus(ctx context.Context, id uuid.UUID, indexed bool) error {
	if err := r.queries.UpdateDocumentIndexed(ctx, queries.UpdateDocumentIndexedParams{
		ID:      id,
		Indexed: indexed,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("update indexed status: %w", err)
	}
	return nil
}

// toDomain converts to domain model
//
// nolint:unused
func (r *DocumentRepository) toDomain(doc *queries.Document) *domain.Document {
	return &domain.Document{
		ID:              doc.ID,
		Title:           doc.Title,
		Description:     doc.Description,
		AuthorID:        doc.AuthorID,
		CategoryID:      int(doc.CategoryID),
		PublicationDate: doc.PublicationDate.Time,
		ContentPath:     doc.ContentPath,
		CurrentVersion:  int(doc.CurrentVersion),
		Indexed:         doc.Indexed,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
	}
}

// Helper functions
func timeToDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func convertStringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
