package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all dto.go

// IDRequest represents a simple ID request (for get/delete operations)
type IDRequest struct {
	ID string `json:"id"`
}

// UpdateDocumentPayload represents the payload for updating a document
type UpdateDocumentPayload struct {
	ID      string                `json:"id"`
	Updates UpdateDocumentRequest `json:"updates"`
}

// Document DTOs for API requests and responses

// CreateDocumentRequest represents the request to create a new document
type CreateDocumentRequest struct {
	Title           string    `json:"title" validate:"required,min=1,max=500"`
	Description     *string   `json:"description" validate:"omitempty,max=2000"`
	AuthorID        uuid.UUID `json:"authorId" validate:"required,uuid"`
	CategoryID      int       `json:"categoryId" validate:"required,min=1"`
	PublicationDate time.Time `json:"publicationDate" validate:"required"`
	Tags            []int     `json:"tags" validate:"omitempty,dive,min=1"`
}

// UpdateDocumentRequest represents the request to update an existing document
type UpdateDocumentRequest struct {
	Title           *string    `json:"title" validate:"omitempty,min=1,max=500"`
	Description     *string    `json:"description" validate:"omitempty,max=2000"`
	AuthorID        *uuid.UUID `json:"authorId" validate:"omitempty,uuid"`
	CategoryID      *int       `json:"categoryId" validate:"omitempty,min=1"`
	PublicationDate *time.Time `json:"publicationDate" validate:"omitempty"`
	Tags            *[]int     `json:"tags" validate:"omitempty,dive,min=1"`
}

// DocumentResponse represents a single document in API response
type DocumentResponse struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	Description     *string   `json:"description"`
	Author          Author    `json:"author"`
	Category        Category  `json:"category"`
	PublicationDate time.Time `json:"publicationDate"`
	Tags            []Tag     `json:"tags"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ListDocumentsQuery represents query parameters for listing documents
type ListDocumentsQuery struct {
	Page       int        `query:"page" validate:"omitempty,min=1"`
	Limit      int        `query:"limit" validate:"omitempty,min=1,max=100"`
	CategoryID *int       `query:"categoryId" validate:"omitempty,min=1"`
	AuthorID   *uuid.UUID `query:"authorId" validate:"omitempty,uuid"`
	Search     *string    `query:"search" validate:"omitempty,max=200"`
	SortBy     *string    `query:"sortBy" validate:"omitempty,oneof=title publicationDate createdAt"`
	SortOrder  *string    `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
}

// ListDocumentsResponse represents the response for listing documents
type ListDocumentsResponse struct {
	Documents  []DocumentResponse `json:"documents"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"totalPages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}
