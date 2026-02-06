package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
)

// DocumentService handles business logic for documents
type DocumentService struct {
	docRepo      DocumentRepository
	versionRepo  VersionRepository
	tagRepo      TagRepository
	metadataRepo MetadataRepository
	storage      Storage
	publisher    EventPublisher
	logger       *slog.Logger
}

// NewDocumentService creates a new document service
func NewDocumentService(
	docRepo DocumentRepository,
	versionRepo VersionRepository,
	tagRepo TagRepository,
	metadataRepo MetadataRepository,
	storage Storage,
	publisher EventPublisher,
	logger *slog.Logger,
) *DocumentService {
	return &DocumentService{
		docRepo:      docRepo,
		versionRepo:  versionRepo,
		tagRepo:      tagRepo,
		metadataRepo: metadataRepo,
		storage:      storage,
		publisher:    publisher,
		logger:       logger,
	}
}

// CreateDocument creates a new document
func (s *DocumentService) CreateDocument(ctx context.Context, req domain.CreateDocumentRequest) (*domain.Document, error) {
	if req.Title == "" || req.Content == "" {
		return nil, fmt.Errorf("%w: title and content are required", ErrInvalidInput)
	}

	documentID := uuid.New()
	contentPath, err := s.storage.SaveDocument(ctx, documentID, 1, []byte(req.Content))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to save document to storage", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	doc := &domain.Document{
		ID:              documentID,
		Title:           req.Title,
		Description:     req.Description,
		AuthorID:        req.AuthorID,
		CategoryID:      req.CategoryID,
		PublicationDate: req.PublicationDate,
		ContentPath:     contentPath,
		CurrentVersion:  1,
		Indexed:         false,
	}

	if err := s.docRepo.Create(ctx, doc); err != nil {
		s.logger.ErrorContext(ctx, "failed to create document", "error", err)
		_ = s.storage.DeleteDocument(ctx, contentPath)
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	version := &domain.DocumentVersion{
		DocumentID:  doc.ID,
		Version:     1,
		ContentPath: contentPath,
		CreatedBy:   req.CreatedBy,
	}
	if err := s.versionRepo.Create(ctx, version); err != nil {
		s.logger.ErrorContext(ctx, "failed to create version", "error", err)
	}

	for _, tagID := range req.TagIDs {
		if err := s.tagRepo.AddToDocument(ctx, doc.ID, tagID); err != nil {
			s.logger.WarnContext(ctx, "failed to add tag", "tag_id", tagID, "error", err)
		}
	}

	// Publish event asynchronously with detached context
	go func() {
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = s.publisher.PublishDocumentCreated(publishCtx, domain.DocumentCreatedEvent{
			DocumentID:      doc.ID,
			Title:           doc.Title,
			AuthorID:        doc.AuthorID,
			CategoryID:      doc.CategoryID,
			PublicationDate: doc.PublicationDate,
			ContentPath:     doc.ContentPath,
			Version:         doc.CurrentVersion,
			Timestamp:       time.Now(),
		})
	}()

	return doc, nil
}

// GetDocument retrieves a document by ID
func (s *DocumentService) GetDocument(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return doc, nil
}

// GetDocumentContent retrieves document content
func (s *DocumentService) GetDocumentContent(ctx context.Context, id uuid.UUID) ([]byte, error) {
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}

	content, err := s.storage.GetDocument(ctx, doc.ContentPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}
	return content, nil
}

// ListDocuments retrieves documents with filters
func (s *DocumentService) ListDocuments(ctx context.Context, params domain.ListDocumentsParams) ([]domain.Document, int, error) {
	docs, err := s.docRepo.List(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list documents: %w", err)
	}

	total, err := s.docRepo.Count(ctx, params)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to count documents", "error", err)
		total = len(docs)
	}

	return docs, total, nil
}

// UpdateDocument updates a document
func (s *DocumentService) UpdateDocument(ctx context.Context, id uuid.UUID, req domain.UpdateDocumentRequest) (*domain.Document, error) {
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}

	oldVersion := doc.CurrentVersion
	newVersion := oldVersion + 1

	if req.Content != nil && *req.Content != "" {
		contentPath, err := s.storage.SaveDocument(ctx, id, newVersion, []byte(*req.Content))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
		}
		doc.ContentPath = contentPath
		doc.CurrentVersion = newVersion

		version := &domain.DocumentVersion{
			DocumentID:  id,
			Version:     newVersion,
			ContentPath: contentPath,
			Changes:     req.Changes,
			CreatedBy:   req.UpdatedBy,
		}
		if err := s.versionRepo.Create(ctx, version); err != nil {
			s.logger.ErrorContext(ctx, "failed to create version", "error", err)
		}
	}

	if req.Title != nil {
		doc.Title = *req.Title
	}
	if req.Description != nil {
		doc.Description = req.Description
	}
	if req.AuthorID != nil {
		doc.AuthorID = *req.AuthorID
	}
	if req.CategoryID != nil {
		doc.CategoryID = *req.CategoryID
	}
	if req.PublicationDate != nil {
		doc.PublicationDate = *req.PublicationDate
	}

	if err := s.docRepo.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}

	if req.TagIDs != nil {
		if err := s.tagRepo.ClearDocument(ctx, id); err != nil {
			s.logger.WarnContext(ctx, "failed to clear tags", "error", err)
		}
		for _, tagID := range req.TagIDs {
			if err := s.tagRepo.AddToDocument(ctx, id, tagID); err != nil {
				s.logger.WarnContext(ctx, "failed to add tag", "tag_id", tagID, "error", err)
			}
		}
	}

	if req.Content != nil && *req.Content != "" {
		go func() {
			publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_ = s.publisher.PublishDocumentUpdated(publishCtx, domain.DocumentUpdatedEvent{
				DocumentID:  id,
				OldVersion:  oldVersion,
				NewVersion:  newVersion,
				ContentPath: doc.ContentPath,
				Changes:     req.Changes,
				Timestamp:   time.Now(),
			})
		}()
	}

	return doc, nil
}

// DeleteDocument deletes a document
func (s *DocumentService) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get document: %w", err)
	}

	if err := s.docRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete document: %w", err)
	}

	versions, err := s.storage.ListVersions(ctx, id)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to list versions for cleanup", "error", err)
	} else {
		for _, path := range versions {
			_ = s.storage.DeleteDocument(ctx, path)
		}
	}

	go func() {
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = s.publisher.PublishDocumentDeleted(publishCtx, domain.DocumentDeletedEvent{
			DocumentID: id,
			Timestamp:  time.Now(),
		})
	}()

	s.logger.InfoContext(ctx, "deleted document", "document_id", id, "title", doc.Title)
	return nil
}

// MarkDocumentIndexed marks document as indexed
func (s *DocumentService) MarkDocumentIndexed(ctx context.Context, id uuid.UUID, indexed bool) error {
	return s.docRepo.UpdateIndexedStatus(ctx, id, indexed)
}

// ListDocumentVersions retrieves document versions
func (s *DocumentService) ListDocumentVersions(ctx context.Context, id uuid.UUID) ([]domain.DocumentVersion, error) {
	return s.versionRepo.ListByDocumentID(ctx, id)
}

// Metadata methods

// ListAuthors retrieves all authors
func (s *DocumentService) ListAuthors(ctx context.Context) ([]domain.Author, error) {
	return s.metadataRepo.ListAuthors(ctx)
}

// CreateAuthor creates a new author
func (s *DocumentService) CreateAuthor(ctx context.Context, name string) (*domain.Author, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	return s.metadataRepo.CreateAuthor(ctx, name)
}

// ListCategories retrieves all categories
func (s *DocumentService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.metadataRepo.ListCategories(ctx)
}

// CreateCategory creates a new category
func (s *DocumentService) CreateCategory(ctx context.Context, title string) (*domain.Category, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	return s.metadataRepo.CreateCategory(ctx, title)
}

// ListTags retrieves all tags
func (s *DocumentService) ListTags(ctx context.Context) ([]domain.Tag, error) {
	return s.metadataRepo.ListTags(ctx)
}

// CreateTag creates a new tag
func (s *DocumentService) CreateTag(ctx context.Context, title string) (*domain.Tag, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	return s.metadataRepo.CreateTag(ctx, title)
}
