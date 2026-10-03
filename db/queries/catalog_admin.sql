-- name: BookIDs :many
SELECT id FROM books;

-- name: GetBookForUpdate :one
SELECT * FROM books WHERE id = $1 FOR UPDATE;

-- name: OtherBooksIsbns :many
SELECT e.isbn FROM editions e WHERE e.isbn IS NOT NULL AND e.book_id <> $1;

-- name: ListEditionsOfBook :many
SELECT * FROM editions WHERE book_id = $1 ORDER BY position;

-- name: InsertBook :exec
INSERT INTO books (id, title, title_bn, short_title, author, author_id, publisher_id, category_id, section,
                   original_language, added_at, rating, tags, cover_seed, hidden, classes, exams, subject_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18);

-- name: UpdateBook :exec
UPDATE books
SET title = $2, title_bn = $3, short_title = $4, author = $5, author_id = $6, publisher_id = $7,
    category_id = $8, section = $9, original_language = $10, cover_seed = $11, classes = $12,
    exams = $13, subject_id = $14
WHERE id = $1;

-- name: SetBookHidden :execrows
UPDATE books SET hidden = $2 WHERE id = $1;

-- name: UpsertEdition :exec
INSERT INTO editions (id, book_id, format, language, price_bdt, list_price_bdt, stock, is_preorder, isbn)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE
SET format = EXCLUDED.format, language = EXCLUDED.language, price_bdt = EXCLUDED.price_bdt,
    list_price_bdt = EXCLUDED.list_price_bdt, stock = EXCLUDED.stock,
    is_preorder = EXCLUDED.is_preorder, isbn = EXCLUDED.isbn;

-- name: DeleteEditionsExcept :exec
DELETE FROM editions WHERE book_id = $1 AND NOT (id = ANY($2::text[]));

-- name: SetEditionStock :execrows
UPDATE editions SET stock = $2 WHERE id = $1;

-- name: GetEditionForUpdate :one
SELECT * FROM editions WHERE id = $1 FOR UPDATE;

-- name: GetPriceLow :one
SELECT * FROM price_lows WHERE edition_id = $1;

-- name: UpsertPriceLow :exec
INSERT INTO price_lows (edition_id, low_bdt, since) VALUES ($1, $2, $3)
ON CONFLICT (edition_id) DO UPDATE SET low_bdt = EXCLUDED.low_bdt, since = EXCLUDED.since;

-- name: CountBooksUsing :one
SELECT count(*) FROM books
WHERE (sqlc.arg(kind)::text = 'category' AND category_id = sqlc.arg(id)::text)
   OR (sqlc.arg(kind)::text = 'author' AND author_id = sqlc.arg(id)::text)
   OR (sqlc.arg(kind)::text = 'publisher' AND publisher_id = sqlc.arg(id)::text);

-- name: ListRecordCounts :many
SELECT 'category'::text AS kind, category_id AS id, count(*)::int AS n FROM books GROUP BY category_id
UNION ALL SELECT 'author', author_id, count(*)::int FROM books GROUP BY author_id
UNION ALL SELECT 'publisher', publisher_id, count(*)::int FROM books GROUP BY publisher_id;

-- name: GetCategory :one
SELECT * FROM categories WHERE id = $1 FOR UPDATE;

-- name: UpsertCategory :exec
INSERT INTO categories (id, section, name_en, name_bn) VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET section = EXCLUDED.section, name_en = EXCLUDED.name_en, name_bn = EXCLUDED.name_bn;

-- name: DeleteCategory :execrows
DELETE FROM categories WHERE id = $1;

-- name: GetAuthor :one
SELECT * FROM authors WHERE id = $1 FOR UPDATE;

-- name: UpsertAuthor :exec
INSERT INTO authors (id, name, name_bn, bio) VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, name_bn = EXCLUDED.name_bn;

-- name: RenameBooksAuthor :exec
UPDATE books SET author = $2 WHERE author_id = $1;

-- name: DeleteAuthor :execrows
DELETE FROM authors WHERE id = $1;

-- name: UpsertPublisher :exec
INSERT INTO publishers (id, name, name_bn) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, name_bn = EXCLUDED.name_bn;

-- name: DeletePublisher :execrows
DELETE FROM publishers WHERE id = $1;

-- name: ListCollectionIDs :many
SELECT id FROM collections;

-- name: UpsertCollection :exec
INSERT INTO collections (id, title_en, title_bn, note_en, note_bn, section, expert_id, book_ids)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET title_en = EXCLUDED.title_en, title_bn = EXCLUDED.title_bn, note_en = EXCLUDED.note_en,
    note_bn = EXCLUDED.note_bn, section = EXCLUDED.section, expert_id = EXCLUDED.expert_id,
    book_ids = EXCLUDED.book_ids;

-- name: DeleteCollection :execrows
DELETE FROM collections WHERE id = $1;

-- name: ListStaffBooklistIDs :many
SELECT id FROM booklists WHERE owner_id IS NULL;

-- name: UpsertStaffBooklist :exec
INSERT INTO booklists (id, owner_id, kind, title_en, title_bn, note_en, note_bn, book_ids, updated_at)
VALUES ($1, NULL, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET kind = EXCLUDED.kind, title_en = EXCLUDED.title_en, title_bn = EXCLUDED.title_bn,
    note_en = EXCLUDED.note_en, note_bn = EXCLUDED.note_bn, book_ids = EXCLUDED.book_ids,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteStaffBooklist :execrows
DELETE FROM booklists WHERE id = $1 AND owner_id IS NULL;

-- name: ExpertExists :one
SELECT EXISTS (SELECT 1 FROM experts WHERE id = $1);

-- name: PublisherExists :one
SELECT EXISTS (SELECT 1 FROM publishers WHERE id = $1);

-- name: LowStockEditions :many
SELECT e.id AS edition_id, e.format, e.language, e.stock, b.id AS book_id, b.title, b.cover_seed
FROM editions e JOIN books b ON b.id = e.book_id
WHERE e.format <> 'ebook' AND NOT e.is_preorder AND e.stock <= $1
ORDER BY e.stock, b.position, e.position;
