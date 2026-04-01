package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres/queries"
)

type BookmarkRepository struct {
	queries *queries.Queries
}

func NewBookmarkRepository(q *queries.Queries) *BookmarkRepository {
	return &BookmarkRepository{queries: q}
}

func (r *BookmarkRepository) Create(ctx context.Context, bookmark *domain.Bookmark) error {
	if bookmark.ID == uuid.Nil {
		bookmark.ID = uuid.New()
	}

	created, err := r.queries.CreateBookmark(ctx, queries.CreateBookmarkParams{
		ID:           bookmark.ID,
		UserID:       bookmark.UserID,
		DocumentID:   bookmark.DocumentID,
		Kind:         string(bookmark.Kind),
		QuoteText:    bookmark.QuoteText,
		QuoteContext: bookmark.Context,
		PageLabel:    bookmark.PageLabel,
	})
	if err != nil {
		return fmt.Errorf("create bookmark: %w", mapBookmarkConstraintError(err))
	}

	bookmark.CreatedAt = created.CreatedAt
	bookmark.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *BookmarkRepository) GetByIDForUser(ctx context.Context, userID, bookmarkID uuid.UUID) (*domain.Bookmark, error) {
	row, err := r.queries.GetBookmarkByIDForUser(ctx, queries.GetBookmarkByIDForUserParams{
		ID:     bookmarkID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get bookmark by id for user: %w", err)
	}

	bookmark := toDomainBookmark(row)
	return &bookmark, nil
}

func (r *BookmarkRepository) GetPublicationByUserAndDocument(ctx context.Context, userID, documentID uuid.UUID) (*domain.Bookmark, error) {
	row, err := r.queries.GetPublicationBookmarkByUserAndDocument(ctx, queries.GetPublicationBookmarkByUserAndDocumentParams{
		UserID:     userID,
		DocumentID: documentID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get publication bookmark by user and document: %w", err)
	}

	bookmark := toDomainBookmark(row)
	return &bookmark, nil
}

func (r *BookmarkRepository) ListByUser(ctx context.Context, params domain.ListBookmarksParams) ([]domain.Bookmark, error) {
	rows, err := r.queries.ListBookmarksByUser(ctx, queries.ListBookmarksByUserParams{
		UserID: params.UserID,
		Limit:  int32(params.Limit),
		Offset: int32((max(params.Page, 1) - 1) * params.Limit),
		Kind:   convertBookmarkKindToPtr(params.Kind),
		Search: convertStringToPtr(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}

	bookmarks := make([]domain.Bookmark, len(rows))
	for i, row := range rows {
		bookmarks[i] = toDomainBookmark(row)
	}
	return bookmarks, nil
}

func (r *BookmarkRepository) CountByUser(ctx context.Context, params domain.ListBookmarksParams) (int, error) {
	count, err := r.queries.CountBookmarksByUser(ctx, queries.CountBookmarksByUserParams{
		UserID: params.UserID,
		Kind:   convertBookmarkKindToPtr(params.Kind),
		Search: convertStringToPtr(params.Search),
	})
	if err != nil {
		return 0, fmt.Errorf("count bookmarks: %w", err)
	}
	return int(count), nil
}

func (r *BookmarkRepository) Delete(ctx context.Context, userID, bookmarkID uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteBookmark(ctx, queries.DeleteBookmarkParams{
		ID:     bookmarkID,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("delete bookmark: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func toDomainBookmark(bookmark queries.DocumentBookmark) domain.Bookmark {
	return domain.Bookmark{
		ID:         bookmark.ID,
		UserID:     bookmark.UserID,
		DocumentID: bookmark.DocumentID,
		Kind:       domain.BookmarkKind(bookmark.Kind),
		QuoteText:  bookmark.QuoteText,
		Context:    bookmark.QuoteContext,
		PageLabel:  bookmark.PageLabel,
		CreatedAt:  bookmark.CreatedAt,
		UpdatedAt:  bookmark.UpdatedAt,
	}
}

func convertBookmarkKindToPtr(kind domain.BookmarkKind) *string {
	if kind == "" {
		return nil
	}
	value := string(kind)
	return &value
}

func mapBookmarkConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch {
	case pgErr.Code == "23503" && pgErr.ConstraintName == "document_bookmarks_document_id_fkey":
		return domain.ErrNotFound
	case pgErr.Code == "23505" && pgErr.ConstraintName == "uq_document_bookmarks_publication":
		return domain.ErrInvalidInput
	case pgErr.Code == "23514" && (pgErr.ConstraintName == "chk_document_bookmarks_kind" || pgErr.ConstraintName == "chk_document_bookmarks_quote"):
		return domain.ErrInvalidInput
	default:
		return err
	}
}
