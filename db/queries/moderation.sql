-- name: SetStrikes :exec
UPDATE users SET strikes = $2, banned = $3 WHERE id = $1;

-- name: GetStrikes :one
SELECT strikes, banned FROM users WHERE id = $1;

-- name: StaffName :one
SELECT COALESCE(NULLIF(name, ''), email::text, id)::text FROM users WHERE id = $1;

-- name: InsertLog :exec
INSERT INTO moderation_log (at, by_name, action, subject, reason) VALUES ($1, $2, $3, $4, $5);

-- name: ListLog :many
SELECT * FROM moderation_log ORDER BY at DESC, id DESC;

-- name: GetPerson :one
SELECT id, name, area, district FROM users WHERE id = $1 AND deleted_at IS NULL;
