-- name: LockUser :one
SELECT id FROM users WHERE id = $1 FOR UPDATE;

-- name: ListCoupons :many
SELECT * FROM coupons ORDER BY position DESC;

-- name: GetCoupon :one
SELECT * FROM coupons WHERE code = $1;

-- name: InsertCoupon :execrows
INSERT INTO coupons (code, kind, value, min_order_bdt, max_discount_bdt, expires_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (code) DO NOTHING;

-- name: InsertOrder :exec
INSERT INTO orders (number, user_id, status, placed_at, address_label, address_line, payment, subtotal_bdt, delivery_fee_bdt,
                    discount_bdt, total_bdt, needs_delivery, points_used, points_earned, gift, gift_wrap_bdt, is_donation,
                    donate_place_id, wallet_used_bdt, refunded_bdt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20);

-- name: InsertOrderLine :exec
INSERT INTO order_lines (order_number, position, book_id, edition_id, title, author, quantity, unit_price_bdt, format, language, cover_seed)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: InsertOrderHistory :exec
INSERT INTO order_history (order_number, position, status, at)
VALUES ($1, (SELECT COALESCE(max(position), 0) + 1 FROM order_history WHERE order_number = $1), $2, $3);

-- name: ListOrdersOf :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY placed_at DESC, number DESC;

-- name: ListAllOrders :many
SELECT * FROM orders ORDER BY placed_at DESC, number DESC;

-- name: GetOrder :one
SELECT * FROM orders WHERE number = $1 FOR UPDATE;

-- name: ListLinesOf :many
SELECT * FROM order_lines WHERE order_number = ANY($1::text[]) ORDER BY order_number, position;

-- name: ListHistoryOf :many
SELECT * FROM order_history WHERE order_number = ANY($1::text[]) ORDER BY order_number, position;

-- name: ListReturnsOf :many
SELECT * FROM order_returns WHERE order_number = ANY($1::text[]);

-- name: SetOrderStatus :exec
UPDATE orders SET status = $2 WHERE number = $1;

-- name: AddOrderRefund :exec
UPDATE orders SET refunded_bdt = refunded_bdt + $2 WHERE number = $1;

-- name: InsertReturn :exec
INSERT INTO order_returns (order_number, reason, status, requested_at, note, photos)
VALUES ($1, $2, 'requested', $3, $4, $5);

-- name: DecideReturn :exec
UPDATE order_returns SET status = $2, decided_at = $3 WHERE order_number = $1;

-- name: WalletBalance :one
SELECT COALESCE(sum(amount_bdt), 0)::bigint FROM wallet_entries WHERE user_id = $1;

-- name: AddWalletEntry :exec
INSERT INTO wallet_entries (user_id, amount_bdt, reason, order_number, note, at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListWalletEntries :many
SELECT * FROM wallet_entries WHERE user_id = $1 ORDER BY at DESC, id DESC;

-- name: PointsBalance :one
SELECT COALESCE(sum(points), 0)::bigint FROM points_entries WHERE user_id = $1;

-- name: AddPointsEntry :exec
INSERT INTO points_entries (user_id, points, reason, order_number, at) VALUES ($1, $2, $3, $4, $5);

-- name: ListPointsEntries :many
SELECT * FROM points_entries WHERE user_id = $1 ORDER BY at DESC, id DESC;

-- name: TakeStock :exec
UPDATE editions SET stock = GREATEST(stock - $2, 0) WHERE id = $1 AND format <> 'ebook' AND stock > 0;

-- name: RestockEdition :exec
UPDATE editions SET stock = stock + $2 WHERE id = $1 AND format <> 'ebook';

-- name: AddSale :exec
INSERT INTO sales_by_month (edition_id, month, copies) VALUES ($1, $2, $3)
ON CONFLICT (edition_id, month) DO UPDATE SET copies = sales_by_month.copies + EXCLUDED.copies;

-- name: DeliveredLinesOf :many
SELECT l.book_id, l.edition_id, o.number, h.at AS delivered_at
FROM orders o
JOIN order_lines l ON l.order_number = o.number
JOIN order_history h ON h.order_number = o.number AND h.status = 'delivered'
WHERE o.user_id = $1 AND o.status = 'delivered'
ORDER BY h.at, o.number, l.position;
