-- name: UpsertSeedCategory :exec
INSERT INTO categories (id, section, name_en, name_bn) VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET section = EXCLUDED.section, name_en = EXCLUDED.name_en, name_bn = EXCLUDED.name_bn;

-- name: UpsertSeedAuthor :exec
INSERT INTO authors (id, name, name_bn, bio) VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, name_bn = EXCLUDED.name_bn, bio = EXCLUDED.bio;

-- name: UpsertSeedPublisher :exec
INSERT INTO publishers (id, name, name_bn) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, name_bn = EXCLUDED.name_bn;

-- name: UpsertSeedSubject :exec
INSERT INTO subjects (id, name_en, name_bn) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET name_en = EXCLUDED.name_en, name_bn = EXCLUDED.name_bn;

-- name: UpsertSeedBook :exec
INSERT INTO books (id, title, title_bn, short_title, author, author_id, publisher_id, category_id, section,
                   original_language, added_at, rating, tags, cover_seed, hidden, classes, exams, subject_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title, title_bn = EXCLUDED.title_bn, short_title = EXCLUDED.short_title,
    author = EXCLUDED.author, author_id = EXCLUDED.author_id, publisher_id = EXCLUDED.publisher_id,
    category_id = EXCLUDED.category_id, section = EXCLUDED.section, original_language = EXCLUDED.original_language,
    added_at = EXCLUDED.added_at, rating = EXCLUDED.rating, tags = EXCLUDED.tags, cover_seed = EXCLUDED.cover_seed,
    classes = EXCLUDED.classes, exams = EXCLUDED.exams, subject_id = EXCLUDED.subject_id;

-- name: UpsertSeedEdition :exec
INSERT INTO editions (id, book_id, format, language, price_bdt, list_price_bdt, stock, is_preorder, isbn)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE
SET book_id = EXCLUDED.book_id, format = EXCLUDED.format, language = EXCLUDED.language,
    price_bdt = EXCLUDED.price_bdt, list_price_bdt = EXCLUDED.list_price_bdt, is_preorder = EXCLUDED.is_preorder,
    isbn = EXCLUDED.isbn;

-- name: UpsertSeedBookDetails :exec
INSERT INTO book_details (book_id, description, pages) VALUES ($1, $2, $3)
ON CONFLICT (book_id) DO UPDATE SET description = EXCLUDED.description, pages = EXCLUDED.pages;

-- name: UpsertSeedLookInside :exec
INSERT INTO look_inside (book_id, contents, sample_pages) VALUES ($1, $2, $3)
ON CONFLICT (book_id) DO UPDATE SET contents = EXCLUDED.contents, sample_pages = EXCLUDED.sample_pages;

-- name: UpsertSeedSeries :exec
INSERT INTO series (id, name, entries) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, entries = EXCLUDED.entries;

-- name: UpsertSeedExpert :exec
INSERT INTO experts (id, name, name_bn, credential_en, credential_bn, kind, verified)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name, name_bn = EXCLUDED.name_bn, credential_en = EXCLUDED.credential_en,
    credential_bn = EXCLUDED.credential_bn, kind = EXCLUDED.kind, verified = EXCLUDED.verified;

-- name: UpsertSeedCollection :exec
INSERT INTO collections (id, title_en, title_bn, note_en, note_bn, section, expert_id, book_ids)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET title_en = EXCLUDED.title_en, title_bn = EXCLUDED.title_bn, note_en = EXCLUDED.note_en,
    note_bn = EXCLUDED.note_bn, section = EXCLUDED.section, expert_id = EXCLUDED.expert_id,
    book_ids = EXCLUDED.book_ids;

-- name: UpsertSeedBooklist :exec
INSERT INTO booklists (id, owner_id, kind, title_en, title_bn, note_en, note_bn, book_ids)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET owner_id = EXCLUDED.owner_id, kind = EXCLUDED.kind, title_en = EXCLUDED.title_en,
    title_bn = EXCLUDED.title_bn, note_en = EXCLUDED.note_en, note_bn = EXCLUDED.note_bn,
    book_ids = EXCLUDED.book_ids;

-- name: UpsertSeedQuestion :exec
INSERT INTO book_questions (id, book_id, asker_name, text, asked_at) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO NOTHING;

-- name: UpsertSeedAnswer :exec
INSERT INTO book_answers (id, question_id, author_name, text, answered_at, is_staff)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO NOTHING;

-- name: UpsertSeedPriceLow :exec
INSERT INTO price_lows (edition_id, low_bdt, since) VALUES ($1, $2, $3)
ON CONFLICT (edition_id) DO UPDATE SET low_bdt = EXCLUDED.low_bdt;

-- name: UpsertSeedSales :exec
INSERT INTO sales_by_month (edition_id, month, copies) VALUES ($1, $2, $3)
ON CONFLICT (edition_id, month) DO UPDATE SET copies = EXCLUDED.copies;
