-- name: ListOpenListings :many
-- On sale or reserved, newest first. only_available keeps the live ones of other readers.
-- Listings of deleted accounts, and of readers who blocked the viewer or were blocked by the viewer, stay out.
SELECT sqlc.embed(l), u.name AS seller_name
FROM listings l JOIN users u ON u.id = l.seller_id
WHERE u.deleted_at IS NULL
  AND CASE WHEN @only_available::boolean THEN l.status = 'live' AND l.seller_id <> @viewer::text
           ELSE l.status IN ('live', 'reserved') END
  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.user_id = @viewer::text AND b.blocked_id = l.seller_id)
                                            OR (b.user_id = l.seller_id AND b.blocked_id = @viewer::text))
ORDER BY l.position DESC
LIMIT COALESCE(sqlc.narg('lim')::int, 1000000);

-- name: ListMineListings :many
SELECT sqlc.embed(l), u.name AS seller_name
FROM listings l JOIN users u ON u.id = l.seller_id
WHERE l.seller_id = $1
ORDER BY l.position DESC;

-- name: ListListingsForBook :many
SELECT sqlc.embed(l), u.name AS seller_name
FROM listings l JOIN users u ON u.id = l.seller_id
WHERE l.book_id = @book_id::text AND l.status = 'live' AND l.seller_id <> @viewer::text AND u.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.user_id = @viewer::text AND b.blocked_id = l.seller_id)
                                            OR (b.user_id = l.seller_id AND b.blocked_id = @viewer::text))
ORDER BY l.position DESC;

-- name: GetListing :one
SELECT sqlc.embed(l), u.name AS seller_name
FROM listings l JOIN users u ON u.id = l.seller_id
WHERE l.id = $1;

-- name: LockListing :one
SELECT * FROM listings WHERE id = $1 FOR UPDATE;

-- name: ListPhotosOf :many
SELECT * FROM listing_photos WHERE listing_id = ANY($1::text[]) ORDER BY listing_id, position;

-- name: InsertListing :exec
INSERT INTO listings (id, seller_id, title, price_bdt, condition, flags, is_negotiable, handover, status, rejection_reason,
                      book_id, cover_seed, district, area, category_id, new_price_bdt, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17);

-- name: UpdateListing :exec
UPDATE listings SET title = $2, price_bdt = $3, condition = $4, flags = $5, is_negotiable = $6, handover = $7, status = $8,
                    rejection_reason = $9, book_id = $10, new_price_bdt = $11, note = $12
WHERE id = $1;

-- name: UpsertListingPhoto :exec
INSERT INTO listing_photos (listing_id, slot, position, url, public_id) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (listing_id, slot) DO UPDATE SET url = EXCLUDED.url, public_id = EXCLUDED.public_id, position = EXCLUDED.position;

-- name: DeleteListingPhotosExcept :many
DELETE FROM listing_photos WHERE listing_id = @listing_id::text AND NOT (slot = ANY(@slots::text[])) RETURNING public_id;

-- name: DeleteAllListingPhotosOf :many
DELETE FROM listing_photos WHERE listing_id IN (SELECT id FROM listings WHERE seller_id = $1) RETURNING public_id;

-- name: DeletePhotosOfListing :many
DELETE FROM listing_photos WHERE listing_id = $1 RETURNING public_id;

-- name: SetListingStatus :exec
UPDATE listings SET status = $2, rejection_reason = $3, buyer_id = $4 WHERE id = $1;

-- name: ListingQueue :many
SELECT sqlc.embed(l), u.name AS seller_name, u.strikes AS seller_strikes
FROM listings l JOIN users u ON u.id = l.seller_id
WHERE l.status = 'inReview' AND u.deleted_at IS NULL
ORDER BY l.position DESC;

-- name: GetReader :one
SELECT id, name, area, district, member_since, sold_before FROM users WHERE id = $1 AND role = 'reader' AND deleted_at IS NULL;

-- name: CountSoldBy :one
SELECT count(*)::int FROM listings WHERE seller_id = $1 AND status = 'sold';

-- name: ListRatingsTo :many
SELECT r.stars, r.at, r.comment, u.name AS from_name
FROM ratings r JOIN users u ON u.id = r.from_id
WHERE r.to_id = $1 ORDER BY r.at DESC, r.id DESC;

-- name: LiveListingsOf :many
SELECT sqlc.embed(l), u.name AS seller_name
FROM listings l JOIN users u ON u.id = l.seller_id
WHERE l.seller_id = $1 AND l.status = 'live'
ORDER BY l.position DESC;

-- name: SeedListing :exec
INSERT INTO listings (id, position, seller_id, title, price_bdt, condition, flags, is_negotiable, handover, status, rejection_reason,
                      book_id, cover_seed, district, area, category_id, new_price_bdt, note, buyer_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
ON CONFLICT (id) DO NOTHING;

-- name: SeedListingPhoto :exec
INSERT INTO listing_photos (listing_id, slot, position) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;

-- name: SeedRating :exec
INSERT INTO ratings (from_id, to_id, listing_id, stars, comment, at)
SELECT @from_id::text, @to_id::text, sqlc.narg('listing_id')::text, @stars::int, sqlc.narg('comment')::text, @at::timestamptz
WHERE NOT EXISTS (SELECT 1 FROM ratings WHERE from_id = @from_id::text AND to_id = @to_id::text AND stars = @stars::int
                  AND COALESCE(comment, '') = COALESCE(sqlc.narg('comment')::text, ''));

-- name: SeedSoldBefore :exec
UPDATE users SET sold_before = $2 WHERE id = $1;

-- name: ListRatingsForListings :many
SELECT listing_id, from_id, stars FROM ratings WHERE listing_id = ANY($1::text[]);

-- name: InsertRating :exec
INSERT INTO ratings (from_id, to_id, listing_id, stars, comment, at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListingCandidates :many
-- Copies that may match a book request: not sold, of readers other than excluded_seller.
SELECT l.id, l.title, l.book_id, l.seller_id, l.status, u.name AS seller_name
FROM listings l JOIN users u ON u.id = l.seller_id
WHERE l.status <> 'sold' AND u.deleted_at IS NULL AND l.seller_id <> @excluded_seller::text
  AND (NOT @only_live::boolean OR l.status = 'live')
ORDER BY l.position DESC;

-- name: ListingsOfSeller :many
SELECT l.id, l.title, l.book_id, l.seller_id, l.status, u.name AS seller_name
FROM listings l JOIN users u ON u.id = l.seller_id
WHERE l.seller_id = $1 AND l.status <> 'sold'
ORDER BY l.position DESC;

-- name: CountLiveListingsOf :one
SELECT count(*)::integer FROM listings WHERE seller_id = $1 AND status = 'live';
