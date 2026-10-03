-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetRefreshTokenForUpdate :one
SELECT rt.id, rt.user_id, rt.expires_at, rt.revoked_at,
       u.banned::boolean AS user_banned, (u.deleted_at IS NOT NULL)::boolean AS user_deleted
FROM refresh_tokens rt
JOIN users u ON u.id = rt.user_id
WHERE rt.token_hash = $1
FOR UPDATE OF rt;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = sqlc.arg(revoked_at)::timestamptz
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeAllRefreshTokens :exec
UPDATE refresh_tokens SET revoked_at = sqlc.arg(revoked_at)::timestamptz
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: RevokeRefreshTokenByHash :exec
UPDATE refresh_tokens SET revoked_at = sqlc.arg(revoked_at)::timestamptz
WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: PruneRefreshTokens :execrows
DELETE FROM refresh_tokens WHERE expires_at < $1;

-- name: UpsertOTP :exec
INSERT INTO otp_codes (contact, purpose, code_hash, expires_at, attempts, created_at)
VALUES ($1, $2, $3, $4, 0, $5)
ON CONFLICT (contact, purpose) DO UPDATE
SET code_hash = EXCLUDED.code_hash, expires_at = EXCLUDED.expires_at,
    attempts = 0, created_at = EXCLUDED.created_at;

-- name: GetOTPForUpdate :one
SELECT * FROM otp_codes WHERE contact = $1 AND purpose = $2 FOR UPDATE;

-- name: BumpOTPAttempts :one
UPDATE otp_codes SET attempts = attempts + 1
WHERE contact = $1 AND purpose = $2
RETURNING attempts;

-- name: DeleteOTP :exec
DELETE FROM otp_codes WHERE contact = $1 AND purpose = $2;

-- name: GetOTP :one
SELECT * FROM otp_codes WHERE contact = $1 AND purpose = $2;
