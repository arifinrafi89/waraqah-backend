-- +goose Up
-- A picture of the book. Books and listings without one keep the generated cover in the app.
ALTER TABLE books ADD COLUMN cover_url text;
ALTER TABLE listings ADD COLUMN cover_url text;

-- +goose Down
ALTER TABLE listings DROP COLUMN IF EXISTS cover_url;
ALTER TABLE books DROP COLUMN IF EXISTS cover_url;
