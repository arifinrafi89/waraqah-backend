-- name: ListBlocks :many
SELECT b.blocked_id, b.at, u.name FROM blocks b JOIN users u ON u.id = b.blocked_id
WHERE b.user_id = $1 ORDER BY b.at DESC, b.blocked_id;

-- name: AddBlock :exec
INSERT INTO blocks (user_id, blocked_id, at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;

-- name: RemoveBlock :exec
DELETE FROM blocks WHERE user_id = $1 AND blocked_id = $2;

-- name: IsBlockedEither :one
SELECT EXISTS (SELECT 1 FROM blocks WHERE (user_id = $1 AND blocked_id = $2) OR (user_id = $2 AND blocked_id = $1));

-- name: FindOpenReport :one
SELECT * FROM reports WHERE reporter_id = $1 AND kind = $2 AND target_id = $3 AND status = 'open' ORDER BY position LIMIT 1;

-- name: InsertReport :exec
INSERT INTO reports (id, kind, target_id, reason, note, reporter_id, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, 'open', $7);

-- name: ListOpenReports :many
SELECT * FROM reports WHERE status = 'open' ORDER BY created_at, position;

-- name: GetOpenReport :one
SELECT * FROM reports WHERE id = $1 AND status = 'open' FOR UPDATE;

-- name: CloseReportsAbout :exec
UPDATE reports SET status = $3 WHERE kind = $1 AND target_id = $2 AND status = 'open';

-- name: SeedReport :exec
INSERT INTO reports (id, kind, target_id, reason, note, reporter_id, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, 'open', $7) ON CONFLICT (id) DO NOTHING;

-- name: ListBlockedEither :many
-- Everyone the reader blocked or who blocked the reader.
SELECT b.blocked_id FROM blocks b WHERE b.user_id = sqlc.arg(reader)::text
UNION
SELECT b.user_id FROM blocks b WHERE b.blocked_id = sqlc.arg(reader)::text;
