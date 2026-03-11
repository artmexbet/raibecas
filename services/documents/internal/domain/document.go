package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all document.go

// Author represents a scientific work author
//
//easyjson:json
type Author struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Bio       *string   `db:"bio" json:"bio,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Category represents a document category
//
//easyjson:json
type Category struct {
	ID          int       `db:"id" json:"id"`
	Title       string    `db:"title" json:"title"`
	Description *string   `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// Tag represents a document tag
//
//easyjson:json
type Tag struct {
	ID        int       `db:"id" json:"id"`
	Title     string    `db:"title" json:"title"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Document represents a scientific document
//
//easyjson:json
type Document struct {
	ID              uuid.UUID `db:"id" json:"id"`
	Title           string    `db:"title" json:"title"`
	Description     *string   `db:"description" json:"description,omitempty"`
	AuthorID        uuid.UUID `db:"author_id" json:"author_id"`
	CategoryID      int       `db:"category_id" json:"category_id"`
	PublicationDate time.Time `db:"publication_date" json:"publication_date"`
	ContentPath     string    `db:"content_path" json:"content_path"`
	CoverPath       *string   `db:"cover_path" json:"cover_path,omitempty"`
	CurrentVersion  int       `db:"current_version" json:"current_version"`
	Indexed         bool      `db:"indexed" json:"indexed"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	Author          *Author   `json:"author,omitempty"`
	Category        *Category `json:"category,omitempty"`
	Tags            []Tag     `json:"tags,omitempty"`
	CoverURL        *string   `db:"-" json:"cover_url,omitempty"`
}

// DocumentVersion represents a version of a document
//
//easyjson:json
type DocumentVersion struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	DocumentID  uuid.UUID  `db:"document_id" json:"document_id"`
	Version     int        `db:"version" json:"version"`
	ContentPath string     `db:"content_path" json:"content_path"`
	Changes     *string    `db:"changes" json:"changes,omitempty"`
	CreatedBy   *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

// CreateDocumentRequest represents a request to create a document
//
//easyjson:json
type CreateDocumentRequest struct {
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	AuthorID        uuid.UUID  `json:"author_id"`
	CategoryID      int        `json:"category_id"`
	PublicationDate time.Time  `json:"publication_date"`
	Content         string     `json:"content"`
	TagIDs          []int      `json:"tag_ids,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
}

// UpdateDocumentRequest represents a request to update a document
//
//easyjson:json
type UpdateDocumentRequest struct {
	Title           *string    `json:"title,omitempty"`
	Description     *string    `json:"description,omitempty"`
	AuthorID        *uuid.UUID `json:"author_id,omitempty"`
	CategoryID      *int       `json:"category_id,omitempty"`
	PublicationDate *time.Time `json:"publication_date,omitempty"`
	Content         *string    `json:"content,omitempty"`
	TagIDs          []int      `json:"tag_ids,omitempty"`
	Changes         *string    `json:"changes,omitempty"`
	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty"`
	CoverPath       *string    `json:"cover_path,omitempty"`
}

// ListDocumentsParams represents parameters for listing documents
type ListDocumentsParams struct {
	Limit      int
	Offset     int
	AuthorID   *uuid.UUID
	CategoryID *int32
	TagID      *int
	Search     string
}
