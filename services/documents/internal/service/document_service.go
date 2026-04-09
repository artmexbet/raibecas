package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
)

// DocumentService handles business logic for documents.
type DocumentService struct {
	docRepo      DocumentRepository
	bookmarkRepo BookmarkRepository
	versionRepo  VersionRepository
	tagRepo      TagRepository
	metadataRepo MetadataRepository
	storage      Storage
	publisher    EventPublisher
	logger       *slog.Logger
}

// NewDocumentService creates a new document service.
func NewDocumentService(
	docRepo DocumentRepository,
	bookmarkRepo BookmarkRepository,
	versionRepo VersionRepository,
	tagRepo TagRepository,
	metadataRepo MetadataRepository,
	storage Storage,
	publisher EventPublisher,
	logger *slog.Logger,
) *DocumentService {
	return &DocumentService{
		docRepo:      docRepo,
		bookmarkRepo: bookmarkRepo,
		versionRepo:  versionRepo,
		tagRepo:      tagRepo,
		metadataRepo: metadataRepo,
		storage:      storage,
		publisher:    publisher,
		logger:       logger,
	}
}

// CreateDocument creates a new document.
func (s *DocumentService) CreateDocument(ctx context.Context, req domain.CreateDocumentRequest) (*domain.Document, error) {
	if req.Title == "" || req.Content == "" {
		return nil, fmt.Errorf("%w: title and content are required", ErrInvalidInput)
	}
	if req.DocumentTypeID <= 0 {
		return nil, fmt.Errorf("%w: document type is required", ErrInvalidInput)
	}

	participants, err := s.normalizeParticipants(req.Participants)
	if err != nil {
		return nil, err
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
		CategoryID:      req.CategoryID,
		DocumentTypeID:  req.DocumentTypeID,
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

	if err := s.replaceDocumentParticipants(ctx, doc.ID, participants); err != nil {
		s.logger.ErrorContext(ctx, "failed to save document participants", "error", err)
		return nil, fmt.Errorf("save document participants: %w", err)
	}

	if err := s.replaceDocumentTags(ctx, doc.ID, req.TagIDs); err != nil {
		s.logger.WarnContext(ctx, "failed to save document tags", "error", err)
	}

	storedDoc, err := s.docRepo.GetByID(ctx, doc.ID)
	if err != nil {
		return nil, fmt.Errorf("load created document: %w", err)
	}

	go s.publishDocumentCreatedEvent(*storedDoc)
	return storedDoc, nil
}

// GetDocument retrieves a document by ID.
func (s *DocumentService) GetDocument(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	s.enrichCoverURL(ctx, doc)
	return doc, nil
}

// GetDocumentContent retrieves document content.
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

// ListDocuments retrieves documents with filters.
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

	for i := range docs {
		s.enrichCoverURL(ctx, &docs[i])
	}

	return docs, total, nil
}

// UpdateDocument updates a document.
func (s *DocumentService) UpdateDocument(ctx context.Context, id uuid.UUID, req domain.UpdateDocumentRequest) (*domain.Document, error) {
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}

	oldVersion := doc.CurrentVersion
	newVersion := oldVersion

	if req.Content != nil && *req.Content != "" {
		newVersion = oldVersion + 1
		contentPath, saveErr := s.storage.SaveDocument(ctx, id, newVersion, []byte(*req.Content))
		if saveErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrStorageFailure, saveErr)
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
	if req.CategoryID != nil {
		doc.CategoryID = req.CategoryID
	}
	if req.DocumentTypeID != nil {
		if *req.DocumentTypeID <= 0 {
			return nil, fmt.Errorf("%w: invalid document type", ErrInvalidInput)
		}
		doc.DocumentTypeID = *req.DocumentTypeID
	}
	if req.PublicationDate != nil {
		doc.PublicationDate = *req.PublicationDate
	}
	if req.CoverPath != nil {
		doc.CoverPath = req.CoverPath
	}

	if len(req.Participants) > 0 {
		participants, normErr := s.normalizeParticipants(req.Participants)
		if normErr != nil {
			return nil, normErr
		}
		if err := s.replaceDocumentParticipants(ctx, id, participants); err != nil {
			return nil, fmt.Errorf("replace document participants: %w", err)
		}
	}

	if err := s.docRepo.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}

	if req.TagIDs != nil {
		if err := s.replaceDocumentTags(ctx, id, req.TagIDs); err != nil {
			s.logger.WarnContext(ctx, "failed to replace tags", "error", err)
		}
	}

	storedDoc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load updated document: %w", err)
	}

	go s.publishDocumentUpdatedEvent(*storedDoc, oldVersion, newVersion, req.Changes)
	return storedDoc, nil
}

// DeleteDocument deletes a document.
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

	if doc.CoverPath != nil && *doc.CoverPath != "" {
		if err := s.storage.DeleteCover(ctx, *doc.CoverPath); err != nil {
			s.logger.WarnContext(ctx, "failed to delete cover", "error", err)
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

// MarkDocumentIndexed marks document as indexed.
func (s *DocumentService) MarkDocumentIndexed(ctx context.Context, id uuid.UUID, indexed bool) error {
	return s.docRepo.UpdateIndexedStatus(ctx, id, indexed)
}

// ListDocumentVersions retrieves document versions.
func (s *DocumentService) ListDocumentVersions(ctx context.Context, id uuid.UUID) ([]domain.DocumentVersion, error) {
	return s.versionRepo.ListByDocumentID(ctx, id)
}

// ListAuthors retrieves all authors.
func (s *DocumentService) ListAuthors(ctx context.Context) ([]domain.Author, error) {
	return s.metadataRepo.ListAuthors(ctx)
}

// CreateAuthor creates a new author.
func (s *DocumentService) CreateAuthor(ctx context.Context, name string) (*domain.Author, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	return s.metadataRepo.CreateAuthor(ctx, name)
}

// ListCategories retrieves all categories.
func (s *DocumentService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.metadataRepo.ListCategories(ctx)
}

// CreateCategory creates a new category.
func (s *DocumentService) CreateCategory(ctx context.Context, title string) (*domain.Category, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	return s.metadataRepo.CreateCategory(ctx, title)
}

// ListDocumentTypes retrieves all document types.
func (s *DocumentService) ListDocumentTypes(ctx context.Context) ([]domain.DocumentType, error) {
	return s.metadataRepo.ListDocumentTypes(ctx)
}

// ListAuthorshipTypes retrieves all authorship types.
func (s *DocumentService) ListAuthorshipTypes(ctx context.Context) ([]domain.AuthorshipType, error) {
	return s.metadataRepo.ListAuthorshipTypes(ctx)
}

// ListTags retrieves all tags.
func (s *DocumentService) ListTags(ctx context.Context) ([]domain.Tag, error) {
	return s.metadataRepo.ListTags(ctx)
}

// CreateTag creates a new tag.
func (s *DocumentService) CreateTag(ctx context.Context, title string) (*domain.Tag, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	return s.metadataRepo.CreateTag(ctx, title)
}

// UploadCover saves a cover image for a document and returns the presigned URL.
func (s *DocumentService) UploadCover(ctx context.Context, id uuid.UUID, data []byte, contentType string) (string, error) {
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get document: %w", err)
	}

	if doc.CoverPath != nil && *doc.CoverPath != "" {
		if err := s.storage.DeleteCover(ctx, *doc.CoverPath); err != nil {
			s.logger.WarnContext(ctx, "failed to delete old cover", "error", err)
		}
	}

	coverPath, err := s.storage.SaveCover(ctx, id, data, contentType)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	doc.CoverPath = &coverPath
	if err := s.docRepo.Update(ctx, doc); err != nil {
		return "", fmt.Errorf("update document cover path: %w", err)
	}

	presignedURL, err := s.storage.GetCoverPresignedURL(ctx, coverPath)
	if err != nil {
		return "", fmt.Errorf("get cover presigned url: %w", err)
	}

	return presignedURL, nil
}

func (s *DocumentService) replaceDocumentTags(ctx context.Context, documentID uuid.UUID, tagIDs []int) error {
	if err := s.tagRepo.ClearDocument(ctx, documentID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if err := s.tagRepo.AddToDocument(ctx, documentID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func (s *DocumentService) replaceDocumentParticipants(ctx context.Context, documentID uuid.UUID, participants []domain.DocumentParticipantRef) error {
	if err := s.docRepo.ClearDocumentAuthors(ctx, documentID); err != nil {
		return err
	}
	for _, participant := range participants {
		if err := s.docRepo.AddDocumentAuthor(ctx, documentID, participant.AuthorID, participant.TypeID); err != nil {
			return err
		}
	}
	return nil
}

func (s *DocumentService) normalizeParticipants(participants []domain.DocumentParticipantRef) ([]domain.DocumentParticipantRef, error) {
	if len(participants) == 0 {
		return nil, fmt.Errorf("%w: at least one participant is required", ErrInvalidInput)
	}

	seen := make(map[string]struct{}, len(participants))
	result := make([]domain.DocumentParticipantRef, 0, len(participants))
	for _, participant := range participants {
		if participant.AuthorID == uuid.Nil || participant.TypeID <= 0 {
			return nil, fmt.Errorf("%w: invalid participant", ErrInvalidInput)
		}
		key := participant.AuthorID.String() + ":" + fmt.Sprintf("%d", participant.TypeID)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, participant)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("%w: at least one participant is required", ErrInvalidInput)
	}
	return result, nil
}

func (s *DocumentService) publishDocumentCreatedEvent(doc domain.Document) {
	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.publisher.PublishDocumentCreated(publishCtx, domain.DocumentCreatedEvent{
		DocumentID:      doc.ID,
		Title:           doc.Title,
		Description:     doc.Description,
		CategoryID:      doc.CategoryID,
		DocumentTypeID:  doc.DocumentTypeID,
		DocumentType:    documentTypeName(doc.DocumentType),
		PublicationDate: doc.PublicationDate,
		ContentPath:     doc.ContentPath,
		Version:         doc.CurrentVersion,
		Participants:    toEventParticipants(doc.Participants),
		Tags:            toEventTags(doc.Tags),
		Timestamp:       time.Now(),
	})
}

func (s *DocumentService) publishDocumentUpdatedEvent(doc domain.Document, oldVersion, newVersion int, changes *string) {
	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.publisher.PublishDocumentUpdated(publishCtx, domain.DocumentUpdatedEvent{
		DocumentID:      doc.ID,
		Title:           doc.Title,
		Description:     doc.Description,
		CategoryID:      doc.CategoryID,
		DocumentTypeID:  doc.DocumentTypeID,
		DocumentType:    documentTypeName(doc.DocumentType),
		PublicationDate: doc.PublicationDate,
		OldVersion:      oldVersion,
		NewVersion:      newVersion,
		ContentPath:     doc.ContentPath,
		Changes:         changes,
		Participants:    toEventParticipants(doc.Participants),
		Tags:            toEventTags(doc.Tags),
		Timestamp:       time.Now(),
	})
}

func documentTypeName(documentType *domain.DocumentType) string {
	if documentType == nil {
		return ""
	}
	return documentType.Name
}

func toEventParticipants(participants []domain.DocumentParticipant) []domain.DocumentEventParticipant {
	result := make([]domain.DocumentEventParticipant, len(participants))
	for i, participant := range participants {
		result[i] = domain.DocumentEventParticipant{
			AuthorID:  participant.Author.ID,
			Name:      participant.Author.Name,
			TypeID:    participant.AuthorshipType.ID,
			TypeTitle: participant.AuthorshipType.Title,
		}
	}
	return result
}

func toEventTags(tags []domain.Tag) []domain.DocumentEventTag {
	result := make([]domain.DocumentEventTag, len(tags))
	for i, tag := range tags {
		result[i] = domain.DocumentEventTag{ID: tag.ID, Title: tag.Title}
	}
	return result
}

// enrichCoverURL adds presigned cover URL to a document if it has a cover.
func (s *DocumentService) enrichCoverURL(ctx context.Context, doc *domain.Document) {
	if doc.CoverPath == nil || *doc.CoverPath == "" {
		return
	}
	url, err := s.storage.GetCoverPresignedURL(ctx, *doc.CoverPath)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get cover presigned url", "document_id", doc.ID, "error", err)
		return
	}
	doc.CoverURL = &url
}
