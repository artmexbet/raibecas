package pg

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func UUIDFromGoogle(uuid uuid.UUID) pgtype.UUID {
	var pgUUID pgtype.UUID
	_ = pgUUID.Scan(uuid)
	return pgUUID
}

func GoogleUUIDFromPG(pgUUID pgtype.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := id.Scan(pgUUID)
	return id, err
}

// ConvertToPGTime converts a time.Time to pgtype.Timestamp
func ConvertToPGTime(t time.Time) pgtype.Timestamp {
	var pgTime pgtype.Timestamp
	_ = pgTime.Scan(t)
	return pgTime
}
