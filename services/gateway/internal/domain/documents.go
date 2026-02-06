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
	Content         string    `json:"content" validate:"omitempty"`
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
	ID              uuid.UUID  `json:"id" validate:"required,uuid"`
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

// Author metadata DTOs

// CreateAuthorRequest represents an author creation request
type CreateAuthorRequest struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
}

// CreateAuthorResponse represents an author creation response
type CreateAuthorResponse struct {
	Author Author `json:"author"`
}

// ListAuthorsResponse represents the response for listing authors
type ListAuthorsResponse struct {
	Authors []Author `json:"authors"`
}

// Category metadata DTOs

// CreateCategoryRequest represents a category creation request
type CreateCategoryRequest struct {
	Title string `json:"title" validate:"required,min=2,max=100"`
}

// CreateCategoryResponse represents a category creation response
type CreateCategoryResponse struct {
	Category Category `json:"category"`
}

// ListCategoriesResponse represents the response for listing categories
type ListCategoriesResponse struct {
	Categories []Category `json:"categories"`
}

// Tag metadata DTOs

// CreateTagRequest represents a tag creation request
type CreateTagRequest struct {
	Title string `json:"title" validate:"required,min=2,max=50"`
}

// CreateTagResponse represents a tag creation response
type CreateTagResponse struct {
	Tag Tag `json:"tag"`
}

// ListTagsResponse represents the response for listing tags
type ListTagsResponse struct {
	Tags []Tag `json:"tags"`
}
