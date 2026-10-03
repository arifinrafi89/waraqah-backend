-- name: UpsertSeedBanner :exec
INSERT INTO banners (id, position, title_en, title_bn, subtitle_en, subtitle_bn, seed, target_kind, target_value, season)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE
SET title_en = EXCLUDED.title_en, title_bn = EXCLUDED.title_bn, subtitle_en = EXCLUDED.subtitle_en,
    subtitle_bn = EXCLUDED.subtitle_bn, seed = EXCLUDED.seed, target_kind = EXCLUDED.target_kind,
    target_value = EXCLUDED.target_value, season = EXCLUDED.season;

-- name: UpsertSeedAyah :exec
INSERT INTO ayahs (position, arabic, translation, surah_en, surah_bn, surah_number, verse_number)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (position) DO UPDATE
SET arabic = EXCLUDED.arabic, translation = EXCLUDED.translation, surah_en = EXCLUDED.surah_en,
    surah_bn = EXCLUDED.surah_bn, surah_number = EXCLUDED.surah_number, verse_number = EXCLUDED.verse_number;

-- name: UpsertSeedIsbn :exec
INSERT INTO isbn_lookup (isbn, title, title_bn, author, publisher, language, format, list_price_bdt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (isbn) DO UPDATE
SET title = EXCLUDED.title, title_bn = EXCLUDED.title_bn, author = EXCLUDED.author,
    publisher = EXCLUDED.publisher, language = EXCLUDED.language, format = EXCLUDED.format,
    list_price_bdt = EXCLUDED.list_price_bdt;
