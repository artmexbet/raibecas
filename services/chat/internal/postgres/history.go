package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/artmexbet/raibecas/services/chat/internal/domain"
	"github.com/artmexbet/raibecas/services/chat/internal/postgres/queries"
)

// q returns a SQLC Queries instance bound to the pool.
func (s *Store) q() *queries.Queries {
	return queries.New(s.pool)
}

func (s *Store) getOwnedSessionID(ctx context.Context, userID, sessionID string) (uuid.UUID, error) {
	parsedSessionID, err := uuid.Parse(sessionID)
	if err != nil {
		return uuid.UUID{}, domain.ErrInvalidChatSessionID
	}

	ownedSession, err := s.q().GetSessionByIDForUser(ctx, queries.GetSessionByIDForUserParams{
		ID:     parsedSessionID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, domain.ErrChatSessionNotFound
		}
		return uuid.UUID{}, fmt.Errorf("getOwnedSessionID query: %w", err)
	}

	return ownedSession.ID, nil
}

func (s *Store) resolveSessionID(ctx context.Context, userID, sessionID string, createIfMissing bool) (uuid.UUID, error) {
	if sessionID != "" {
		return s.getOwnedSessionID(ctx, userID, sessionID)
	}

	latestSessionID, err := s.q().GetLatestSession(ctx, userID)
	if err == nil {
		return latestSessionID, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, fmt.Errorf("resolveSessionID latest session: %w", err)
	}

	if !createIfMissing {
		return uuid.UUID{}, pgx.ErrNoRows
	}

	createdSessionID, err := s.q().InsertSession(ctx, queries.InsertSessionParams{
		UserID: userID,
		Title:  "Новый чат",
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("resolveSessionID insert: %w", err)
	}

	return createdSessionID, nil
}

// ensureSession returns the most recent session ID for userID, creating one if none exists.
func (s *Store) ensureSession(ctx context.Context, userID string) (uuid.UUID, error) {
	return s.resolveSessionID(ctx, userID, "", true)
}

// RetrieveChatHistory returns all messages from the latest session for userID.
func (s *Store) RetrieveChatHistory(ctx context.Context, userID, sessionID string) ([]domain.Message, error) {
	resolvedSessionID, err := s.resolveSessionID(ctx, userID, sessionID, false)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []domain.Message{}, nil
		}
		return nil, fmt.Errorf("RetrieveChatHistory session: %w", err)
	}

	rows, err := s.q().GetSessionMessages(ctx, resolvedSessionID)
	if err != nil {
		return nil, fmt.Errorf("RetrieveChatHistory messages: %w", err)
	}

	msgs := make([]domain.Message, len(rows))
	for i, r := range rows {
		msgs[i] = domain.Message{Role: r.Role, Content: r.Content}
	}
	return msgs, nil
}

// SaveMessage saves a single message to the latest (or newly created) session.
func (s *Store) SaveMessage(ctx context.Context, userID, sessionID string, message domain.Message) error {
	resolvedSessionID, err := s.resolveSessionID(ctx, userID, sessionID, true)
	if err != nil {
		return err
	}

	if err := s.q().InsertMessage(ctx, queries.InsertMessageParams{
		SessionID: resolvedSessionID,
		Role:      message.Role,
		Content:   message.Content,
	}); err != nil {
		return fmt.Errorf("SaveMessage insert: %w", err)
	}

	// Bump updated_at so latest-session detection stays correct.
	if err := s.q().BumpSessionUpdatedAt(ctx, resolvedSessionID); err != nil {
		return fmt.Errorf("SaveMessage bump session: %w", err)
	}

	return nil
}

// ClearChatHistory deletes all messages from the latest session for userID.
func (s *Store) ClearChatHistory(ctx context.Context, userID string) error {
	sessionID, err := s.q().GetLatestSession(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("ClearChatHistory session: %w", err)
	}

	if err := s.q().DeleteSessionMessages(ctx, sessionID); err != nil {
		return fmt.Errorf("ClearChatHistory delete: %w", err)
	}
	return nil
}

// GetChatSize returns the number of messages in the latest session for userID.
func (s *Store) GetChatSize(ctx context.Context, userID string) (int, error) {
	history, err := s.RetrieveChatHistory(ctx, userID, "")
	if err != nil {
		return 0, err
	}
	return len(history), nil
}

// --- Admin/History API methods ---

// GetUserSessions returns all sessions with their messages for a given userID.
func (s *Store) GetUserSessions(ctx context.Context, userID string) ([]domain.ChatSession, error) {
	rows, err := s.q().GetUserSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserSessions: %w", err)
	}

	sessions := make([]domain.ChatSession, 0, len(rows))
	for _, row := range rows {
		msgs, err := s.getSessionMessages(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, domain.ChatSession{
			ID:        row.ID.String(),
			UserID:    row.UserID,
			Title:     row.Title,
			CreatedAt: row.CreatedAt.Time.String(),
			UpdatedAt: row.UpdatedAt.Time.String(),
			Messages:  msgs,
		})
	}
	return sessions, nil
}

// CreateSession creates a new chat session for userID and returns its ID.
func (s *Store) CreateSession(ctx context.Context, userID, title string) (string, error) {
	if title == "" {
		title = "Новый чат"
	}
	id, err := s.q().InsertSession(ctx, queries.InsertSessionParams{
		UserID: userID,
		Title:  title,
	})
	if err != nil {
		return "", fmt.Errorf("CreateSession: %w", err)
	}
	return id.String(), nil
}

// getSessionMessages fetches all messages for a session by its UUID.
func (s *Store) getSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]domain.Message, error) {
	rows, err := s.q().GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getSessionMessages: %w", err)
	}

	msgs := make([]domain.Message, len(rows))
	for i, r := range rows {
		msgs[i] = domain.Message{Role: r.Role, Content: r.Content}
	}
	return msgs, nil
}
