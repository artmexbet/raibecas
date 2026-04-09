package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
)

// Storage defines the interface for document storage
type Storage interface {
	SaveDocument(ctx context.Context, documentID uuid.UUID, version int, content []byte) (string, error)
	GetDocument(ctx context.Context, path string) ([]byte, error)
	DeleteDocument(ctx context.Context, path string) error
	ListVersions(ctx context.Context, documentID uuid.UUID) ([]string, error)
	SaveCover(ctx context.Context, documentID uuid.UUID, data []byte, contentType string) (string, error)
	GetCoverPresignedURL(ctx context.Context, path string) (string, error)
	DeleteCover(ctx context.Context, path string) error
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	PublishDocumentCreated(ctx context.Context, event domain.DocumentCreatedEvent) error
	PublishDocumentUpdated(ctx context.Context, event domain.DocumentUpdatedEvent) error
	PublishDocumentDeleted(ctx context.Context, event domain.DocumentDeletedEvent) error
}

// DocumentRepository defines the interface for document data access
type DocumentRepository interface {
	Create(ctx context.Context, doc *domain.Document) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Document, error)
	List(ctx context.Context, params domain.ListDocumentsParams) ([]domain.Document, error)
	Count(ctx context.Context, params domain.ListDocumentsParams) (int, error)
	Update(ctx context.Context, doc *domain.Document) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateIndexedStatus(ctx context.Context, id uuid.UUID, indexed bool) error
	AddDocumentAuthor(ctx context.Context, documentID, authorID uuid.UUID, typeID int) error
	ClearDocumentAuthors(ctx context.Context, documentID uuid.UUID) error
}

// BookmarkRepository defines the interface for bookmark persistence.
type BookmarkRepository interface {
	Create(ctx context.Context, bookmark *domain.Bookmark) error
	GetByIDForUser(ctx context.Context, userID, bookmarkID uuid.UUID) (*domain.Bookmark, error)
	GetPublicationByUserAndDocument(ctx context.Context, userID, documentID uuid.UUID) (*domain.Bookmark, error)
	ListByUser(ctx context.Context, params domain.ListBookmarksParams) ([]domain.Bookmark, error)
	CountByUser(ctx context.Context, params domain.ListBookmarksParams) (int, error)
	Delete(ctx context.Context, userID, bookmarkID uuid.UUID) error
}

// VersionRepository defines the interface for version data access
type VersionRepository interface {
	Create(ctx context.Context, version *domain.DocumentVersion) error
	ListByDocumentID(ctx context.Context, documentID uuid.UUID) ([]domain.DocumentVersion, error)
}

// TagRepository defines the interface for tag operations
type TagRepository interface {
	AddToDocument(ctx context.Context, documentID uuid.UUID, tagID int) error
	ClearDocument(ctx context.Context, documentID uuid.UUID) error
}

// MetadataRepository defines the interface for metadata operations
type MetadataRepository interface {
	// Authors
	ListAuthors(ctx context.Context) ([]domain.Author, error)
	CreateAuthor(ctx context.Context, name string) (*domain.Author, error)
	GetAuthorByID(ctx context.Context, id uuid.UUID) (*domain.Author, error)

	// Categories
	ListCategories(ctx context.Context) ([]domain.Category, error)
	CreateCategory(ctx context.Context, title string) (*domain.Category, error)
	GetCategoryByID(ctx context.Context, id int) (*domain.Category, error)

	// Document types
	ListDocumentTypes(ctx context.Context) ([]domain.DocumentType, error)
	GetDocumentTypeByID(ctx context.Context, id int) (*domain.DocumentType, error)

	// Authorship types
	ListAuthorshipTypes(ctx context.Context) ([]domain.AuthorshipType, error)

	// Tags
	ListTags(ctx context.Context) ([]domain.Tag, error)
	CreateTag(ctx context.Context, title string) (*domain.Tag, error)
	GetTagByID(ctx context.Context, id int) (*domain.Tag, error)
}
