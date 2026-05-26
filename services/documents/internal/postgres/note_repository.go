package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres/queries"
)

// NoteRepository implements note persistence using PostgreSQL.
type NoteRepository struct {
	queries *queries.Queries
}

// NewNoteRepository creates a new NoteRepository.
func NewNoteRepository(q *queries.Queries) *NoteRepository {
	return &NoteRepository{queries: q}
}

func (r *NoteRepository) Create(ctx context.Context, note *domain.Note) error {
	if note.ID == uuid.Nil {
		note.ID = uuid.New()
	}

	created, err := r.queries.CreateNote(ctx, queries.CreateNoteParams{
		ID:                 note.ID,
		UserID:             note.UserID,
		Title:              note.Title,
		Content:            note.Content,
		DocumentID:         note.DocumentID,
		BookmarkID:         note.BookmarkID,
		PositionInDocument: note.PositionInDocument,
	})
	if err != nil {
		return fmt.Errorf("create note: %w", err)
	}

	note.CreatedAt = created.CreatedAt
	note.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *NoteRepository) GetByIDForUser(ctx context.Context, userID, noteID uuid.UUID) (*domain.Note, error) {
	row, err := r.queries.GetNoteByIDForUser(ctx, queries.GetNoteByIDForUserParams{
		ID:     noteID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get note by id for user: %w", err)
	}

	note := toDomainNote(row)
	return &note, nil
}

func (r *NoteRepository) ListByUser(ctx context.Context, params domain.ListNotesParams) ([]domain.Note, error) {
	rows, err := r.queries.ListNotesByUser(ctx, queries.ListNotesByUserParams{
		UserID:     params.UserID,
		Limit:      int32(params.Limit),
		Offset:     int32((max(params.Page, 1) - 1) * params.Limit),
		Search:     convertStringToPtr(params.Search),
		DocumentID: params.DocumentID,
		BookmarkID: params.BookmarkID,
	})
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}

	notes := make([]domain.Note, len(rows))
	for i, row := range rows {
		notes[i] = toDomainNote(row)
	}
	return notes, nil
}

func (r *NoteRepository) CountByUser(ctx context.Context, params domain.ListNotesParams) (int, error) {
	count, err := r.queries.CountNotesByUser(ctx, queries.CountNotesByUserParams{
		UserID:     params.UserID,
		Search:     convertStringToPtr(params.Search),
		DocumentID: params.DocumentID,
		BookmarkID: params.BookmarkID,
	})
	if err != nil {
		return 0, fmt.Errorf("count notes: %w", err)
	}
	return int(count), nil
}

func (r *NoteRepository) Update(ctx context.Context, req domain.UpdateNoteRequest) (*domain.Note, error) {
	updated, err := r.queries.UpdateNote(ctx, queries.UpdateNoteParams{
		ID:                 req.ID,
		UserID:             req.UserID,
		Title:              req.Title,
		Content:            req.Content,
		ClearDocumentID:    req.ClearDocumentID,
		DocumentID:         req.DocumentID,
		ClearBookmarkID:    req.ClearBookmarkID,
		BookmarkID:         req.BookmarkID,
		ClearPosition:      req.ClearPosition,
		PositionInDocument: req.PositionInDocument,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update note: %w", err)
	}

	note := toDomainNote(updated)
	return &note, nil
}

func (r *NoteRepository) Delete(ctx context.Context, userID, noteID uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteNote(ctx, queries.DeleteNoteParams{
		ID:     noteID,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func toDomainNote(note queries.Note) domain.Note {
	return domain.Note{
		ID:                 note.ID,
		UserID:             note.UserID,
		Title:              note.Title,
		Content:            note.Content,
		DocumentID:         note.DocumentID,
		BookmarkID:         note.BookmarkID,
		PositionInDocument: note.PositionInDocument,
		CreatedAt:          note.CreatedAt,
		UpdatedAt:          note.UpdatedAt,
	}
}
