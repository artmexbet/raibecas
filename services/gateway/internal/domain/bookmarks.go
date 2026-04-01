package domain

import (
	"time"

	"github.com/google/uuid"
)

type BookmarkKind string

// ListBookmarksQuery represents query parameters for listing user bookmarks.
type ListBookmarksQuery struct {
	Page   int          `query:"page" validate:"omitempty,min=1"`
	Limit  int          `query:"limit" validate:"omitempty,min=1,max=100"`
	Search string       `query:"search" validate:"omitempty,max=255"`
	Kind   BookmarkKind `query:"kind" validate:"omitempty,oneof=publication quote"`
	UserID uuid.UUID    `json:"-" validate:"-"`
}

// BookmarkItem represents either a saved publication or a saved quote.
type BookmarkItem struct {
	ID        string       `json:"id"`
	Kind      BookmarkKind `json:"kind"`
	SavedAt   time.Time    `json:"saved_at"`
	Document  Document     `json:"document"`
	QuoteText *string      `json:"quote_text,omitempty"`
	Context   *string      `json:"context,omitempty"`
	PageLabel *string      `json:"page_label,omitempty"`
}

// ListBookmarksResponse represents the response for bookmarks listing.
type ListBookmarksResponse struct {
	Items      []BookmarkItem `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"totalPages"`
}

// CreateBookmarkRequest represents a request to save a bookmark.
type CreateBookmarkRequest struct {
	DocumentID uuid.UUID    `json:"documentId" validate:"required,uuid"`
	Kind       BookmarkKind `json:"kind" validate:"required,oneof=publication quote"`
	QuoteText  *string      `json:"quoteText,omitempty" validate:"omitempty,max=4000"`
	Context    *string      `json:"context,omitempty" validate:"omitempty,max=4000"`
	PageLabel  *string      `json:"pageLabel,omitempty" validate:"omitempty,max=64"`
	UserID     uuid.UUID    `json:"-" validate:"-"`
}

// CreateBookmarkResponse represents a bookmark creation response.
type CreateBookmarkResponse struct {
	Item BookmarkItem `json:"item"`
}

// DeleteBookmarkResponse represents a bookmark deletion response.
type DeleteBookmarkResponse struct {
	Success bool `json:"success"`
}
