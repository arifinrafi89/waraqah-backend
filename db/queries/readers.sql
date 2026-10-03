-- name: GetReaderPage :one
SELECT id, name, area, district, member_since FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: IsFollowing :one
SELECT EXISTS (SELECT 1 FROM follows WHERE follower_id = $1 AND followee_id = $2);

-- name: Follow :execrows
INSERT INTO follows (follower_id, followee_id, at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;

-- name: Unfollow :exec
DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2;

-- name: FollowCounts :one
SELECT (SELECT count(*) FROM follows f JOIN users u ON u.id = f.follower_id WHERE f.followee_id = $1 AND u.deleted_at IS NULL)::integer AS followers,
       (SELECT count(*) FROM follows f JOIN users u ON u.id = f.followee_id WHERE f.follower_id = $1 AND u.deleted_at IS NULL)::integer AS following;

-- name: ListFollowing :many
SELECT followee_id FROM follows WHERE follower_id = $1;

-- name: SeedFollow :exec
INSERT INTO follows (follower_id, followee_id, at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;

-- name: SeedPrivateProfile :exec
INSERT INTO profile_prefs (user_id, profile_visible) VALUES ($1, false) ON CONFLICT (user_id) DO NOTHING;
