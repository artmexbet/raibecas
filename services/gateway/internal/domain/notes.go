package domain

import (
	"time"

	"github.com/google/uuid"
)

// ListNotesQuery represents query parameters for listing user notes.
type ListNotesQuery struct {
	Page       int        `query:"page" validate:"omitempty,min=1"`
	Limit      int        `query:"limit" validate:"omitempty,min=1,max=100"`
	Search     string     `query:"search" validate:"omitempty,max=255"`
	DocumentID *uuid.UUID `query:"document_id" validate:"omitempty"`
	BookmarkID *uuid.UUID `query:"bookmark_id" validate:"omitempty"`
	UserID     uuid.UUID  `json:"-" validate:"-"`
}

// NoteItem represents a single note.
type NoteItem struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Content            string     `json:"content"`
	DocumentID         *uuid.UUID `json:"document_id,omitempty"`
	BookmarkID         *uuid.UUID `json:"bookmark_id,omitempty"`
	PositionInDocument *string    `json:"position_in_document,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// ListNotesResponse represents the response for notes listing.
type ListNotesResponse struct {
	Items      []NoteItem `json:"items"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"totalPages"`
}

// GetNoteResponse represents a single note response.
type GetNoteResponse struct {
	Item NoteItem `json:"item"`
}

// CreateNoteRequest represents a request to create a note.
type CreateNoteRequest struct {
	Title              string     `json:"title" validate:"required,max=100"`
	Content            string     `json:"content" validate:"required,max=15000"`
	DocumentID         *uuid.UUID `json:"documentId,omitempty"`
	BookmarkID         *uuid.UUID `json:"bookmarkId,omitempty"`
	PositionInDocument *string    `json:"positionInDocument,omitempty" validate:"omitempty,max=255"`
	UserID             uuid.UUID  `json:"-" validate:"-"`
}

// CreateNoteResponse represents a note creation response.
type CreateNoteResponse struct {
	Item NoteItem `json:"item"`
}

// UpdateNoteRequest represents a request to update a note.
type UpdateNoteRequest struct {
	ID                 uuid.UUID  `json:"-" validate:"-"`
	Title              *string    `json:"title,omitempty" validate:"omitempty,min=1,max=100"`
	Content            *string    `json:"content,omitempty" validate:"omitempty,max=15000"`
	DocumentID         *uuid.UUID `json:"documentId,omitempty"`
	BookmarkID         *uuid.UUID `json:"bookmarkId,omitempty"`
	PositionInDocument *string    `json:"positionInDocument,omitempty" validate:"omitempty,max=255"`
	ClearDocumentID    bool       `json:"clearDocumentId,omitempty"`
	ClearBookmarkID    bool       `json:"clearBookmarkId,omitempty"`
	ClearPosition      bool       `json:"clearPosition,omitempty"`
	UserID             uuid.UUID  `json:"-" validate:"-"`
}

// UpdateNoteResponse represents a note update response.
type UpdateNoteResponse struct {
	Item NoteItem `json:"item"`
}
