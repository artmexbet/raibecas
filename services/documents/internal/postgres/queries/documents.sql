-- name: CreateDocument :one
INSERT INTO documents (
    title,
    description,
    author_id,
    category_id,
    publication_date,
    content_path,
    current_version
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDocumentByID :one
SELECT
    documents.*,
    sqlc.embed(authors),
    sqlc.embed(categories)
FROM documents
LEFT JOIN authors ON documents.author_id = authors.id
LEFT JOIN categories ON documents.category_id = categories.id
WHERE documents.id = $1;

-- name: ListDocuments :many
SELECT
    d.*,
    sqlc.embed(a),
    sqlc.embed(c)
FROM documents d
LEFT JOIN authors a ON d.author_id = a.id
LEFT JOIN categories c ON d.category_id = c.id
WHERE (
    CASE
        WHEN sqlc.narg('author_id')::uuid IS NOT NULL THEN d.author_id = sqlc.narg('author_id')::uuid
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('category_id')::int IS NOT NULL THEN d.category_id = sqlc.narg('category_id')::int
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('search')::text IS NOT NULL AND sqlc.narg('search')::text != ''
        THEN to_tsvector('russian', d.title || ' ' || COALESCE(d.description, '')) @@ plainto_tsquery('russian', sqlc.narg('search')::text)
        ELSE TRUE
    END
)
ORDER BY d.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountDocuments :one
SELECT COUNT(*) FROM documents
WHERE (
    CASE
        WHEN sqlc.narg('author_id')::uuid IS NOT NULL THEN author_id = sqlc.narg('author_id')::uuid
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('category_id')::int IS NOT NULL THEN category_id = sqlc.narg('category_id')::int
        ELSE TRUE
    END
) AND (
    CASE
        WHEN sqlc.narg('search')::text IS NOT NULL AND sqlc.narg('search')::text != ''
        THEN to_tsvector('russian', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('russian', sqlc.narg('search')::text)
        ELSE TRUE
    END
);

-- name: UpdateDocument :one
UPDATE documents
SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    author_id = COALESCE(sqlc.narg('author_id'), author_id),
    category_id = COALESCE(sqlc.narg('category_id'), category_id),
    publication_date = COALESCE(sqlc.narg('publication_date'), publication_date),
    content_path = COALESCE(sqlc.narg('content_path'), content_path),
    current_version = COALESCE(sqlc.narg('current_version'), current_version),
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
