-- name: CreateNote :one
INSERT INTO notes (
    id,
    user_id,
    title,
    content,
    document_id,
    bookmark_id,
    position_in_document
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetNoteByIDForUser :one
SELECT *
FROM notes
WHERE id = $1 AND user_id = $2;

-- name: ListNotesByUser :many
SELECT n.*
FROM notes n
LEFT JOIN documents d ON d.id = n.document_id
WHERE n.user_id = $1
  AND (
    CASE
      WHEN sqlc.narg('search')::text IS NOT NULL AND sqlc.narg('search')::text != ''
      THEN (
        n.title ILIKE '%' || sqlc.narg('search')::text || '%'
        OR n.content ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(d.title, '') ILIKE '%' || sqlc.narg('search')::text || '%'
      )
      ELSE TRUE
    END
  )
  AND (
    CASE
      WHEN sqlc.narg('document_id')::uuid IS NOT NULL THEN n.document_id = sqlc.narg('document_id')::uuid
      ELSE TRUE
    END
  )
  AND (
    CASE
      WHEN sqlc.narg('bookmark_id')::uuid IS NOT NULL THEN n.bookmark_id = sqlc.narg('bookmark_id')::uuid
      ELSE TRUE
    END
  )
ORDER BY n.created_at DESC, n.id DESC
LIMIT $2 OFFSET $3;

-- name: CountNotesByUser :one
SELECT COUNT(*)
FROM notes n
LEFT JOIN documents d ON d.id = n.document_id
WHERE n.user_id = $1
  AND (
    CASE
      WHEN sqlc.narg('search')::text IS NOT NULL AND sqlc.narg('search')::text != ''
      THEN (
        n.title ILIKE '%' || sqlc.narg('search')::text || '%'
        OR n.content ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(d.title, '') ILIKE '%' || sqlc.narg('search')::text || '%'
      )
      ELSE TRUE
    END
  )
  AND (
    CASE
      WHEN sqlc.narg('document_id')::uuid IS NOT NULL THEN n.document_id = sqlc.narg('document_id')::uuid
      ELSE TRUE
    END
  )
  AND (
    CASE
      WHEN sqlc.narg('bookmark_id')::uuid IS NOT NULL THEN n.bookmark_id = sqlc.narg('bookmark_id')::uuid
      ELSE TRUE
    END
  );

-- name: UpdateNote :one
UPDATE notes
SET
    title = COALESCE(sqlc.narg('title')::varchar, title),
    content = COALESCE(sqlc.narg('content')::text, content),
    document_id = CASE WHEN sqlc.arg('clear_document_id')::bool THEN NULL ELSE COALESCE(sqlc.narg('document_id')::uuid, document_id) END,
    bookmark_id = CASE WHEN sqlc.arg('clear_bookmark_id')::bool THEN NULL ELSE COALESCE(sqlc.narg('bookmark_id')::uuid, bookmark_id) END,
    position_in_document = CASE WHEN sqlc.arg('clear_position')::bool THEN NULL ELSE COALESCE(sqlc.narg('position_in_document')::text, position_in_document) END,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteNote :execrows
DELETE FROM notes
WHERE id = $1 AND user_id = $2;
