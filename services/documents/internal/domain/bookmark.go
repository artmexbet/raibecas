package domain

import (
	"time"

	"github.com/google/uuid"
)

type BookmarkKind string

const (
	BookmarkKindPublication BookmarkKind = "publication"
	BookmarkKindQuote       BookmarkKind = "quote"
)

type ListBookmarksParams struct {
	Page   int
	Limit  int
	Search string
	Kind   BookmarkKind
	UserID uuid.UUID
}

type BookmarkItem struct {
	ID        string
	Kind      BookmarkKind
	SavedAt   time.Time
	Document  Document
	QuoteText *string
	Context   *string
	PageLabel *string
}

type Bookmark struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DocumentID uuid.UUID
	Kind       BookmarkKind
	QuoteText  *string
	Context    *string
	PageLabel  *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateBookmarkRequest struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	Kind       BookmarkKind
	QuoteText  *string
	Context    *string
	PageLabel  *string
}
