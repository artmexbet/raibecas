package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
)

type BookmarkRepository struct {
	pool *pgxpool.Pool
}

func NewBookmarkRepository(pool *pgxpool.Pool) *BookmarkRepository {
	return &BookmarkRepository{pool: pool}
}

func (r *BookmarkRepository) Create(ctx context.Context, bookmark *domain.Bookmark) error {
	if bookmark.ID == uuid.Nil {
		bookmark.ID = uuid.New()
	}

	const query = `
		INSERT INTO document_bookmarks (
			id, user_id, document_id, kind, quote_text, quote_context, page_label
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(
		ctx,
		query,
		bookmark.ID,
		bookmark.UserID,
		bookmark.DocumentID,
		string(bookmark.Kind),
		bookmark.QuoteText,
		bookmark.Context,
		bookmark.PageLabel,
	).Scan(&bookmark.CreatedAt, &bookmark.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create bookmark: %w", err)
	}

	return nil
}

func (r *BookmarkRepository) GetByIDForUser(ctx context.Context, userID, bookmarkID uuid.UUID) (*domain.Bookmark, error) {
	const query = `
		SELECT id, user_id, document_id, kind, quote_text, quote_context, page_label, created_at, updated_at
		FROM document_bookmarks
		WHERE id = $1 AND user_id = $2`

	return r.getBookmark(ctx, query, bookmarkID, userID)
}

func (r *BookmarkRepository) GetPublicationByUserAndDocument(ctx context.Context, userID, documentID uuid.UUID) (*domain.Bookmark, error) {
	const query = `
		SELECT id, user_id, document_id, kind, quote_text, quote_context, page_label, created_at, updated_at
		FROM document_bookmarks
		WHERE user_id = $1 AND document_id = $2 AND kind = 'publication'
		LIMIT 1`

	return r.getBookmark(ctx, query, userID, documentID)
}

func (r *BookmarkRepository) ListByUser(ctx context.Context, params domain.ListBookmarksParams) ([]domain.Bookmark, error) {
	query, args := buildBookmarkListQuery(params, false)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}
	defer rows.Close()

	bookmarks := make([]domain.Bookmark, 0)
	for rows.Next() {
		bookmark, err := scanBookmark(rows)
		if err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, bookmark)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bookmarks rows: %w", err)
	}

	return bookmarks, nil
}

func (r *BookmarkRepository) CountByUser(ctx context.Context, params domain.ListBookmarksParams) (int, error) {
	query, args := buildBookmarkListQuery(params, true)

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count bookmarks: %w", err)
	}
	return count, nil
}

func (r *BookmarkRepository) Delete(ctx context.Context, userID, bookmarkID uuid.UUID) error {
	const query = `DELETE FROM document_bookmarks WHERE id = $1 AND user_id = $2`

	commandTag, err := r.pool.Exec(ctx, query, bookmarkID, userID)
	if err != nil {
		return fmt.Errorf("delete bookmark: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *BookmarkRepository) getBookmark(ctx context.Context, query string, args ...any) (*domain.Bookmark, error) {
	row := r.pool.QueryRow(ctx, query, args...)

	bookmark, err := scanBookmark(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &bookmark, nil
}

func buildBookmarkListQuery(params domain.ListBookmarksParams, countOnly bool) (string, []any) {
	var builder strings.Builder
	args := []any{params.UserID}
	argIndex := 2

	if countOnly {
		builder.WriteString(`
			SELECT COUNT(*)
			FROM document_bookmarks b
			JOIN documents d ON d.id = b.document_id
			JOIN authors a ON a.id = d.author_id
			JOIN categories c ON c.id = d.category_id
			WHERE b.user_id = $1`)
	} else {
		builder.WriteString(`
			SELECT b.id, b.user_id, b.document_id, b.kind, b.quote_text, b.quote_context, b.page_label, b.created_at, b.updated_at
			FROM document_bookmarks b
			JOIN documents d ON d.id = b.document_id
			JOIN authors a ON a.id = d.author_id
			JOIN categories c ON c.id = d.category_id
			WHERE b.user_id = $1`)
	}

	if params.Kind != "" {
		builder.WriteString(fmt.Sprintf(" AND b.kind = $%d", argIndex))
		args = append(args, string(params.Kind))
		argIndex++
	}

	if search := strings.TrimSpace(params.Search); search != "" {
		builder.WriteString(fmt.Sprintf(`
			AND (
				d.title ILIKE '%%' || $%d || '%%'
				OR COALESCE(d.description, '') ILIKE '%%' || $%d || '%%'
				OR a.name ILIKE '%%' || $%d || '%%'
				OR c.title ILIKE '%%' || $%d || '%%'
				OR COALESCE(b.quote_text, '') ILIKE '%%' || $%d || '%%'
				OR COALESCE(b.quote_context, '') ILIKE '%%' || $%d || '%%'
				OR EXISTS (
					SELECT 1
					FROM document_tags dt
					JOIN tags t ON t.id = dt.tag_id
					WHERE dt.document_id = d.id
					  AND t.title ILIKE '%%' || $%d || '%%'
				)
			)`, argIndex, argIndex, argIndex, argIndex, argIndex, argIndex, argIndex))
		args = append(args, search)
		argIndex++
	}

	if !countOnly {
		builder.WriteString(fmt.Sprintf(" ORDER BY b.created_at DESC, b.id DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1))
		args = append(args, params.Limit, (max(params.Page, 1)-1)*params.Limit)
	}

	return builder.String(), args
}

func scanBookmark(scanner interface{ Scan(dest ...any) error }) (domain.Bookmark, error) {
	var bookmark domain.Bookmark
	var kind string

	err := scanner.Scan(
		&bookmark.ID,
		&bookmark.UserID,
		&bookmark.DocumentID,
		&kind,
		&bookmark.QuoteText,
		&bookmark.Context,
		&bookmark.PageLabel,
		&bookmark.CreatedAt,
		&bookmark.UpdatedAt,
	)
	if err != nil {
		return domain.Bookmark{}, fmt.Errorf("scan bookmark: %w", err)
	}

	bookmark.Kind = domain.BookmarkKind(kind)
	return bookmark, nil
}
