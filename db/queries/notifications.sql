-- name: InsertNotification :exec
INSERT INTO notifications (id, user_id, kind, params, target, read_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListNotifications :many
SELECT id, kind, params, target, read_at, created_at
FROM notifications WHERE user_id = $1
ORDER BY created_at DESC, id DESC;

-- name: MarkNotificationRead :execrows
UPDATE notifications SET read_at = COALESCE(read_at, $3) WHERE user_id = $1 AND id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read_at = $2 WHERE user_id = $1 AND read_at IS NULL;

-- name: CountUnreadNotifications :one
SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL;
