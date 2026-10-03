-- name: ListBanners :many
SELECT * FROM banners ORDER BY position, id;

-- name: GetBanner :one
SELECT * FROM banners WHERE id = $1 FOR UPDATE;

-- name: NextBannerPosition :one
SELECT (COALESCE(max(position), 0) + 1)::int FROM banners;

-- name: InsertBanner :exec
INSERT INTO banners (id, position, title_en, title_bn, subtitle_en, subtitle_bn, seed, target_kind, target_value, season)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: UpdateBanner :exec
UPDATE banners
SET title_en = $2, title_bn = $3, subtitle_en = $4, subtitle_bn = $5, seed = $6,
    target_kind = $7, target_value = $8, season = $9
WHERE id = $1;

-- name: DeleteBanner :execrows
DELETE FROM banners WHERE id = $1;

-- name: SetBannerPosition :exec
UPDATE banners SET position = $2 WHERE id = $1;

-- name: GetConfig :one
SELECT value FROM app_config WHERE key = $1;

-- name: SetConfig :exec
INSERT INTO app_config (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- name: ListAyahs :many
SELECT * FROM ayahs ORDER BY position;

-- name: GetIsbnLookup :one
SELECT * FROM isbn_lookup WHERE isbn = $1;
