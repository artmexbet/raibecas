package server

import (
	"context"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// DocumentServiceConnector defines the interface for communicating with the document service
type DocumentServiceConnector interface {
	// ListDocuments retrieves a list of documents with filtering and pagination
	ListDocuments(ctx context.Context, query domain.ListDocumentsQuery) (*domain.ListDocumentsResponse, error)

	// CreateDocument creates a new document
	CreateDocument(ctx context.Context, req domain.CreateDocumentRequest) (*domain.CreateDocumentResponse, error)

	// GetDocument retrieves a single document by ID
	GetDocument(ctx context.Context, id uuid.UUID) (*domain.GetDocumentResponse, error)

	// UpdateDocument updates an existing document
	UpdateDocument(ctx context.Context, id uuid.UUID, req domain.UpdateDocumentRequest) (*domain.UpdateDocumentResponse, error)

	// DeleteDocument deletes a document by ID
	DeleteDocument(ctx context.Context, id uuid.UUID) error
}
