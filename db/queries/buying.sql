-- name: ListCartLines :many
SELECT * FROM cart_lines WHERE user_id = $1 ORDER BY position;

-- name: GetCartLine :one
SELECT * FROM cart_lines WHERE user_id = $1 AND kind = $2 AND item_id = $3 FOR UPDATE;

-- name: InsertCartLine :exec
INSERT INTO cart_lines (user_id, kind, item_id, quantity, added_at) VALUES ($1, $2, $3, $4, $5);

-- name: SetCartQuantity :exec
UPDATE cart_lines SET quantity = $4 WHERE user_id = $1 AND kind = $2 AND item_id = $3;

-- name: DeleteCartLine :exec
DELETE FROM cart_lines WHERE user_id = $1 AND kind = $2 AND item_id = $3;

-- name: ClearCart :exec
DELETE FROM cart_lines WHERE user_id = $1;

-- name: ListFlashItems :many
SELECT * FROM flash_sale_items ORDER BY position;

-- name: ListBundles :many
SELECT * FROM bundles ORDER BY position;

-- name: ListPreorders :many
SELECT * FROM preorders ORDER BY position;

-- name: ListWishlistBookIDs :many
SELECT book_id FROM wishlist_items WHERE user_id = $1 ORDER BY added_at DESC, book_id;

-- name: SaveWishlistItem :exec
INSERT INTO wishlist_items (user_id, book_id, added_at) VALUES ($1, $2, $3)
ON CONFLICT (user_id, book_id) DO UPDATE SET added_at = EXCLUDED.added_at;

-- name: RemoveWishlistItem :exec
DELETE FROM wishlist_items WHERE user_id = $1 AND book_id = $2;

-- name: GetShareByUser :one
SELECT * FROM wishlist_shares WHERE user_id = $1;

-- name: GetShareByID :one
SELECT * FROM wishlist_shares WHERE id = $1;

-- name: UpsertShare :exec
INSERT INTO wishlist_shares (id, user_id, owner_name) VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE SET owner_name = EXCLUDED.owner_name;

-- name: ListAlerts :many
SELECT * FROM alerts WHERE user_id = $1 ORDER BY position;

-- name: DeleteAlertsOfKind :exec
DELETE FROM alerts WHERE user_id = $1 AND kind = $2 AND edition_id = $3;

-- name: InsertAlert :exec
INSERT INTO alerts (id, user_id, kind, book_id, edition_id, target_price_bdt, fired_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: DeleteAlert :exec
DELETE FROM alerts WHERE user_id = $1 AND id = $2;

-- name: ListUnfiredAlerts :many
SELECT * FROM alerts WHERE fired_at IS NULL ORDER BY position;

-- name: MarkAlertFired :execrows
UPDATE alerts SET fired_at = $2 WHERE id = $1 AND fired_at IS NULL;
