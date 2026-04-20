package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres/queries"
)

// MetadataRepository implements metadata repository using PostgreSQL with sqlc
type MetadataRepository struct {
	queries *queries.Queries
}

// NewMetadataRepository creates a new metadata repository
func NewMetadataRepository(q *queries.Queries) *MetadataRepository {
	return &MetadataRepository{
		queries: q,
	}
}

// ListAuthors retrieves all authors
func (r *MetadataRepository) ListAuthors(ctx context.Context) ([]domain.Author, error) {
	rows, err := r.queries.ListAuthors(ctx)
	if err != nil {
		return nil, fmt.Errorf("list authors: %w", err)
	}

	authors := make([]domain.Author, len(rows))
	for i, row := range rows {
		authors[i] = domain.Author{
			ID:        row.ID,
			Name:      row.Name,
			Bio:       row.Bio,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}

	return authors, nil
}

// CreateAuthor creates a new author
func (r *MetadataRepository) CreateAuthor(ctx context.Context, name string) (*domain.Author, error) {
	id := uuid.New()
	now := time.Now()

	row, err := r.queries.CreateAuthor(ctx, queries.CreateAuthorParams{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("create author: %w", err)
	}

	return &domain.Author{
		ID:        row.ID,
		Name:      row.Name,
		Bio:       row.Bio,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

// GetAuthorByID retrieves an author by ID
func (r *MetadataRepository) GetAuthorByID(ctx context.Context, id uuid.UUID) (*domain.Author, error) {
	row, err := r.queries.GetAuthorByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get author: %w", err)
	}

	return &domain.Author{
		ID:        row.ID,
		Name:      row.Name,
		Bio:       row.Bio,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

// ListCategories retrieves all categories
func (r *MetadataRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.queries.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	categories := make([]domain.Category, len(rows))
	for i, row := range rows {
		categories[i] = domain.Category{
			ID:          int(row.ID),
			Title:       row.Title,
			Description: row.Description,
			CreatedAt:   row.CreatedAt,
		}
	}

	return categories, nil
}

// CreateCategory creates a new category
func (r *MetadataRepository) CreateCategory(ctx context.Context, title string) (*domain.Category, error) {
	now := time.Now()

	row, err := r.queries.CreateCategory(ctx, queries.CreateCategoryParams{
		Title:     title,
		CreatedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	return &domain.Category{
		ID:          int(row.ID),
		Title:       row.Title,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
	}, nil
}

// GetCategoryByID retrieves a category by ID
func (r *MetadataRepository) GetCategoryByID(ctx context.Context, id int) (*domain.Category, error) {
	row, err := r.queries.GetCategoryByID(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}

	return &domain.Category{
		ID:          int(row.ID),
		Title:       row.Title,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
	}, nil
}

// ListDocumentTypes retrieves all document types.
func (r *MetadataRepository) ListDocumentTypes(ctx context.Context) ([]domain.DocumentType, error) {
	rows, err := r.queries.ListDocumentTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list document types: %w", err)
	}

	types := make([]domain.DocumentType, len(rows))
	for i, row := range rows {
		types[i] = domain.DocumentType{
			ID:        int(row.ID),
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
		}
	}

	return types, nil
}

// GetDocumentTypeByID retrieves a document type by ID.
func (r *MetadataRepository) GetDocumentTypeByID(ctx context.Context, id int) (*domain.DocumentType, error) {
	row, err := r.queries.GetDocumentTypeByID(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("get document type: %w", err)
	}

	return &domain.DocumentType{
		ID:        int(row.ID),
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
	}, nil
}

// ListAuthorshipTypes retrieves all authorship types.
func (r *MetadataRepository) ListAuthorshipTypes(ctx context.Context) ([]domain.AuthorshipType, error) {
	rows, err := r.queries.ListAuthorshipTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list authorship types: %w", err)
	}

	types := make([]domain.AuthorshipType, len(rows))
	for i, row := range rows {
		types[i] = domain.AuthorshipType{
			ID:        int(row.ID),
			Title:     row.Title,
			CreatedAt: row.CreatedAt,
		}
	}

	return types, nil
}

// ListTags retrieves all tags
func (r *MetadataRepository) ListTags(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.queries.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	tags := make([]domain.Tag, len(rows))
	for i, row := range rows {
		tags[i] = domain.Tag{
			ID:        int(row.ID),
			Title:     row.Title,
			CreatedAt: row.CreatedAt,
		}
	}

	return tags, nil
}

// CreateTag creates a new tag
func (r *MetadataRepository) CreateTag(ctx context.Context, title string) (*domain.Tag, error) {
	now := time.Now()

	row, err := r.queries.CreateTag(ctx, queries.CreateTagParams{
		Title:     title,
		CreatedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}

	return &domain.Tag{
		ID:        int(row.ID),
		Title:     row.Title,
		CreatedAt: row.CreatedAt,
	}, nil
}

// GetTagByID retrieves a tag by ID
func (r *MetadataRepository) GetTagByID(ctx context.Context, id int) (*domain.Tag, error) {
	row, err := r.queries.GetTagByID(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("get tag: %w", err)
	}

	return &domain.Tag{
		ID:        int(row.ID),
		Title:     row.Title,
		CreatedAt: row.CreatedAt,
	}, nil
}
