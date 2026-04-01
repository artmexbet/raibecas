package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
)

func TestCreateBookmarkRejectsQuoteWithoutText(t *testing.T) {
	t.Parallel()

	svc := &DocumentService{}
	_, err := svc.CreateBookmark(t.Context(), domain.CreateBookmarkRequest{
		UserID:     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DocumentID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Kind:       domain.BookmarkKindQuote,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateBookmarkRejectsPublicationWithQuoteDetails(t *testing.T) {
	t.Parallel()

	quoteText := "Лишнее содержимое"
	svc := &DocumentService{}
	_, err := svc.CreateBookmark(t.Context(), domain.CreateBookmarkRequest{
		UserID:     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DocumentID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Kind:       domain.BookmarkKindPublication,
		QuoteText:  &quoteText,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateBookmarkReturnsNotFoundWhenDocumentMissing(t *testing.T) {
	t.Parallel()

	documentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	docRepo := NewMockDocumentRepository(t)
	bookmarkRepo := NewMockBookmarkRepository(t)
	docRepo.EXPECT().GetByID(mock.Anything, documentID).Return(nil, ErrNotFound).Once()

	svc := &DocumentService{docRepo: docRepo, bookmarkRepo: bookmarkRepo}
	_, err := svc.CreateBookmark(t.Context(), domain.CreateBookmarkRequest{
		UserID:     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DocumentID: documentID,
		Kind:       domain.BookmarkKindPublication,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateBookmarkReturnsExistingPublication(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	documentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	bookmarkID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	createdAt := time.Date(2026, time.March, 30, 10, 0, 0, 0, time.UTC)
	doc := testBookmarkDocument(documentID, "Системность и детерминизм", "Описание")
	bookmark := &domain.Bookmark{
		ID:         bookmarkID,
		UserID:     userID,
		DocumentID: documentID,
		Kind:       domain.BookmarkKindPublication,
		CreatedAt:  createdAt,
	}

	docRepo := NewMockDocumentRepository(t)
	bookmarkRepo := NewMockBookmarkRepository(t)
	docRepo.EXPECT().GetByID(mock.Anything, documentID).Return(&doc, nil).Twice()
	bookmarkRepo.EXPECT().GetPublicationByUserAndDocument(mock.Anything, userID, documentID).Return(bookmark, nil).Once()

	svc := &DocumentService{docRepo: docRepo, bookmarkRepo: bookmarkRepo}
	item, err := svc.CreateBookmark(t.Context(), domain.CreateBookmarkRequest{
		UserID:     userID,
		DocumentID: documentID,
		Kind:       domain.BookmarkKindPublication,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.ID != bookmarkID.String() {
		t.Fatalf("expected existing bookmark ID %s, got %s", bookmarkID, item.ID)
	}
}

func TestCreateBookmarkPersistsQuote(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	documentID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	quoteText := "Уточняющий вопрос делает предмет обсуждения общим."
	contextText := "Фрагмент семинарского обсуждения"
	pageLabel := "23"
	createdAt := time.Date(2026, time.March, 31, 9, 0, 0, 0, time.UTC)
	doc := testBookmarkDocument(documentID, "Диалог как форма научного уточнения", "Описание")

	docRepo := NewMockDocumentRepository(t)
	bookmarkRepo := NewMockBookmarkRepository(t)
	docRepo.EXPECT().GetByID(mock.Anything, documentID).Return(&doc, nil).Twice()
	bookmarkRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("*domain.Bookmark")).RunAndReturn(
		func(_ context.Context, bookmark *domain.Bookmark) error {
			bookmark.ID = uuid.MustParse("66666666-6666-6666-6666-666666666666")
			bookmark.CreatedAt = createdAt
			bookmark.UpdatedAt = createdAt
			return nil
		},
	).Once()

	svc := &DocumentService{docRepo: docRepo, bookmarkRepo: bookmarkRepo}
	item, err := svc.CreateBookmark(t.Context(), domain.CreateBookmarkRequest{
		UserID:     userID,
		DocumentID: documentID,
		Kind:       domain.BookmarkKindQuote,
		QuoteText:  &quoteText,
		Context:    &contextText,
		PageLabel:  &pageLabel,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.Kind != domain.BookmarkKindQuote {
		t.Fatalf("expected quote bookmark kind, got %s", item.Kind)
	}
	if item.QuoteText == nil || *item.QuoteText != quoteText {
		t.Fatalf("expected quote text %q, got %v", quoteText, item.QuoteText)
	}
}

func TestCreateBookmarkReturnsExistingPublicationAfterConcurrentInsert(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	documentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	bookmarkID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	createdAt := time.Date(2026, time.March, 30, 10, 0, 0, 0, time.UTC)
	doc := testBookmarkDocument(documentID, "Системность и детерминизм", "Описание")
	bookmark := &domain.Bookmark{
		ID:         bookmarkID,
		UserID:     userID,
		DocumentID: documentID,
		Kind:       domain.BookmarkKindPublication,
		CreatedAt:  createdAt,
	}

	docRepo := NewMockDocumentRepository(t)
	bookmarkRepo := NewMockBookmarkRepository(t)
	docRepo.EXPECT().GetByID(mock.Anything, documentID).Return(&doc, nil).Twice()
	bookmarkRepo.EXPECT().GetPublicationByUserAndDocument(mock.Anything, userID, documentID).Return(nil, ErrNotFound).Once()
	bookmarkRepo.EXPECT().Create(mock.Anything, mock.AnythingOfType("*domain.Bookmark")).Return(ErrInvalidInput).Once()
	bookmarkRepo.EXPECT().GetPublicationByUserAndDocument(mock.Anything, userID, documentID).Return(bookmark, nil).Once()

	svc := &DocumentService{docRepo: docRepo, bookmarkRepo: bookmarkRepo}
	item, err := svc.CreateBookmark(t.Context(), domain.CreateBookmarkRequest{
		UserID:     userID,
		DocumentID: documentID,
		Kind:       domain.BookmarkKindPublication,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.ID != bookmarkID.String() {
		t.Fatalf("expected bookmark ID %s, got %s", bookmarkID, item.ID)
	}
}

func TestDeleteBookmarkRejectsMissingIDs(t *testing.T) {
	t.Parallel()

	svc := &DocumentService{bookmarkRepo: NewMockBookmarkRepository(t)}
	err := svc.DeleteBookmark(t.Context(), uuid.Nil, uuid.Nil)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteBookmarkPropagatesNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	bookmarkID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	bookmarkRepo := NewMockBookmarkRepository(t)
	bookmarkRepo.EXPECT().Delete(mock.Anything, userID, bookmarkID).Return(ErrNotFound).Once()

	svc := &DocumentService{bookmarkRepo: bookmarkRepo}
	err := svc.DeleteBookmark(t.Context(), userID, bookmarkID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListBookmarksNormalizesPagination(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	bookmarkRepo := NewMockBookmarkRepository(t)
	expected := domain.ListBookmarksParams{
		UserID: userID,
		Page:   1,
		Limit:  16,
	}
	bookmarkRepo.EXPECT().ListByUser(mock.Anything, expected).Return([]domain.Bookmark{}, nil).Once()
	bookmarkRepo.EXPECT().CountByUser(mock.Anything, expected).Return(0, nil).Once()

	svc := &DocumentService{bookmarkRepo: bookmarkRepo}
	_, _, err := svc.ListBookmarks(t.Context(), domain.ListBookmarksParams{
		UserID: userID,
		Page:   0,
		Limit:  999,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNormalizeOptionalString(t *testing.T) {
	t.Parallel()

	blank := "   "
	if got := normalizeOptionalString(&blank); got != nil {
		t.Fatalf("expected blank string to normalize to nil, got %q", *got)
	}

	value := "  важный фрагмент  "
	got := normalizeOptionalString(&value)
	if got == nil || *got != "важный фрагмент" {
		t.Fatalf("expected trimmed value, got %v", got)
	}
}

func testBookmarkDocument(id uuid.UUID, title, description string) domain.Document {
	descriptionCopy := description
	createdAt := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	return domain.Document{
		ID:          id,
		Title:       title,
		Description: &descriptionCopy,
		Author: &domain.Author{
			ID:   uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			Name: "К. Р. Райбекас",
		},
		Category: &domain.Category{
			ID:    1,
			Title: "Философия",
		},
		Tags:      []domain.Tag{{ID: 1, Title: "методология"}},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
