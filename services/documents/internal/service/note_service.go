package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
)

const (
	maxNoteTitleLength   = 100
	maxNoteContentLength = 15000
)

// ListNotes returns a paginated list of notes for the given user.
func (s *DocumentService) ListNotes(ctx context.Context, params domain.ListNotesParams) ([]domain.Note, int, error) {
	if params.UserID == uuid.Nil {
		return nil, 0, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}

	params.Page = max(params.Page, 1)
	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 16
	}

	notes, err := s.noteRepo.ListByUser(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list notes: %w", err)
	}

	total, err := s.noteRepo.CountByUser(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("count notes: %w", err)
	}

	return notes, total, nil
}

// GetNote returns a single note by ID for the given user.
func (s *DocumentService) GetNote(ctx context.Context, userID, noteID uuid.UUID) (*domain.Note, error) {
	if userID == uuid.Nil || noteID == uuid.Nil {
		return nil, fmt.Errorf("%w: user id and note id are required", ErrInvalidInput)
	}

	note, err := s.noteRepo.GetByIDForUser(ctx, userID, noteID)
	if err != nil {
		return nil, fmt.Errorf("get note: %w", err)
	}

	return note, nil
}

// CreateNote creates a new note for the given user.
func (s *DocumentService) CreateNote(ctx context.Context, req domain.CreateNoteRequest) (*domain.Note, error) {
	if req.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if len([]rune(req.Title)) > maxNoteTitleLength {
		return nil, fmt.Errorf("%w: title must be at most %d characters", ErrInvalidInput, maxNoteTitleLength)
	}

	if len([]rune(req.Content)) > maxNoteContentLength {
		return nil, fmt.Errorf("%w: content must be at most %d characters", ErrInvalidInput, maxNoteContentLength)
	}

	req.PositionInDocument = normalizeOptionalString(req.PositionInDocument)

	// Validate document_id reference if provided
	if req.DocumentID != nil {
		if _, err := s.docRepo.GetByID(ctx, *req.DocumentID); err != nil {
			return nil, fmt.Errorf("get referenced document: %w", err)
		}
	}

	// Validate bookmark_id reference if provided
	if req.BookmarkID != nil && req.UserID != uuid.Nil {
		if _, err := s.bookmarkRepo.GetByIDForUser(ctx, req.UserID, *req.BookmarkID); err != nil {
			return nil, fmt.Errorf("get referenced bookmark: %w", err)
		}
	}

	note := &domain.Note{
		UserID:             req.UserID,
		Title:              req.Title,
		Content:            req.Content,
		DocumentID:         req.DocumentID,
		BookmarkID:         req.BookmarkID,
		PositionInDocument: req.PositionInDocument,
	}

	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}

	return note, nil
}

// UpdateNote updates an existing note for the given user.
func (s *DocumentService) UpdateNote(ctx context.Context, req domain.UpdateNoteRequest) (*domain.Note, error) {
	if req.UserID == uuid.Nil || req.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: user id and note id are required", ErrInvalidInput)
	}

	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: title cannot be empty", ErrInvalidInput)
		}
		if len([]rune(trimmed)) > maxNoteTitleLength {
			return nil, fmt.Errorf("%w: title must be at most %d characters", ErrInvalidInput, maxNoteTitleLength)
		}
		req.Title = &trimmed
	}

	if req.Content != nil {
		if len([]rune(*req.Content)) > maxNoteContentLength {
			return nil, fmt.Errorf("%w: content must be at most %d characters", ErrInvalidInput, maxNoteContentLength)
		}
	}

	// Validate document_id reference if provided
	if req.DocumentID != nil {
		if _, err := s.docRepo.GetByID(ctx, *req.DocumentID); err != nil {
			return nil, fmt.Errorf("get referenced document: %w", err)
		}
	}

	// Validate bookmark_id reference if provided
	if req.BookmarkID != nil {
		if _, err := s.bookmarkRepo.GetByIDForUser(ctx, req.UserID, *req.BookmarkID); err != nil {
			return nil, fmt.Errorf("get referenced bookmark: %w", err)
		}
	}

	req.PositionInDocument = normalizeOptionalString(req.PositionInDocument)

	note, err := s.noteRepo.Update(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}

	return note, nil
}

// DeleteNote deletes a note for the given user.
func (s *DocumentService) DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error {
	if userID == uuid.Nil || noteID == uuid.Nil {
		return fmt.Errorf("%w: user id and note id are required", ErrInvalidInput)
	}

	if err := s.noteRepo.Delete(ctx, userID, noteID); err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	return nil
}
