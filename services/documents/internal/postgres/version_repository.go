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

// VersionRepository implements repository.VersionRepository using PostgreSQL
type VersionRepository struct {
	queries *queries.Queries
}

// NewVersionRepository creates a new PostgreSQL version repository
func NewVersionRepository(queries *queries.Queries) *VersionRepository {
	return &VersionRepository{queries: queries}
}

// Create creates a new document version
func (r *VersionRepository) Create(ctx context.Context, version *domain.DocumentVersion) error {
	created, err := r.queries.CreateDocumentVersion(ctx, queries.CreateDocumentVersionParams{
		DocumentID:  version.DocumentID,
		Version:     int32(version.Version),
		ContentPath: version.ContentPath,
		Changes:     version.Changes,
		CreatedBy:   version.CreatedBy,
	})
	if err != nil {
		return fmt.Errorf("create document version: %w", err)
	}

	version.ID = created.ID
	version.CreatedAt = created.CreatedAt

	return nil
}

// ListByDocumentID retrieves all versions for a document
func (r *VersionRepository) ListByDocumentID(ctx context.Context, documentID uuid.UUID) ([]domain.DocumentVersion, error) {
	versions, err := r.queries.ListDocumentVersions(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("list document versions: %w", err)
	}

	result := make([]domain.DocumentVersion, len(versions))
	for i, v := range versions {
		result[i] = r.toDomain(&v)
	}

	return result, nil
}

// GetByDocumentAndVersion retrieves a specific version of a document
func (r *VersionRepository) GetByDocumentAndVersion(ctx context.Context, documentID uuid.UUID, version int) (*domain.DocumentVersion, error) {
	v, err := r.queries.GetDocumentVersion(ctx, struct {
		DocumentID uuid.UUID
		Version    int32
	}{
		DocumentID: documentID,
		Version:    int32(version),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get document version: %w", err)
	}

	result := r.toDomain(&v)
	return &result, nil
}

// toDomain converts queries.DocumentVersion to domain.DocumentVersion
func (r *VersionRepository) toDomain(v *queries.DocumentVersion) domain.DocumentVersion {
	return domain.DocumentVersion{
		ID:          v.ID,
		DocumentID:  v.DocumentID,
		Version:     int(v.Version),
		ContentPath: v.ContentPath,
		Changes:     v.Changes,
		CreatedBy:   v.CreatedBy,
		CreatedAt:   v.CreatedAt,
	}
}
