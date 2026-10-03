-- name: UpsertSeedCoupon :exec
INSERT INTO coupons (code, kind, value, min_order_bdt, max_discount_bdt, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (code) DO UPDATE
SET kind = EXCLUDED.kind, value = EXCLUDED.value, min_order_bdt = EXCLUDED.min_order_bdt,
    max_discount_bdt = EXCLUDED.max_discount_bdt, expires_at = EXCLUDED.expires_at;

-- name: SeedOrderExists :one
SELECT EXISTS (SELECT 1 FROM orders WHERE number = $1);

-- name: SeedOrderHistory :exec
INSERT INTO order_history (order_number, position, status, at) VALUES ($1, $2, $3, $4);

-- name: SeedWalletEntryExists :one
SELECT EXISTS (SELECT 1 FROM wallet_entries WHERE user_id = $1 AND reason = $2 AND COALESCE(note, '') = $3 AND order_number IS NULL);

-- name: SeedPointsEntryExists :one
SELECT EXISTS (SELECT 1 FROM points_entries WHERE user_id = $1 AND reason = $2 AND COALESCE(order_number, '') = $3);
