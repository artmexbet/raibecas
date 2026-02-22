package queries

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Helper functions for converting between domain types and pgx nullable types

// NullString converts *string to pgtype.Text
func NullString(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// NullInt32 converts *int to pgtype.Int4
func NullInt32(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

// NullInt32Ptr converts *int to pgtype.Int4
func NullInt32Ptr(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

// NullUUID converts *uuid.UUID to pgtype.UUID
func NullUUID(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

// NullDate converts *time.Time to pgtype.Date
func NullDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{
		Time:  *t,
		Valid: true,
	}
}

// NullTimestamp converts *time.Time to pgtype.Timestamp
func NullTimestamp(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{
		Time:  *t,
		Valid: true,
	}
}
