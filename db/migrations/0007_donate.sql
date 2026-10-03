-- +goose Up
-- Verified places that take donated books. Staff remove a place with removed_at; it stays for the
-- donation orders that name it.
CREATE TABLE donate_places (
    id         text PRIMARY KEY,
    position   bigserial NOT NULL,
    name       text NOT NULL,
    kind       text NOT NULL CHECK (kind IN ('library', 'school', 'madrasa', 'orphanage')),
    district   text NOT NULL,
    area       text NOT NULL,
    story      text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    removed_at timestamptz
);

CREATE TABLE donate_needs (
    place_id text NOT NULL REFERENCES donate_places (id) ON DELETE CASCADE,
    position integer NOT NULL,
    book_id  text NOT NULL,
    wanted   integer NOT NULL CHECK (wanted > 0),
    received integer NOT NULL DEFAULT 0 CHECK (received >= 0),
    PRIMARY KEY (place_id, book_id)
);

-- +goose Down
DROP TABLE IF EXISTS donate_needs;
DROP TABLE IF EXISTS donate_places;
