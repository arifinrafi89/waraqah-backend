-- name: NextSellBackNumber :one
SELECT nextval('sell_back_seq')::bigint;

-- name: InsertSellBack :exec
INSERT INTO sell_backs (id, user_id, book_id, title, author, new_price_bdt, cover_seed, condition, flags, quote_bdt, pickup_address, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: SeedSellBack :exec
INSERT INTO sell_backs (id, user_id, book_id, title, author, new_price_bdt, cover_seed, condition, flags, quote_bdt, status, pickup_address,
                        created_at, graded_condition, paid_bdt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (id) DO NOTHING;

-- name: ListSellBacksOf :many
SELECT * FROM sell_backs WHERE user_id = $1 ORDER BY created_at DESC, id DESC;

-- name: ListSellBackQueue :many
SELECT sqlc.embed(sell_backs), u.name AS reader_name
FROM sell_backs JOIN users u ON u.id = sell_backs.user_id
WHERE sell_backs.status = 'pickedUp' ORDER BY sell_backs.created_at, sell_backs.id;

-- name: CountSellBackQueue :one
SELECT count(*)::bigint FROM sell_backs WHERE status = 'pickedUp';

-- name: LockSellBack :one
SELECT * FROM sell_backs WHERE id = $1 FOR UPDATE;

-- name: GradeSellBack :exec
UPDATE sell_backs SET status = $2, graded_condition = $3, paid_bdt = $4 WHERE id = $1;

-- name: PickUpDue :many
-- The courier job: every pickup booked before due is collected (catches up after a sleep).
UPDATE sell_backs SET status = 'pickedUp' WHERE status = 'scheduled' AND created_at <= $1 RETURNING id;

-- name: InsertCertified :exec
INSERT INTO certified_used (id, book_id, condition, price_bdt, created_at) VALUES ($1, $2, $3, $4, $5);

-- name: SeedCertified :exec
INSERT INTO certified_used (id, book_id, condition, price_bdt, created_at) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO NOTHING;

-- name: CountCertifiedOf :one
SELECT count(*)::bigint FROM certified_used WHERE book_id = $1;

-- name: CheapestCertified :one
SELECT * FROM certified_used WHERE book_id = $1 AND sold_at IS NULL ORDER BY price_bdt, id LIMIT 1;

-- name: GetCertified :one
SELECT * FROM certified_used WHERE id = $1 AND sold_at IS NULL;

-- name: SellCertified :execrows
UPDATE certified_used SET sold_at = $2 WHERE id = $1 AND sold_at IS NULL;
