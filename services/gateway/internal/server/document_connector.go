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
	CreateDocument(ctx context.Context, req domain.CreateDocumentRequest, userRole string) (*domain.CreateDocumentResponse, error)

	// GetDocument retrieves a single document by ID
	GetDocument(ctx context.Context, id uuid.UUID) (*domain.GetDocumentResponse, error)

	// UpdateDocument updates an existing document
	UpdateDocument(ctx context.Context, req domain.UpdateDocumentRequest, userRole string) (*domain.UpdateDocumentResponse, error)

	// DeleteDocument deletes a document by ID
	DeleteDocument(ctx context.Context, id uuid.UUID, userRole string) error

	// UploadCover uploads a cover image for a document
	UploadCover(ctx context.Context, id uuid.UUID, data []byte, contentType string, userRole string) (string, error)

	// Metadata methods

	// ListAuthors retrieves all authors
	ListAuthors(ctx context.Context) (*domain.ListAuthorsResponse, error)

	// CreateAuthor creates a new author
	CreateAuthor(ctx context.Context, req domain.CreateAuthorRequest, userRole string) (*domain.CreateAuthorResponse, error)

	// ListCategories retrieves all categories
	ListCategories(ctx context.Context) (*domain.ListCategoriesResponse, error)

	// CreateCategory creates a new category
	CreateCategory(ctx context.Context, req domain.CreateCategoryRequest, userRole string) (*domain.CreateCategoryResponse, error)

	// ListTags retrieves all tags
	ListTags(ctx context.Context) (*domain.ListTagsResponse, error)

	// CreateTag creates a new tag
	CreateTag(ctx context.Context, req domain.CreateTagRequest, userRole string) (*domain.CreateTagResponse, error)
}
