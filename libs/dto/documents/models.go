package documents

import (
	"time"

	"github.com/google/uuid"
)

type BookmarkKind string

//go:generate easyjson -all models.go

// Document DTOs - Shared data structures for document service communication

// ListDocumentsQuery represents query parameters for listing documents
//
//easyjson:json
type ListDocumentsQuery struct {
	Page           int       `json:"page,omitempty"`
	Limit          int       `json:"limit,omitempty"`
	Offset         int       `json:"offset,omitempty"`
	AuthorID       uuid.UUID `json:"author_id,omitempty"`
	CategoryID     int       `json:"category_id,omitempty"`
	DocumentTypeID int       `json:"document_type_id,omitempty"`
	TagID          int       `json:"tag_id,omitempty"`
	Search         string    `json:"search,omitempty"`
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

// ListBookmarksQuery represents query parameters for listing bookmarks.
//
//easyjson:json
type ListBookmarksQuery struct {
	Page   int          `json:"page,omitempty"`
	Limit  int          `json:"limit,omitempty"`
	Search string       `json:"search,omitempty"`
	Kind   BookmarkKind `json:"kind,omitempty"`
	UserID uuid.UUID    `json:"user_id,omitempty"`
}

// BookmarkItem represents either a publication bookmark or a quote bookmark.
//
//easyjson:json
type BookmarkItem struct {
	ID        string       `json:"id"`
	Kind      BookmarkKind `json:"kind"`
	SavedAt   time.Time    `json:"saved_at"`
	Document  Document     `json:"document"`
	QuoteText *string      `json:"quote_text,omitempty"`
	Context   *string      `json:"context,omitempty"`
	PageLabel *string      `json:"page_label,omitempty"`
}

// ListBookmarksResponse represents the response for listing bookmarks.
//
//easyjson:json
type ListBookmarksResponse struct {
	Items      []BookmarkItem `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page,omitempty"`
	Limit      int            `json:"limit,omitempty"`
	TotalPages int            `json:"totalPages,omitempty"`
}

// CreateBookmarkRequest represents a request to save a bookmark.
//
//easyjson:json
type CreateBookmarkRequest struct {
	UserID     uuid.UUID    `json:"user_id"`
	DocumentID uuid.UUID    `json:"document_id"`
	Kind       BookmarkKind `json:"kind"`
	QuoteText  *string      `json:"quote_text,omitempty"`
	Context    *string      `json:"context,omitempty"`
	PageLabel  *string      `json:"page_label,omitempty"`
}

// CreateBookmarkResponse represents a bookmark creation response.
//
//easyjson:json
type CreateBookmarkResponse struct {
	Item BookmarkItem `json:"item"`
}

// DeleteBookmarkRequest represents a request to delete a bookmark.
//
//easyjson:json
type DeleteBookmarkRequest struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
}

// DeleteBookmarkResponse represents a bookmark deletion response.
//
//easyjson:json
type DeleteBookmarkResponse struct {
	Success bool `json:"success"`
}

// CreateDocumentRequest represents a document creation request
//
//easyjson:json
type CreateDocumentRequest struct {
	Title           string                   `json:"title"`
	Description     *string                  `json:"description,omitempty"`
	CategoryID      int                      `json:"category_id,omitempty"`
	DocumentTypeID  int                      `json:"document_type_id"`
	Participants    []DocumentParticipantRef `json:"participants,omitempty"`
	PublicationDate time.Time                `json:"publication_date"`
	Content         string                   `json:"content,omitempty"`
	TagIDs          []int                    `json:"tag_ids,omitempty"`
	CreatedBy       *uuid.UUID               `json:"created_by,omitempty"`
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
	ID              uuid.UUID                `json:"id,omitempty"`
	Title           *string                  `json:"title,omitempty"`
	Description     *string                  `json:"description,omitempty"`
	CategoryID      int                      `json:"category_id,omitempty"`
	DocumentTypeID  *int                     `json:"document_type_id,omitempty"`
	Participants    []DocumentParticipantRef `json:"participants,omitempty"`
	PublicationDate *time.Time               `json:"publication_date,omitempty"`
	Content         *string                  `json:"content,omitempty"`
	TagIDs          []int                    `json:"tag_ids,omitempty"`
	Changes         *string                  `json:"changes,omitempty"`
	UpdatedBy       *uuid.UUID               `json:"updated_by,omitempty"`
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
	ID              uuid.UUID             `json:"id"`
	Title           string                `json:"title"`
	Description     *string               `json:"description,omitempty"`
	CategoryID      int                   `json:"category_id,omitempty"`
	DocumentTypeID  int                   `json:"document_type_id"`
	PublicationDate time.Time             `json:"publication_date"`
	ContentPath     string                `json:"content_path"`
	CurrentVersion  int                   `json:"current_version"`
	Indexed         bool                  `json:"indexed"`
	CoverURL        *string               `json:"cover_url,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Author          *Author               `json:"author,omitempty"`
	Category        *Category             `json:"category,omitempty"`
	DocumentType    *DocumentType         `json:"document_type,omitempty"`
	Participants    []DocumentParticipant `json:"participants,omitempty"`
	Tags            []Tag                 `json:"tags,omitempty"`
}

// DocumentParticipantRef represents participant input for document create/update
//
//easyjson:json
type DocumentParticipantRef struct {
	AuthorID uuid.UUID `json:"author_id"`
	TypeID   int       `json:"type_id"`
}

// AuthorshipType represents participant role metadata
//
//easyjson:json
type AuthorshipType struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// DocumentParticipant represents a linked participant with a role
//
//easyjson:json
type DocumentParticipant struct {
	Author         Author         `json:"author"`
	AuthorshipType AuthorshipType `json:"authorship_type"`
}

// DocumentType represents a document kind
//
//easyjson:json
type DocumentType struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
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

// Metadata DTOs

// CreateAuthorRequest represents an author creation request
//
//easyjson:json
type CreateAuthorRequest struct {
	Name string `json:"name"`
}

// CreateAuthorResponse represents an author creation response
//
//easyjson:json
type CreateAuthorResponse struct {
	Author Author `json:"author"`
}

// ListAuthorsResponse represents the response for listing authors
//
//easyjson:json
type ListAuthorsResponse struct {
	Authors []Author `json:"authors"`
}

// CreateCategoryRequest represents a category creation request
//
//easyjson:json
type CreateCategoryRequest struct {
	Title string `json:"title"`
}

// CreateCategoryResponse represents a category creation response
//
//easyjson:json
type CreateCategoryResponse struct {
	Category Category `json:"category"`
}

// ListCategoriesResponse represents the response for listing categories
//
//easyjson:json
type ListCategoriesResponse struct {
	Categories []Category `json:"categories"`
}

// ListDocumentTypesResponse represents the response for listing document types
//
//easyjson:json
type ListDocumentTypesResponse struct {
	DocumentTypes []DocumentType `json:"document_types"`
}

// ListAuthorshipTypesResponse represents the response for listing authorship types
//
//easyjson:json
type ListAuthorshipTypesResponse struct {
	AuthorshipTypes []AuthorshipType `json:"authorship_types"`
}

// CreateTagRequest represents a tag creation request
//
//easyjson:json
type CreateTagRequest struct {
	Title string `json:"title"`
}

// CreateTagResponse represents a tag creation response
//
//easyjson:json
type CreateTagResponse struct {
	Tag Tag `json:"tag"`
}

// ListTagsResponse represents the response for listing tags
//
//easyjson:json
type ListTagsResponse struct {
	Tags []Tag `json:"tags"`
}

// UploadCoverRequest represents a cover image upload request
//
//easyjson:json
type UploadCoverRequest struct {
	ID          uuid.UUID `json:"id"`
	Data        []byte    `json:"data"`
	ContentType string    `json:"content_type"`
}

// UploadCoverResponse represents a cover image upload response
//
//easyjson:json
type UploadCoverResponse struct {
	CoverURL string `json:"cover_url"`
}
