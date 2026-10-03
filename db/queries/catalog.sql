-- name: ListBooks :many
SELECT * FROM books ORDER BY position;

-- name: ListEditions :many
SELECT * FROM editions ORDER BY book_id, position;

-- name: ListCategories :many
SELECT * FROM categories ORDER BY position;

-- name: ListAuthors :many
SELECT * FROM authors ORDER BY position;

-- name: ListPublishers :many
SELECT * FROM publishers ORDER BY position;

-- name: ListSubjects :many
SELECT * FROM subjects ORDER BY position;

-- name: ListSeries :many
SELECT * FROM series ORDER BY position;

-- name: GetBookDetails :one
SELECT * FROM book_details WHERE book_id = $1;

-- name: GetLookInside :one
SELECT * FROM look_inside WHERE book_id = $1;

-- name: ListExperts :many
SELECT * FROM experts ORDER BY position;

-- name: ListCollections :many
SELECT * FROM collections ORDER BY position;

-- name: ListBooklists :many
SELECT * FROM booklists
WHERE owner_id IS NULL OR owner_id = sqlc.arg(viewer)::text
ORDER BY position;

-- name: GetBooklist :one
SELECT * FROM booklists WHERE id = $1;

-- name: InsertBooklist :one
INSERT INTO booklists (id, owner_id, kind, title_en, title_bn, book_ids, updated_at)
VALUES ($1, $2, 'personal', $3, $3, $4, $5)
RETURNING *;

-- name: UpdateBooklist :one
UPDATE booklists SET title_en = $2, title_bn = $3, book_ids = $4, updated_at = $5
WHERE id = $1
RETURNING *;

-- name: DeleteOwnBooklist :execrows
DELETE FROM booklists WHERE id = $1 AND owner_id = $2;

-- name: ListQuestions :many
SELECT * FROM book_questions WHERE book_id = $1 ORDER BY position;

-- name: ListAnswersForBook :many
SELECT a.* FROM book_answers a
JOIN book_questions q ON q.id = a.question_id
WHERE q.book_id = $1
ORDER BY a.position;

-- name: InsertQuestion :exec
INSERT INTO book_questions (id, book_id, user_id, asker_name, text, asked_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertAnswer :exec
INSERT INTO book_answers (id, question_id, user_id, author_name, text, answered_at, is_staff)
SELECT $1, q.id, $3, $4, $5, $6, $7 FROM book_questions q WHERE q.id = $2 AND q.book_id = $8;

-- name: ListPriceLows :many
SELECT * FROM price_lows;

-- name: ListSalesSince :many
SELECT edition_id, sum(copies)::int AS copies FROM sales_by_month
WHERE month >= $1 GROUP BY edition_id;

-- name: SetBookRating :exec
UPDATE books SET rating = $2 WHERE id = $1;
