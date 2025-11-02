-- name: CreateRegistrationRequest :one
INSERT INTO registration_requests (username, email, password, metadata)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRegistrationRequestByID :one
SELECT *
FROM registration_requests
WHERE id = $1
LIMIT 1;

-- name: UpdateRegistrationStatus :exec
UPDATE registration_requests
SET status      = $1,
    approved_by = $2,
    approved_at = CASE WHEN $1 = 'approved' THEN NOW() END,
    updated_at  = NOW()
WHERE id = $3;

-- name: RegistrationExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM registration_requests WHERE email = $1 AND status = 'pending');

-- name: RegistrationExistsByUsername :one
SELECT EXISTS(SELECT 1 FROM registration_requests WHERE username = $1 AND status = 'pending');

-- name: ListPendingRegistrations :many
SELECT *
FROM registration_requests
WHERE status = 'pending'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
