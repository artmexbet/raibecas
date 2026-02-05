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
