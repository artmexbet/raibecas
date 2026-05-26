package domain

import (
	"time"

	"github.com/google/uuid"
)

// Note represents a user annotation.
type Note struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Title              string
	Content            string
	DocumentID         *uuid.UUID
	BookmarkID         *uuid.UUID
	PositionInDocument *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ListNotesParams holds filtering and pagination parameters for listing notes.
type ListNotesParams struct {
	Page       int
	Limit      int
	Search     string
	DocumentID *uuid.UUID
	BookmarkID *uuid.UUID
	UserID     uuid.UUID
}

// CreateNoteRequest holds the data needed to create a note.
type CreateNoteRequest struct {
	UserID             uuid.UUID
	Title              string
	Content            string
	DocumentID         *uuid.UUID
	BookmarkID         *uuid.UUID
	PositionInDocument *string
}

// UpdateNoteRequest holds the data needed to update a note.
type UpdateNoteRequest struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Title              *string
	Content            *string
	DocumentID         *uuid.UUID
	BookmarkID         *uuid.UUID
	PositionInDocument *string
	ClearDocumentID    bool
	ClearBookmarkID    bool
	ClearPosition      bool
}
