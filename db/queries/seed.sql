-- name: UpsertSeedUser :exec
INSERT INTO users (id, email, name, role, password_hash, member_since)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email, name = EXCLUDED.name, role = EXCLUDED.role,
    password_hash = COALESCE(users.password_hash, EXCLUDED.password_hash);

-- name: SetUserRoleByEmail :execrows
UPDATE users SET role = $2 WHERE email = $1 AND deleted_at IS NULL;

-- name: UpsertSeedAddress :exec
INSERT INTO addresses (id, user_id, label, recipient, phone, line, upazila, district, division, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE
SET label = EXCLUDED.label, recipient = EXCLUDED.recipient, phone = EXCLUDED.phone, line = EXCLUDED.line,
    upazila = EXCLUDED.upazila, district = EXCLUDED.district, division = EXCLUDED.division;

-- name: UpsertSeedNotification :exec
INSERT INTO notifications (id, user_id, kind, params, target, read_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET kind = EXCLUDED.kind, params = EXCLUDED.params, target = EXCLUDED.target;
