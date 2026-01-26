package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all documents.go

// Document DTOs - Request/Response models for document management endpoints

// ListDocumentsQuery represents query parameters for listing documents
type ListDocumentsQuery struct {
	Page       int       `query:"page" validate:"omitempty,min=1"`
	Limit      int       `query:"limit" validate:"omitempty,min=1,max=100"`
	AuthorID   uuid.UUID `query:"authorId" validate:"omitempty,uuid"`
	CategoryID int       `query:"categoryId" validate:"omitempty,min=1"`
	TagID      int       `query:"tagId" validate:"omitempty,min=1"`
	Search     string    `query:"search" validate:"omitempty,min=1,max=255"`
}

// ListDocumentsResponse represents the response for listing documents
type ListDocumentsResponse struct {
	Documents  []Document `json:"documents"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"totalPages"`
}

// CreateDocumentRequest represents a document creation request
type CreateDocumentRequest struct {
	Title           string    `json:"title" validate:"required,min=1,max=255"`
	Description     *string   `json:"description" validate:"omitempty,max=1000"`
	AuthorID        uuid.UUID `json:"authorId" validate:"required,uuid"`
	CategoryID      int       `json:"categoryId" validate:"required,min=1"`
	PublicationDate time.Time `json:"publicationDate" validate:"required"`
	TagIDs          []int     `json:"tagIds" validate:"omitempty,dive,min=1"`
}

// CreateDocumentResponse represents a document creation response
type CreateDocumentResponse struct {
	Document Document `json:"document"`
}

// GetDocumentResponse represents a single document response
type GetDocumentResponse struct {
	Document Document `json:"document"`
}

// UpdateDocumentRequest represents a document update request
type UpdateDocumentRequest struct {
	Title           *string    `json:"title" validate:"omitempty,min=1,max=255"`
	Description     *string    `json:"description" validate:"omitempty,max=1000"`
	AuthorID        *uuid.UUID `json:"authorId" validate:"omitempty,uuid"`
	CategoryID      *int       `json:"categoryId" validate:"omitempty,min=1"`
	PublicationDate *time.Time `json:"publicationDate" validate:"omitempty"`
	TagIDs          []int      `json:"tagIds" validate:"omitempty,dive,min=1"`
}

// UpdateDocumentResponse represents a document update response
type UpdateDocumentResponse struct {
	Document Document `json:"document"`
}
