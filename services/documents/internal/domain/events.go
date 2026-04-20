package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all events.go

// DocumentCreatedEvent represents an event when a document is created.
//
//easyjson:json
type DocumentCreatedEvent struct {
	DocumentID      uuid.UUID                  `json:"document_id"`
	Title           string                     `json:"title"`
	Description     *string                    `json:"description,omitempty"`
	CategoryID      *int                       `json:"category_id,omitempty"`
	DocumentTypeID  int                        `json:"document_type_id"`
	DocumentType    string                     `json:"document_type"`
	PublicationDate time.Time                  `json:"publication_date"`
	ContentPath     string                     `json:"content_path"`
	Version         int                        `json:"version"`
	Participants    []DocumentEventParticipant `json:"participants,omitempty"`
	Tags            []DocumentEventTag         `json:"tags,omitempty"`
	Timestamp       time.Time                  `json:"timestamp"`
}

// DocumentUpdatedEvent represents an event when a document is updated.
//
//easyjson:json
type DocumentUpdatedEvent struct {
	DocumentID      uuid.UUID                  `json:"document_id"`
	Title           string                     `json:"title"`
	Description     *string                    `json:"description,omitempty"`
	CategoryID      *int                       `json:"category_id,omitempty"`
	DocumentTypeID  int                        `json:"document_type_id"`
	DocumentType    string                     `json:"document_type"`
	PublicationDate time.Time                  `json:"publication_date"`
	OldVersion      int                        `json:"old_version"`
	NewVersion      int                        `json:"new_version"`
	ContentPath     string                     `json:"content_path"`
	Changes         *string                    `json:"changes,omitempty"`
	Participants    []DocumentEventParticipant `json:"participants,omitempty"`
	Tags            []DocumentEventTag         `json:"tags,omitempty"`
	Timestamp       time.Time                  `json:"timestamp"`
}

// DocumentDeletedEvent represents an event when a document is deleted.
//
//easyjson:json
type DocumentDeletedEvent struct {
	DocumentID uuid.UUID `json:"document_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// DocumentIndexedEvent represents an event when a document is indexed.
//
//easyjson:json
type DocumentIndexedEvent struct {
	DocumentID  uuid.UUID `json:"document_id"`
	ChunksCount int       `json:"chunks_count"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}

// DocumentEventParticipant represents a participant snapshot for indexing events.
//
//easyjson:json
type DocumentEventParticipant struct {
	AuthorID  uuid.UUID `json:"author_id"`
	Name      string    `json:"name"`
	TypeID    int       `json:"type_id"`
	TypeTitle string    `json:"type_title"`
}

// DocumentEventTag represents a tag snapshot for indexing events.
//
//easyjson:json
type DocumentEventTag struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}
