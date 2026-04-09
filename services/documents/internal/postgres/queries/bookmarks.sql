-- name: CreateBookmark :one
INSERT INTO document_bookmarks (
    id,
    user_id,
    document_id,
    kind,
    quote_text,
    quote_context,
    page_label
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetBookmarkByIDForUser :one
SELECT *
FROM document_bookmarks
WHERE id = $1 AND user_id = $2;

-- name: GetPublicationBookmarkByUserAndDocument :one
SELECT *
FROM document_bookmarks
WHERE user_id = $1 AND document_id = $2 AND kind = 'publication'
LIMIT 1;

-- name: ListBookmarksByUser :many
SELECT b.*
FROM document_bookmarks b
JOIN documents d ON d.id = b.document_id
LEFT JOIN categories c ON c.id = d.category_id
LEFT JOIN document_types dtp ON dtp.id = d.document_type_id
WHERE b.user_id = $1
  AND (
    CASE
      WHEN sqlc.narg('kind')::text IS NOT NULL AND sqlc.narg('kind')::text != '' THEN b.kind = sqlc.narg('kind')::text
      ELSE TRUE
    END
  )
  AND (
    CASE
      WHEN sqlc.narg('search')::text IS NOT NULL AND sqlc.narg('search')::text != ''
      THEN (
        d.title ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(d.description, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(c.title, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(dtp.name, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(b.quote_text, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(b.quote_context, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR EXISTS (
          SELECT 1
          FROM document_authors da
          JOIN authors sa ON sa.id = da.author_id
          JOIN authorship_types sat ON sat.id = da.type_id
          WHERE da.document_id = d.id
            AND (
              sa.name ILIKE '%' || sqlc.narg('search')::text || '%'
              OR sat.title ILIKE '%' || sqlc.narg('search')::text || '%'
            )
        )
        OR EXISTS (
          SELECT 1
          FROM document_tags dt
          JOIN tags t ON t.id = dt.tag_id
          WHERE dt.document_id = d.id
            AND t.title ILIKE '%' || sqlc.narg('search')::text || '%'
        )
      )
      ELSE TRUE
    END
  )
ORDER BY b.created_at DESC, b.id DESC
LIMIT $2 OFFSET $3;

-- name: CountBookmarksByUser :one
SELECT COUNT(*)
FROM document_bookmarks b
JOIN documents d ON d.id = b.document_id
LEFT JOIN categories c ON c.id = d.category_id
LEFT JOIN document_types dtp ON dtp.id = d.document_type_id
WHERE b.user_id = $1
  AND (
    CASE
      WHEN sqlc.narg('kind')::text IS NOT NULL AND sqlc.narg('kind')::text != '' THEN b.kind = sqlc.narg('kind')::text
      ELSE TRUE
    END
  )
  AND (
    CASE
      WHEN sqlc.narg('search')::text IS NOT NULL AND sqlc.narg('search')::text != ''
      THEN (
        d.title ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(d.description, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(c.title, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(dtp.name, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(b.quote_text, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR COALESCE(b.quote_context, '') ILIKE '%' || sqlc.narg('search')::text || '%'
        OR EXISTS (
          SELECT 1
          FROM document_authors da
          JOIN authors sa ON sa.id = da.author_id
          JOIN authorship_types sat ON sat.id = da.type_id
          WHERE da.document_id = d.id
            AND (
              sa.name ILIKE '%' || sqlc.narg('search')::text || '%'
              OR sat.title ILIKE '%' || sqlc.narg('search')::text || '%'
            )
        )
        OR EXISTS (
          SELECT 1
          FROM document_tags dt
          JOIN tags t ON t.id = dt.tag_id
          WHERE dt.document_id = d.id
            AND t.title ILIKE '%' || sqlc.narg('search')::text || '%'
        )
      )
      ELSE TRUE
    END
  );

-- name: DeleteBookmark :execrows
DELETE FROM document_bookmarks
WHERE id = $1 AND user_id = $2;

