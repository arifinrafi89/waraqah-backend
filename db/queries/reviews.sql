-- name: ListReviews :many
SELECT r.*, u.name AS author_name
FROM reviews r JOIN users u ON u.id = r.user_id
WHERE r.book_id = $1 AND u.deleted_at IS NULL
ORDER BY r.created_at DESC, r.id DESC;

-- name: FindReview :one
SELECT * FROM reviews WHERE id = $1;

-- name: GetReviewOf :one
SELECT * FROM reviews WHERE book_id = $1 AND user_id = $2;

-- name: InsertReview :exec
INSERT INTO reviews (id, book_id, user_id, stars, text, created_at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: EditReview :exec
UPDATE reviews SET stars = $3, text = $4, edited_at = $5 WHERE book_id = $1 AND user_id = $2;

-- name: DeleteReview :exec
DELETE FROM reviews WHERE book_id = $1 AND user_id = $2;

-- name: ReviewStats :one
SELECT count(*)::integer AS count, coalesce(avg(stars), 0)::float8 AS average FROM reviews WHERE book_id = $1;

-- name: SeedReview :exec
INSERT INTO reviews (id, book_id, user_id, stars, text, seed_verified, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT DO NOTHING;
