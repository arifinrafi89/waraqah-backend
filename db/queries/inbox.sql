-- name: LockThread :one
SELECT * FROM threads WHERE id = $1 FOR UPDATE;

-- name: GetThread :one
SELECT * FROM threads WHERE id = $1;

-- name: FindThreadOf :one
SELECT * FROM threads WHERE listing_id = $1 AND buyer_id = $2;

-- name: InsertThread :exec
INSERT INTO threads (id, listing_id, buyer_id, seller_id) VALUES ($1, $2, $3, $4) ON CONFLICT (listing_id, buyer_id) DO NOTHING;

-- name: ListThreadsOf :many
SELECT * FROM threads WHERE buyer_id = $1 OR seller_id = $1;

-- name: ListThreadsAbout :many
SELECT * FROM threads WHERE listing_id = $1 ORDER BY created_at, id;

-- name: ListMessagesOf :many
SELECT m.id, m.position, m.thread_id, m.author_id, m.at, m.text, m.event, m.amount_bdt,
       o.id AS offer_id, o.amount_bdt AS offer_amount_bdt, o.handover AS offer_handover, o.status AS offer_status
FROM messages m LEFT JOIN offers o ON o.id = m.offer_id
WHERE m.thread_id = ANY($1::text[])
ORDER BY m.thread_id, m.position;

-- name: InsertMessage :one
INSERT INTO messages (id, thread_id, author_id, at, text, offer_id, event, amount_bdt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING position;

-- name: InsertOffer :exec
INSERT INTO offers (id, thread_id, amount_bdt, handover) VALUES ($1, $2, $3, $4);

-- name: PendingOffer :one
SELECT * FROM offers WHERE thread_id = $1 AND status = 'pending' ORDER BY position DESC LIMIT 1;

-- name: SetOfferStatus :exec
UPDATE offers SET status = $2 WHERE id = $1;

-- name: ClosePendingOffers :exec
UPDATE offers SET status = 'closed' WHERE thread_id = $1 AND status = 'pending';

-- name: ReadUpBuyer :exec
UPDATE threads SET buyer_read = COALESCE((SELECT max(position) FROM messages WHERE thread_id = threads.id), 0) WHERE threads.id = $1;

-- name: ReadUpSeller :exec
UPDATE threads SET seller_read = COALESCE((SELECT max(position) FROM messages WHERE thread_id = threads.id), 0) WHERE threads.id = $1;

-- name: MarkBotReplied :execrows
UPDATE threads SET bot_replied = true WHERE id = $1 AND NOT bot_replied;

-- name: GetMessage :one
SELECT * FROM messages WHERE id = $1;

-- name: DeleteMessage :one
DELETE FROM messages WHERE id = $1 RETURNING thread_id;

-- name: ListUserNames :many
SELECT id, name FROM users WHERE id = ANY($1::text[]);

-- name: SeedThread :exec
INSERT INTO threads (id, listing_id, buyer_id, seller_id, bot_replied, created_at) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO NOTHING;

-- name: SeedOffer :exec
INSERT INTO offers (id, thread_id, amount_bdt, handover, status) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING;

-- name: SeedMessage :exec
INSERT INTO messages (id, thread_id, author_id, at, text, offer_id, event, amount_bdt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (id) DO NOTHING;

-- name: SeedSetReadBuyer :exec
UPDATE threads SET buyer_read = $2 WHERE id = $1;

-- name: SeedSetReadSeller :exec
UPDATE threads SET seller_read = $2 WHERE id = $1;

-- name: ListMessagePositions :many
SELECT position FROM messages WHERE thread_id = $1 ORDER BY position;
