-- name: InsertSale :exec
INSERT INTO handled_sales (id, listing_id, buyer_id, seller_id, price_bdt, fee_bdt, delivery_bdt, method, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9);

-- name: GetSale :one
SELECT * FROM handled_sales WHERE id = $1;

-- name: LockSale :one
SELECT * FROM handled_sales WHERE id = $1 FOR UPDATE;

-- name: ListSalesOf :many
SELECT * FROM handled_sales WHERE buyer_id = $1 OR seller_id = $1 ORDER BY created_at DESC, id DESC;

-- name: SetSaleStatus :exec
UPDATE handled_sales SET status = $2, updated_at = $3 WHERE id = $1;

-- name: SetSaleDispute :exec
UPDATE handled_sales SET status = 'disputed', dispute_reason = $2, dispute_note = $3, dispute_photos = $4, updated_at = $5
WHERE id = $1;

-- name: ListDisputedSales :many
SELECT * FROM handled_sales WHERE status = 'disputed' ORDER BY created_at, id;

-- name: CountDisputedSales :one
SELECT count(*)::bigint FROM handled_sales WHERE status = 'disputed';

-- name: SellerEarnings :one
SELECT coalesce(sum(price_bdt - fee_bdt) FILTER (WHERE status IN ('paid', 'sent', 'disputed')), 0)::bigint AS held_bdt,
       coalesce(sum(price_bdt - fee_bdt) FILTER (WHERE status IN ('completed', 'released')), 0)::bigint AS earned_bdt
FROM handled_sales WHERE seller_id = $1;

-- name: PaidOut :one
SELECT coalesce(sum(amount_bdt), 0)::bigint FROM payouts WHERE user_id = $1;

-- name: ListPayouts :many
SELECT amount_bdt, at FROM payouts WHERE user_id = $1 ORDER BY at DESC, id DESC;

-- name: InsertPayout :exec
INSERT INTO payouts (user_id, amount_bdt, at) VALUES ($1, $2, $3);

-- name: SendDemoSales :many
-- The demo bot: demo sellers (ids "p-...") hand paid books to the courier once they are old enough.
UPDATE handled_sales SET status = 'sent', updated_at = sqlc.arg(now)
WHERE status = 'paid' AND seller_id LIKE 'p-%' AND created_at <= sqlc.arg(due)
RETURNING *;

-- name: SeedSale :exec
INSERT INTO handled_sales (id, listing_id, buyer_id, seller_id, price_bdt, fee_bdt, delivery_bdt, method, status, created_at, updated_at,
                           dispute_reason, dispute_note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10, $11, $12)
ON CONFLICT (id) DO NOTHING;

-- name: SeedPayout :exec
INSERT INTO payouts (user_id, amount_bdt, at)
SELECT $1, $2, $3 WHERE NOT EXISTS (SELECT 1 FROM payouts WHERE user_id = $1);

-- name: SeedListingBuyer :exec
UPDATE listings SET status = $2, buyer_id = $3 WHERE id = $1 AND buyer_id IS NULL;
