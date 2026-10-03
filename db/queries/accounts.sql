-- name: CreateUser :one
INSERT INTO users (id, email, name, role, password_hash, photo_url, member_since, created_at)
VALUES ($1, $2, $3, 'reader', $4, $5, $6, $6)
RETURNING *;

-- name: SetUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: UpsertPendingSignup :exec
INSERT INTO pending_signups (contact, name, password_hash, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (contact) DO UPDATE
SET name = EXCLUDED.name, password_hash = EXCLUDED.password_hash, created_at = EXCLUDED.created_at;

-- name: GetPendingSignup :one
SELECT * FROM pending_signups WHERE contact = $1;

-- name: DeletePendingSignup :exec
DELETE FROM pending_signups WHERE contact = $1;
