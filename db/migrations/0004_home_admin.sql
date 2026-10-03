-- +goose Up
-- Records keep the order they were added in (Staff lists show them that way).
ALTER TABLE categories ADD COLUMN position bigserial;
ALTER TABLE authors ADD COLUMN position bigserial;
ALTER TABLE publishers ADD COLUMN position bigserial;
ALTER TABLE subjects ADD COLUMN position bigserial;

-- Banners in display order; position is explicit so a move swaps two neighbours.
CREATE TABLE banners (
    id           text PRIMARY KEY,
    position     integer NOT NULL,
    title_en     text NOT NULL,
    title_bn     text NOT NULL,
    subtitle_en  text NOT NULL,
    subtitle_bn  text NOT NULL,
    seed         integer NOT NULL DEFAULT 0,
    target_kind  text NOT NULL,
    target_value text NOT NULL,
    season       text
);
CREATE INDEX banners_position_idx ON banners (position);

-- Small settings kept as json. The Season that Staff forced on Home is the key season_override.
CREATE TABLE app_config (
    key   text PRIMARY KEY,
    value jsonb NOT NULL
);

-- The verses Home rotates through, one a day (day of year modulo the count).
CREATE TABLE ayahs (
    position     integer PRIMARY KEY,
    arabic       text NOT NULL,
    translation  text NOT NULL,
    surah_en     text NOT NULL,
    surah_bn     text NOT NULL,
    surah_number integer NOT NULL,
    verse_number integer NOT NULL
);

-- Books outside the catalog that the ISBN lookup knows (a stand-in for a real ISBN service).
CREATE TABLE isbn_lookup (
    isbn           text PRIMARY KEY,
    title          text NOT NULL,
    title_bn       text,
    author         text NOT NULL,
    publisher      text NOT NULL,
    language       text NOT NULL DEFAULT 'english',
    format         text NOT NULL DEFAULT 'paperback',
    list_price_bdt integer NOT NULL
);

-- +goose Down
ALTER TABLE subjects DROP COLUMN IF EXISTS position;
ALTER TABLE publishers DROP COLUMN IF EXISTS position;
ALTER TABLE authors DROP COLUMN IF EXISTS position;
ALTER TABLE categories DROP COLUMN IF EXISTS position;
DROP TABLE IF EXISTS isbn_lookup;
DROP TABLE IF EXISTS ayahs;
DROP TABLE IF EXISTS app_config;
DROP TABLE IF EXISTS banners;
