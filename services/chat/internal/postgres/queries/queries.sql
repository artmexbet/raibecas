-- name: GetLatestSession :one
SELECT id FROM chat_sessions
WHERE user_id = $1
ORDER BY updated_at DESC
LIMIT 1;

-- name: InsertSession :one
INSERT INTO chat_sessions (user_id, title)
VALUES ($1, $2)
RETURNING id;

-- name: GetSessionByIDForUser :one
SELECT id, user_id, title, created_at, updated_at
FROM chat_sessions
WHERE id = $1 AND user_id = $2
LIMIT 1;

-- name: GetSessionMessages :many
SELECT role, content FROM chat_messages
WHERE session_id = $1
ORDER BY created_at ASC;

-- name: InsertMessage :exec
INSERT INTO chat_messages (session_id, role, content)
VALUES ($1, $2, $3);

-- name: BumpSessionUpdatedAt :exec
UPDATE chat_sessions
SET updated_at = NOW()
WHERE id = $1;

-- name: DeleteSessionMessages :exec
DELETE FROM chat_messages
WHERE session_id = $1;

-- name: GetUserSessions :many
SELECT id, user_id, title, created_at, updated_at
FROM chat_sessions
WHERE user_id = $1
ORDER BY updated_at DESC;

