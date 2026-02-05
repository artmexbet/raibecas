-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, full_name, role, is_active, created_at, last_login_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
RETURNING id, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, email, full_name, role, is_active, created_at, last_login_at, updated_at
FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, username, email, full_name, role, is_active, created_at, last_login_at, updated_at
FROM users
WHERE email = $1 LIMIT 1;

-- name: ListUsers :many
SELECT id, username, email, full_name, role, is_active, created_at, last_login_at, updated_at
FROM users
WHERE
    (CASE WHEN @search::text != '' THEN
        (username ILIKE '%' || @search || '%' OR email ILIKE '%' || @search || '%' OR full_name ILIKE '%' || @search || '%')
    ELSE TRUE END)
    AND (CASE WHEN sqlc.narg('is_active_filter')::boolean IS NOT NULL THEN is_active = sqlc.narg('is_active_filter') ELSE TRUE END)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*)
FROM users
WHERE
    (CASE WHEN @search::text != '' THEN
        (username ILIKE '%' || @search || '%' OR email ILIKE '%' || @search || '%' OR full_name ILIKE '%' || @search || '%')
    ELSE TRUE END)
    AND (CASE WHEN sqlc.narg('is_active_filter')::boolean IS NOT NULL THEN is_active = sqlc.narg('is_active_filter') ELSE TRUE END);

-- name: UpdateUser :one
UPDATE users
SET
    email = COALESCE(sqlc.narg('email'), email),
    username = COALESCE(sqlc.narg('username'), username),
    full_name = COALESCE(sqlc.narg('full_name'), full_name),
    role = COALESCE(sqlc.narg('role'), role),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING id, username, email, full_name, role, is_active, created_at, last_login_at, updated_at;

-- name: DeleteUser :execresult
UPDATE users
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: CreateRegistrationRequest :one
INSERT INTO registration_requests (username, email, password_hash, status, metadata, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
RETURNING id, created_at, updated_at;

-- name: GetRegistrationRequestByID :one
SELECT id, username, email, password_hash, status, metadata, created_at, updated_at, approved_by, approved_at
FROM registration_requests
WHERE id = $1 LIMIT 1;

-- name: ListRegistrationRequests :many
SELECT id, username, email, status, metadata, created_at, updated_at, approved_by, approved_at
FROM registration_requests
WHERE
    (CASE WHEN @status_filter::text != '' THEN status = @status_filter ELSE TRUE END)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountRegistrationRequests :one
SELECT COUNT(*)
FROM registration_requests
WHERE
    (CASE WHEN @status_filter::text != '' THEN status = @status_filter ELSE TRUE END);

-- name: UpdateRegistrationRequestStatus :execresult
UPDATE registration_requests
SET status = $2, approved_by = $3, approved_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: RejectRegistrationRequest :execresult
UPDATE registration_requests
SET status = 'rejected', approved_by = $2, approved_at = NOW(), metadata = jsonb_set(metadata, '{rejection_reason}', to_jsonb(@reason::text)), updated_at = NOW()
WHERE id = $1 AND status = 'pending';
