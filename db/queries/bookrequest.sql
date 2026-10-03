-- name: InsertBookRequest :exec
INSERT INTO book_requests (id, requester_id, title, author, book_id, max_price_bdt, note, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListBookRequestsOf :many
SELECT * FROM book_requests WHERE requester_id = $1 ORDER BY position DESC;

-- name: CloseBookRequest :execrows
UPDATE book_requests SET is_open = false WHERE id = $1 AND requester_id = $2;

-- name: ListOpenBookRequestsOfOthers :many
SELECT r.*, u.name AS requester_name FROM book_requests r JOIN users u ON u.id = r.requester_id
WHERE r.is_open AND r.requester_id <> $1 AND u.deleted_at IS NULL ORDER BY r.position DESC;

-- name: ListOpenBookRequests :many
SELECT * FROM book_requests WHERE is_open ORDER BY position;

-- name: SeedBookRequest :exec
INSERT INTO book_requests (id, requester_id, title, author, book_id, max_price_bdt, note, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (id) DO NOTHING;
