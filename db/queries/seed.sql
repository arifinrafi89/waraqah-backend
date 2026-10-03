-- name: UpsertSeedUser :exec
INSERT INTO users (id, email, name, role, password_hash, member_since)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email, name = EXCLUDED.name, role = EXCLUDED.role,
    password_hash = COALESCE(users.password_hash, EXCLUDED.password_hash);

-- name: SetUserRoleByEmail :execrows
UPDATE users SET role = $2 WHERE email = $1 AND deleted_at IS NULL;
