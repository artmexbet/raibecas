package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres/queries"
)

func TestToDomainBookmark(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	quote := "Фрагмент"
	contextText := "Контекст"
	pageLabel := "12"

	bookmark := toDomainBookmark(queries.DocumentBookmark{
		ID:           uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		UserID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		DocumentID:   uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Kind:         "quote",
		QuoteText:    &quote,
		QuoteContext: &contextText,
		PageLabel:    &pageLabel,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	})

	if bookmark.Kind != domain.BookmarkKindQuote {
		t.Fatalf("expected quote kind, got %s", bookmark.Kind)
	}
	if bookmark.Context == nil || *bookmark.Context != contextText {
		t.Fatalf("expected context %q, got %v", contextText, bookmark.Context)
	}
	if !bookmark.CreatedAt.Equal(createdAt) || !bookmark.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected timestamps to be preserved")
	}
}

func TestConvertBookmarkKindToPtr(t *testing.T) {
	t.Parallel()

	if got := convertBookmarkKindToPtr(""); got != nil {
		t.Fatalf("expected nil for empty kind, got %v", got)
	}

	got := convertBookmarkKindToPtr(domain.BookmarkKindPublication)
	if got == nil || *got != "publication" {
		t.Fatalf("expected publication, got %v", got)
	}
}

func TestMapBookmarkConstraintError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "bookmark document foreign key maps to not found", err: &pgconn.PgError{Code: "23503", ConstraintName: "document_bookmarks_document_id_fkey"}, want: domain.ErrNotFound},
		{name: "publication unique index maps to invalid input", err: &pgconn.PgError{Code: "23505", ConstraintName: "uq_document_bookmarks_publication"}, want: domain.ErrInvalidInput},
		{name: "bookmark quote check maps to invalid input", err: &pgconn.PgError{Code: "23514", ConstraintName: "chk_document_bookmarks_quote"}, want: domain.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mapBookmarkConstraintError(tt.err)
			if !errors.Is(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
