package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
)

func (s *DocumentService) ListBookmarks(ctx context.Context, params domain.ListBookmarksParams) ([]domain.BookmarkItem, int, error) {
	if params.UserID == uuid.Nil {
		return nil, 0, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}

	params.Page = max(params.Page, 1)
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 16
	}

	bookmarks, err := s.bookmarkRepo.ListByUser(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list bookmarks: %w", err)
	}

	total, err := s.bookmarkRepo.CountByUser(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("count bookmarks: %w", err)
	}

	items := make([]domain.BookmarkItem, len(bookmarks))
	for i, bookmark := range bookmarks {
		item, err := s.buildBookmarkItem(ctx, bookmark)
		if err != nil {
			return nil, 0, fmt.Errorf("build bookmark item: %w", err)
		}
		items[i] = item
	}

	return items, total, nil
}

func (s *DocumentService) CreateBookmark(ctx context.Context, req domain.CreateBookmarkRequest) (*domain.BookmarkItem, error) {
	if req.UserID == uuid.Nil || req.DocumentID == uuid.Nil {
		return nil, fmt.Errorf("%w: user id and document id are required", ErrInvalidInput)
	}
	if req.Kind != domain.BookmarkKindPublication && req.Kind != domain.BookmarkKindQuote {
		return nil, fmt.Errorf("%w: unsupported bookmark kind", ErrInvalidInput)
	}
	if req.Kind == domain.BookmarkKindQuote {
		if req.QuoteText == nil || strings.TrimSpace(*req.QuoteText) == "" {
			return nil, fmt.Errorf("%w: quote text is required for quote bookmarks", ErrInvalidInput)
		}
	}
	if req.Kind == domain.BookmarkKindPublication {
		req.QuoteText = nil
		req.Context = nil
		req.PageLabel = nil
	}

	if _, err := s.docRepo.GetByID(ctx, req.DocumentID); err != nil {
		return nil, fmt.Errorf("get bookmarked document: %w", err)
	}

	if req.Kind == domain.BookmarkKindPublication {
		existing, err := s.bookmarkRepo.GetPublicationByUserAndDocument(ctx, req.UserID, req.DocumentID)
		if err == nil {
			item, buildErr := s.buildBookmarkItem(ctx, *existing)
			if buildErr != nil {
				return nil, fmt.Errorf("build existing bookmark item: %w", buildErr)
			}
			return &item, nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("check existing publication bookmark: %w", err)
		}
	}

	bookmark := &domain.Bookmark{
		UserID:     req.UserID,
		DocumentID: req.DocumentID,
		Kind:       req.Kind,
		QuoteText:  normalizeOptionalString(req.QuoteText),
		Context:    normalizeOptionalString(req.Context),
		PageLabel:  normalizeOptionalString(req.PageLabel),
	}

	if err := s.bookmarkRepo.Create(ctx, bookmark); err != nil {
		return nil, fmt.Errorf("create bookmark: %w", err)
	}

	item, err := s.buildBookmarkItem(ctx, *bookmark)
	if err != nil {
		return nil, fmt.Errorf("build created bookmark item: %w", err)
	}

	return &item, nil
}

func (s *DocumentService) DeleteBookmark(ctx context.Context, userID, bookmarkID uuid.UUID) error {
	if userID == uuid.Nil || bookmarkID == uuid.Nil {
		return fmt.Errorf("%w: user id and bookmark id are required", ErrInvalidInput)
	}

	if err := s.bookmarkRepo.Delete(ctx, userID, bookmarkID); err != nil {
		return fmt.Errorf("delete bookmark: %w", err)
	}

	return nil
}

func (s *DocumentService) buildBookmarkItem(ctx context.Context, bookmark domain.Bookmark) (domain.BookmarkItem, error) {
	doc, err := s.docRepo.GetByID(ctx, bookmark.DocumentID)
	if err != nil {
		return domain.BookmarkItem{}, fmt.Errorf("get bookmark document: %w", err)
	}
	s.enrichCoverURL(ctx, doc)

	return domain.BookmarkItem{
		ID:        bookmark.ID.String(),
		Kind:      bookmark.Kind,
		SavedAt:   bookmark.CreatedAt,
		Document:  *doc,
		QuoteText: bookmark.QuoteText,
		Context:   bookmark.Context,
		PageLabel: bookmark.PageLabel,
	}, nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
