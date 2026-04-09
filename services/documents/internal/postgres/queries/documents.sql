-- name: CreateDocument :one
INSERT INTO documents (
    title,
    description,
    category_id,
    publication_date,
    content_path,
    current_version,
    document_type_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDocumentByID :one
SELECT
    documents.*,
    sqlc.embed(categories),
    sqlc.embed(document_types)
FROM documents
LEFT JOIN categories ON documents.category_id = categories.id
LEFT JOIN document_types ON documents.document_type_id = document_types.id
WHERE documents.id = $1;

-- name: ListDocuments :many
SELECT
    d.*,
    sqlc.embed(c),
    sqlc.embed(dt)
FROM documents d
LEFT JOIN categories c ON d.category_id = c.id
LEFT JOIN document_types dt ON d.document_type_id = dt.id
WHERE (
    CASE
        WHEN sqlc.narg('author_id')::uuid IS NOT NULL THEN EXISTS (
            SELECT 1
            FROM document_authors da
            WHERE da.document_id = d.id
              AND da.author_id = sqlc.narg('author_id')::uuid
        )
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('category_id')::int IS NOT NULL THEN d.category_id = sqlc.narg('category_id')::int
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('document_type_id')::int IS NOT NULL THEN d.document_type_id = sqlc.narg('document_type_id')::int
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('tag_id')::int IS NOT NULL THEN EXISTS (
            SELECT 1
            FROM document_tags dtt
            WHERE dtt.document_id = d.id
              AND dtt.tag_id = sqlc.narg('tag_id')::int
        )
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('search')::text IS NOT NULL AND sqlc.narg('search')::text != ''
        THEN (
            to_tsvector('russian', d.title || ' ' || COALESCE(d.description, '') || ' ' || COALESCE(dt.name, '')) @@ plainto_tsquery('russian', sqlc.narg('search')::text)
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
                FROM document_tags dtt
                JOIN tags tt ON tt.id = dtt.tag_id
                WHERE dtt.document_id = d.id
                  AND tt.title ILIKE '%' || sqlc.narg('search')::text || '%'
            )
        )
        ELSE TRUE
    END
)
ORDER BY d.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountDocuments :one
SELECT COUNT(*) FROM documents
WHERE (
    CASE
        WHEN sqlc.narg('author_id')::uuid IS NOT NULL THEN EXISTS (
            SELECT 1
            FROM document_authors da
            WHERE da.document_id = documents.id
              AND da.author_id = sqlc.narg('author_id')::uuid
        )
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('category_id')::int IS NOT NULL THEN category_id = sqlc.narg('category_id')::int
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('document_type_id')::int IS NOT NULL THEN document_type_id = sqlc.narg('document_type_id')::int
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('tag_id')::int IS NOT NULL THEN EXISTS (
            SELECT 1
            FROM document_tags dt
            WHERE dt.document_id = documents.id
              AND dt.tag_id = sqlc.narg('tag_id')::int
        )
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('search')::text IS NOT NULL AND sqlc.narg('search')::text != ''
        THEN (
            to_tsvector('russian', title || ' ' || COALESCE(description, '') || ' ' || COALESCE((SELECT name FROM document_types WHERE id = documents.document_type_id), '')) @@ plainto_tsquery('russian', sqlc.narg('search')::text)
            OR EXISTS (
                SELECT 1
                FROM document_authors da
                JOIN authors a ON a.id = da.author_id
                JOIN authorship_types at ON at.id = da.type_id
                WHERE da.document_id = documents.id
                  AND (
                      a.name ILIKE '%' || sqlc.narg('search')::text || '%'
                      OR at.title ILIKE '%' || sqlc.narg('search')::text || '%'
                  )
            )
            OR EXISTS (
                SELECT 1
                FROM document_tags dt
                JOIN tags t ON t.id = dt.tag_id
                WHERE dt.document_id = documents.id
                  AND t.title ILIKE '%' || sqlc.narg('search')::text || '%'
            )
        )
        ELSE TRUE
    END
);

-- name: UpdateDocument :one
UPDATE documents
SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    category_id = COALESCE(sqlc.narg('category_id'), category_id),
    publication_date = COALESCE(sqlc.narg('publication_date'), publication_date),
    content_path = COALESCE(sqlc.narg('content_path'), content_path),
    current_version = COALESCE(sqlc.narg('current_version'), current_version),
    cover_path = COALESCE(sqlc.narg('cover_path'), cover_path),
    document_type_id = COALESCE(sqlc.narg('document_type_id'), document_type_id),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDocumentIndexed :exec
UPDATE documents
SET indexed = $2
WHERE id = $1;

-- name: DeleteDocument :exec
DELETE FROM documents
WHERE id = $1;

-- name: CreateDocumentVersion :one
INSERT INTO document_versions (
    document_id,
    version,
    content_path,
    changes,
    created_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListDocumentVersions :many
SELECT * FROM document_versions
WHERE document_id = $1
ORDER BY version DESC;

-- name: GetDocumentVersion :one
SELECT * FROM document_versions
WHERE document_id = $1 AND version = $2;

-- name: CreateAuthor :one
INSERT INTO authors (id, name, created_at, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAuthorByID :one
SELECT * FROM authors
WHERE id = $1;

-- name: ListAuthors :many
SELECT * FROM authors
ORDER BY name;

-- name: AddDocumentAuthor :exec
INSERT INTO document_authors (document_id, author_id, type_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: ClearDocumentAuthors :exec
DELETE FROM document_authors
WHERE document_id = $1;

-- name: GetDocumentAuthors :many
SELECT
    da.document_id,
    a.id AS author_id,
    a.name AS author_name,
    a.bio AS author_bio,
    a.created_at AS author_created_at,
    a.updated_at AS author_updated_at,
    at.id AS authorship_type_id,
    at.title AS authorship_type_title,
    at.created_at AS authorship_type_created_at
FROM document_authors da
INNER JOIN authors a ON a.id = da.author_id
INNER JOIN authorship_types at ON at.id = da.type_id
WHERE da.document_id = $1
ORDER BY
    CASE at.title
        WHEN 'автор' THEN 0
        WHEN 'редактор' THEN 1
        WHEN 'рецензент' THEN 2
        ELSE 10
    END,
    a.name;

-- name: CreateCategory :one
INSERT INTO categories (title, created_at)
VALUES ($1, $2)
RETURNING *;

-- name: GetCategoryByID :one
SELECT * FROM categories
WHERE id = $1;

-- name: ListCategories :many
SELECT * FROM categories
ORDER BY title;

-- name: CreateDocumentType :one
INSERT INTO document_types (name)
VALUES ($1)
RETURNING *;

-- name: GetDocumentTypeByID :one
SELECT * FROM document_types
WHERE id = $1;

-- name: ListDocumentTypes :many
SELECT * FROM document_types
ORDER BY name;

-- name: ListAuthorshipTypes :many
SELECT * FROM authorship_types
ORDER BY id;

-- name: CreateTag :one
INSERT INTO tags (title, created_at)
VALUES ($1, $2)
RETURNING *;

-- name: GetTagByID :one
SELECT * FROM tags
WHERE id = $1;

-- name: ListTags :many
SELECT * FROM tags
ORDER BY title;

-- name: AddDocumentTag :exec
INSERT INTO document_tags (document_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveDocumentTag :exec
DELETE FROM document_tags
WHERE document_id = $1 AND tag_id = $2;

-- name: GetDocumentTags :many
SELECT t.* FROM tags t
INNER JOIN document_tags dt ON dt.tag_id = t.id
WHERE dt.document_id = $1
ORDER BY t.title;

-- name: ClearDocumentTags :exec
DELETE FROM document_tags
WHERE document_id = $1;
