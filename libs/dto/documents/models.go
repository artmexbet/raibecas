package documents

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all models.go

// Document DTOs - Shared data structures for document service communication

// ListDocumentsQuery represents query parameters for listing documents
//
//easyjson:json
type ListDocumentsQuery struct {
	Page       int       `json:"page,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
	AuthorID   uuid.UUID `json:"author_id,omitempty"`
	CategoryID int       `json:"category_id,omitempty"`
	TagID      int       `json:"tag_id,omitempty"`
	Search     string    `json:"search,omitempty"`
}

// ListDocumentsResponse represents the response for listing documents
//
//easyjson:json
type ListDocumentsResponse struct {
	Documents  []Document `json:"documents"`
	Total      int        `json:"total"`
	Page       int        `json:"page,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	TotalPages int        `json:"totalPages,omitempty"`
}

// CreateDocumentRequest represents a document creation request
//
//easyjson:json
type CreateDocumentRequest struct {
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	AuthorID        uuid.UUID  `json:"author_id"`
	CategoryID      int        `json:"category_id"`
	PublicationDate time.Time  `json:"publication_date"`
	Content         string     `json:"content,omitempty"`
	TagIDs          []int      `json:"tag_ids,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
}

// CreateDocumentResponse represents a document creation response
//
//easyjson:json
type CreateDocumentResponse struct {
	Document Document `json:"document"`
}

// GetDocumentRequest represents a document retrieval request
//
//easyjson:json
type GetDocumentRequest struct {
	ID uuid.UUID `json:"id"`
}

// GetDocumentResponse represents a single document response
//
//easyjson:json
type GetDocumentResponse struct {
	Document Document `json:"document"`
}

// GetDocumentContentRequest represents a document content retrieval request
//
//easyjson:json
type GetDocumentContentRequest struct {
	ID uuid.UUID `json:"id"`
}

// GetDocumentContentResponse represents a document content response
//
//easyjson:json
type GetDocumentContentResponse struct {
	Content string `json:"content"`
}

// UpdateDocumentRequest represents a document update request
//
//easyjson:json
type UpdateDocumentRequest struct {
	ID              uuid.UUID  `json:"id,omitempty"`
	Title           *string    `json:"title,omitempty"`
	Description     *string    `json:"description,omitempty"`
	AuthorID        *uuid.UUID `json:"author_id,omitempty"`
	CategoryID      *int       `json:"category_id,omitempty"`
	PublicationDate *time.Time `json:"publication_date,omitempty"`
	Content         *string    `json:"content,omitempty"`
	TagIDs          []int      `json:"tag_ids,omitempty"`
	Changes         *string    `json:"changes,omitempty"`
	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty"`
}

// UpdateDocumentResponse represents a document update response
//
//easyjson:json
type UpdateDocumentResponse struct {
	Document Document `json:"document"`
}

// DeleteDocumentRequest represents a document deletion request
//
//easyjson:json
type DeleteDocumentRequest struct {
	ID uuid.UUID `json:"id"`
}

// DeleteDocumentResponse represents a document deletion response
//
//easyjson:json
type DeleteDocumentResponse struct {
	Success bool `json:"success"`
}

// ListDocumentVersionsRequest represents a request to list document versions
//
//easyjson:json
type ListDocumentVersionsRequest struct {
	ID uuid.UUID `json:"id"`
}

// ListDocumentVersionsResponse represents a response with document versions
//
//easyjson:json
type ListDocumentVersionsResponse struct {
	Versions []DocumentVersion `json:"versions"`
}

// Document represents a scientific document
//
//easyjson:json
type Document struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	Description     *string   `json:"description,omitempty"`
	AuthorID        uuid.UUID `json:"author_id"`
	CategoryID      int       `json:"category_id"`
	PublicationDate time.Time `json:"publication_date"`
	ContentPath     string    `json:"content_path"`
	CurrentVersion  int       `json:"current_version"`
	Indexed         bool      `json:"indexed"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Author          *Author   `json:"author,omitempty"`
	Category        *Category `json:"category,omitempty"`
	Tags            []Tag     `json:"tags,omitempty"`
}

// Author represents a scientific work author
//
//easyjson:json
type Author struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Bio       *string   `json:"bio,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Category represents a document category
//
//easyjson:json
type Category struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Tag represents a document tag
//
//easyjson:json
type Tag struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// DocumentVersion represents a version of a document
//
//easyjson:json
type DocumentVersion struct {
	ID          uuid.UUID  `json:"id"`
	DocumentID  uuid.UUID  `json:"document_id"`
	Version     int        `json:"version"`
	ContentPath string     `json:"content_path"`
	Changes     *string    `json:"changes,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
