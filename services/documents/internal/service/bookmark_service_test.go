package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

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

func TestCreateBookmarkReturnsExistingPublication(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	documentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	bookmarkID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	createdAt := time.Date(2026, time.March, 30, 10, 0, 0, 0, time.UTC)

	docRepo := &stubDocumentRepository{
		documents: map[uuid.UUID]domain.Document{
			documentID: testBookmarkDocument(documentID, "Системность и детерминизм", "Описание"),
		},
	}
	bookmarkRepo := &stubBookmarkRepository{
		publicationByDocument: map[string]domain.Bookmark{
			publicationKey(userID, documentID): {
				ID:         bookmarkID,
				UserID:     userID,
				DocumentID: documentID,
				Kind:       domain.BookmarkKindPublication,
				CreatedAt:  createdAt,
			},
		},
	}

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
	if bookmarkRepo.createCalls != 0 {
		t.Fatalf("expected repository Create not to be called, got %d calls", bookmarkRepo.createCalls)
	}
}

func TestCreateBookmarkPersistsQuote(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	documentID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	quoteText := "Уточняющий вопрос делает предмет обсуждения общим."
	contextText := "Фрагмент семинарского обсуждения"
	pageLabel := "23"

	docRepo := &stubDocumentRepository{
		documents: map[uuid.UUID]domain.Document{
			documentID: testBookmarkDocument(documentID, "Диалог как форма научного уточнения", "Описание"),
		},
	}
	bookmarkRepo := &stubBookmarkRepository{}

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
	if bookmarkRepo.createCalls != 1 {
		t.Fatalf("expected repository Create to be called once, got %d", bookmarkRepo.createCalls)
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

type stubDocumentRepository struct {
	documents map[uuid.UUID]domain.Document
}

func (s *stubDocumentRepository) Create(context.Context, *domain.Document) error { return nil }
func (s *stubDocumentRepository) List(context.Context, domain.ListDocumentsParams) ([]domain.Document, error) {
	return nil, nil
}
func (s *stubDocumentRepository) Count(context.Context, domain.ListDocumentsParams) (int, error) {
	return 0, nil
}
func (s *stubDocumentRepository) Update(context.Context, *domain.Document) error { return nil }
func (s *stubDocumentRepository) Delete(context.Context, uuid.UUID) error        { return nil }
func (s *stubDocumentRepository) UpdateIndexedStatus(context.Context, uuid.UUID, bool) error {
	return nil
}
func (s *stubDocumentRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Document, error) {
	doc, ok := s.documents[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copyDoc := doc
	return &copyDoc, nil
}

type stubBookmarkRepository struct {
	publicationByDocument map[string]domain.Bookmark
	createCalls           int
}

func (s *stubBookmarkRepository) Create(_ context.Context, bookmark *domain.Bookmark) error {
	s.createCalls++
	bookmark.ID = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	bookmark.CreatedAt = time.Date(2026, time.March, 31, 9, 0, 0, 0, time.UTC)
	bookmark.UpdatedAt = bookmark.CreatedAt
	return nil
}
func (s *stubBookmarkRepository) GetByIDForUser(context.Context, uuid.UUID, uuid.UUID) (*domain.Bookmark, error) {
	return nil, domain.ErrNotFound
}
func (s *stubBookmarkRepository) GetPublicationByUserAndDocument(_ context.Context, userID, documentID uuid.UUID) (*domain.Bookmark, error) {
	if s.publicationByDocument == nil {
		return nil, domain.ErrNotFound
	}
	bookmark, ok := s.publicationByDocument[publicationKey(userID, documentID)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copyBookmark := bookmark
	return &copyBookmark, nil
}
func (s *stubBookmarkRepository) ListByUser(context.Context, domain.ListBookmarksParams) ([]domain.Bookmark, error) {
	return nil, nil
}
func (s *stubBookmarkRepository) CountByUser(context.Context, domain.ListBookmarksParams) (int, error) {
	return 0, nil
}
func (s *stubBookmarkRepository) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func publicationKey(userID, documentID uuid.UUID) string {
	return userID.String() + ":" + documentID.String()
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
