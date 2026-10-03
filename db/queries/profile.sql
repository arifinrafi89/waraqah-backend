-- name: GetProfile :one
SELECT name, phone, photo_data FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: SaveProfile :exec
UPDATE users SET name = $2, phone = $3, photo_data = $4 WHERE id = $1;

-- name: GetPrefs :one
SELECT muted, profile_visible, activity_visible FROM profile_prefs WHERE user_id = $1;

-- name: UpsertPrefs :exec
INSERT INTO profile_prefs (user_id, muted, profile_visible, activity_visible)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET muted = EXCLUDED.muted, profile_visible = EXCLUDED.profile_visible,
    activity_visible = EXCLUDED.activity_visible;

-- name: ListAddresses :many
SELECT id, label, recipient, phone, line, upazila, district, division, is_default
FROM addresses WHERE user_id = $1
ORDER BY is_default DESC, position;

-- name: GetAddress :one
SELECT id, label, recipient, phone, line, upazila, district, division, is_default
FROM addresses WHERE user_id = $1 AND id = $2 FOR UPDATE;

-- name: CountAddresses :one
SELECT count(*) FROM addresses WHERE user_id = $1;

-- name: InsertAddress :exec
INSERT INTO addresses (id, user_id, label, recipient, phone, line, upazila, district, division, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: UpdateAddress :exec
UPDATE addresses
SET label = $3, recipient = $4, phone = $5, line = $6, upazila = $7, district = $8, division = $9
WHERE user_id = $1 AND id = $2;

-- name: DeleteAddress :exec
DELETE FROM addresses WHERE user_id = $1 AND id = $2;

-- name: ClearDefaultAddress :exec
UPDATE addresses SET is_default = false WHERE user_id = $1 AND is_default;

-- name: SetDefaultAddress :exec
UPDATE addresses SET is_default = true WHERE user_id = $1 AND id = $2;

-- name: FirstAddressID :one
SELECT id FROM addresses WHERE user_id = $1 ORDER BY position LIMIT 1;

-- name: AnonymiseUser :exec
UPDATE users
SET deleted_at = $2, email = NULL, name = 'Deleted reader', phone = '', photo_url = NULL,
    photo_data = NULL, password_hash = NULL
WHERE id = $1;

-- name: DeletePrefs :exec
DELETE FROM profile_prefs WHERE user_id = $1;

-- name: DeleteAddresses :exec
DELETE FROM addresses WHERE user_id = $1;
