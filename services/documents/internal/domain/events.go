package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all events.go

// DocumentCreatedEvent represents an event when a document is created
//
//easyjson:json
type DocumentCreatedEvent struct {
	DocumentID      uuid.UUID `json:"document_id"`
	Title           string    `json:"title"`
	AuthorID        uuid.UUID `json:"author_id"`
	CategoryID      int       `json:"category_id"`
	PublicationDate time.Time `json:"publication_date"`
	ContentPath     string    `json:"content_path"`
	Version         int       `json:"version"`
	Timestamp       time.Time `json:"timestamp"`
}

// DocumentUpdatedEvent represents an event when a document is updated
//
//easyjson:json
type DocumentUpdatedEvent struct {
	DocumentID  uuid.UUID `json:"document_id"`
	OldVersion  int       `json:"old_version"`
	NewVersion  int       `json:"new_version"`
	ContentPath string    `json:"content_path"`
	Changes     *string   `json:"changes,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// DocumentDeletedEvent represents an event when a document is deleted
//
//easyjson:json
type DocumentDeletedEvent struct {
	DocumentID uuid.UUID `json:"document_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// DocumentIndexedEvent represents an event when a document is indexed
//
//easyjson:json
type DocumentIndexedEvent struct {
	DocumentID  uuid.UUID `json:"document_id"`
	ChunksCount int       `json:"chunks_count"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}
