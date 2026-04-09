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

// DocumentRepository implements repository.DocumentRepository using PostgreSQL.
type DocumentRepository struct {
	queries *queries.Queries
}

// NewDocumentRepository creates a new PostgreSQL document repository.
func NewDocumentRepository(queries *queries.Queries) *DocumentRepository {
	return &DocumentRepository{queries: queries}
}

// Create creates a new document.
func (r *DocumentRepository) Create(ctx context.Context, doc *domain.Document) error {
	created, err := r.queries.CreateDocument(ctx, queries.CreateDocumentParams{
		Title:           doc.Title,
		Description:     doc.Description,
		CategoryID:      intPtrToInt32Ptr(doc.CategoryID),
		PublicationDate: timeToDate(doc.PublicationDate),
		ContentPath:     doc.ContentPath,
		CurrentVersion:  int32(doc.CurrentVersion),
		DocumentTypeID:  int32(doc.DocumentTypeID),
	})
	if err != nil {
		return fmt.Errorf("create document: %w", err)
	}

	doc.ID = created.ID
	doc.CreatedAt = created.CreatedAt
	doc.UpdatedAt = created.UpdatedAt
	return nil
}

// GetByID retrieves a document by ID.
func (r *DocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	row, err := r.queries.GetDocumentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get document by id: %w", err)
	}

	doc, err := r.toDomainDocumentFromRow(ctx, row)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// List retrieves documents with filters.
func (r *DocumentRepository) List(ctx context.Context, params domain.ListDocumentsParams) ([]domain.Document, error) {
	rows, err := r.queries.ListDocuments(ctx, queries.ListDocumentsParams{
		Limit:          int32(params.Limit),
		Offset:         int32(params.Offset),
		AuthorID:       params.AuthorID,
		CategoryID:     params.CategoryID,
		DocumentTypeID: params.DocumentTypeID,
		TagID:          params.TagID,
		Search:         convertStringToPtr(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	result := make([]domain.Document, len(rows))
	for i, row := range rows {
		doc, err := r.toDomainDocumentFromListRow(ctx, row)
		if err != nil {
			return nil, err
		}
		result[i] = *doc
	}

	return result, nil
}

// Count counts documents.
func (r *DocumentRepository) Count(ctx context.Context, params domain.ListDocumentsParams) (int, error) {
	count, err := r.queries.CountDocuments(ctx, queries.CountDocumentsParams{
		AuthorID:       params.AuthorID,
		CategoryID:     params.CategoryID,
		DocumentTypeID: params.DocumentTypeID,
		TagID:          params.TagID,
		Search:         convertStringToPtr(params.Search),
	})
	if err != nil {
		return 0, fmt.Errorf("count documents: %w", err)
	}
	return int(count), nil
}

// Update updates a document.
func (r *DocumentRepository) Update(ctx context.Context, doc *domain.Document) error {
	currentVersion := int32(doc.CurrentVersion)
	documentTypeID := int32(doc.DocumentTypeID)

	updated, err := r.queries.UpdateDocument(ctx, queries.UpdateDocumentParams{
		ID:              doc.ID,
		Title:           &doc.Title,
		Description:     doc.Description,
		CategoryID:      intPtrToInt32Ptr(doc.CategoryID),
		PublicationDate: timeToDate(doc.PublicationDate),
		ContentPath:     &doc.ContentPath,
		CurrentVersion:  &currentVersion,
		CoverPath:       doc.CoverPath,
		DocumentTypeID:  &documentTypeID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("update document: %w", err)
	}
	_ = updated
	doc.UpdatedAt = updated.UpdatedAt
	return nil
}

// Delete deletes a document.
func (r *DocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteDocument(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete document: %w", err)
	}
	return nil
}

// UpdateIndexedStatus updates indexed status.
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

// AddDocumentAuthor stores a document participant relation.
func (r *DocumentRepository) AddDocumentAuthor(ctx context.Context, documentID, authorID uuid.UUID, typeID int) error {
	if err := r.queries.AddDocumentAuthor(ctx, queries.AddDocumentAuthorParams{
		DocumentID: documentID,
		AuthorID:   authorID,
		TypeID:     int32(typeID),
	}); err != nil {
		return fmt.Errorf("add document author: %w", err)
	}
	return nil
}

// ClearDocumentAuthors removes all participant relations for a document.
func (r *DocumentRepository) ClearDocumentAuthors(ctx context.Context, documentID uuid.UUID) error {
	if err := r.queries.ClearDocumentAuthors(ctx, documentID); err != nil {
		return fmt.Errorf("clear document authors: %w", err)
	}
	return nil
}

func (r *DocumentRepository) toDomainDocumentFromRow(ctx context.Context, row queries.GetDocumentByIDRow) (*domain.Document, error) {
	base := queries.Document{
		ID:              row.ID,
		Title:           row.Title,
		Description:     row.Description,
		CategoryID:      row.CategoryID,
		PublicationDate: row.PublicationDate,
		ContentPath:     row.ContentPath,
		CurrentVersion:  row.CurrentVersion,
		Indexed:         row.Indexed,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		CoverPath:       row.CoverPath,
		DocumentTypeID:  row.DocumentTypeID,
	}
	return r.toDomainDocument(ctx, row.ID, base, row.Category, row.DocumentType)
}

func (r *DocumentRepository) toDomainDocumentFromListRow(ctx context.Context, row queries.ListDocumentsRow) (*domain.Document, error) {
	base := queries.Document{
		ID:              row.ID,
		Title:           row.Title,
		Description:     row.Description,
		CategoryID:      row.CategoryID,
		PublicationDate: row.PublicationDate,
		ContentPath:     row.ContentPath,
		CurrentVersion:  row.CurrentVersion,
		Indexed:         row.Indexed,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		CoverPath:       row.CoverPath,
		DocumentTypeID:  row.DocumentTypeID,
	}
	return r.toDomainDocument(ctx, row.ID, base, row.Category, row.DocumentType)
}

func (r *DocumentRepository) toDomainDocument(
	ctx context.Context,
	documentID uuid.UUID,
	row queries.Document,
	category queries.Category,
	documentType queries.DocumentType,
) (*domain.Document, error) {
	doc := &domain.Document{
		ID:              row.ID,
		Title:           row.Title,
		Description:     row.Description,
		CategoryID:      int32PtrToIntPtr(row.CategoryID),
		DocumentTypeID:  int(documentType.ID),
		PublicationDate: row.PublicationDate.Time,
		ContentPath:     row.ContentPath,
		CoverPath:       row.CoverPath,
		CurrentVersion:  int(row.CurrentVersion),
		Indexed:         row.Indexed,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	if category.ID != 0 {
		doc.Category = &domain.Category{
			ID:          int(category.ID),
			Title:       category.Title,
			Description: category.Description,
			CreatedAt:   category.CreatedAt,
		}
	}

	if documentType.ID != 0 {
		doc.DocumentType = &domain.DocumentType{
			ID:        int(documentType.ID),
			Name:      documentType.Name,
			CreatedAt: documentType.CreatedAt,
		}
		doc.DocumentTypeID = int(documentType.ID)
	}

	participants, err := r.queries.GetDocumentAuthors(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("get document participants: %w", err)
	}
	if len(participants) > 0 {
		doc.Participants = make([]domain.DocumentParticipant, len(participants))
		for i, participant := range participants {
			doc.Participants[i] = domain.DocumentParticipant{
				Author: domain.Author{
					ID:        participant.AuthorID,
					Name:      participant.AuthorName,
					Bio:       participant.AuthorBio,
					CreatedAt: participant.AuthorCreatedAt,
					UpdatedAt: participant.AuthorUpdatedAt,
				},
				AuthorshipType: domain.AuthorshipType{
					ID:        int(participant.AuthorshipTypeID),
					Title:     participant.AuthorshipTypeTitle,
					CreatedAt: participant.AuthorshipTypeCreatedAt,
				},
			}
		}

		primaryAuthor := doc.Participants[0].Author
		doc.Author = &primaryAuthor
	}

	tags, err := r.queries.GetDocumentTags(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("get document tags: %w", err)
	}
	if len(tags) > 0 {
		doc.Tags = make([]domain.Tag, len(tags))
		for i, tag := range tags {
			doc.Tags[i] = domain.Tag{
				ID:        int(tag.ID),
				Title:     tag.Title,
				CreatedAt: tag.CreatedAt,
			}
		}
	}

	return doc, nil
}

// Helper functions.
func timeToDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func convertStringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtrToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func int32PtrToIntPtr(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}
