-- name: ListShelf :many
SELECT * FROM shelf_entries WHERE user_id = $1 ORDER BY added_at DESC, position;

-- name: GetShelfEntry :one
SELECT * FROM shelf_entries WHERE user_id = $1 AND book_id = $2;

-- name: PutShelfEntry :exec
INSERT INTO shelf_entries (user_id, book_id, shelf, added_at, finished_at, progress, pages_read, total_pages)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id, book_id) DO UPDATE SET shelf = excluded.shelf, added_at = excluded.added_at, finished_at = excluded.finished_at,
    progress = excluded.progress, pages_read = excluded.pages_read, total_pages = excluded.total_pages;

-- name: AddShelfEntryIfAbsent :exec
INSERT INTO shelf_entries (user_id, book_id, shelf, added_at) VALUES ($1, $2, 'wantToRead', $3) ON CONFLICT DO NOTHING;

-- name: DeleteShelfEntry :exec
DELETE FROM shelf_entries WHERE user_id = $1 AND book_id = $2;

-- name: MarkShelfSynced :execrows
INSERT INTO shelf_order_sync (user_id, book_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: AddReadingDay :exec
INSERT INTO reading_days (user_id, day) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListReadingDays :many
SELECT day FROM reading_days WHERE user_id = $1 AND day >= $2 ORDER BY day DESC;

-- name: GetReadingGoal :one
SELECT goal FROM reading_goals WHERE user_id = $1 AND year = $2;

-- name: SetReadingGoal :exec
INSERT INTO reading_goals (user_id, year, goal) VALUES ($1, $2, $3) ON CONFLICT (user_id, year) DO UPDATE SET goal = excluded.goal;

-- name: SeedShelfEntry :exec
INSERT INTO shelf_entries (user_id, book_id, shelf, added_at, finished_at, progress, pages_read, total_pages)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT DO NOTHING;

-- name: SeedReadingGoal :exec
INSERT INTO reading_goals (user_id, year, goal) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;
