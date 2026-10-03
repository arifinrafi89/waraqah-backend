-- +goose Up
-- The cart keeps only what was added and how many; prices come from the catalog when it is read.
CREATE TABLE cart_lines (
    user_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('edition', 'certifiedUsed', 'listing', 'bundle')),
    item_id    text NOT NULL,
    position   bigserial NOT NULL,
    quantity   integer NOT NULL DEFAULT 1,
    added_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, kind, item_id)
);
CREATE INDEX cart_lines_user_idx ON cart_lines (user_id, position);

-- The flash sale runs all day, every day (it ends at the next midnight, Dhaka time); only its
-- items and prices are stored.
CREATE TABLE flash_sale_items (
    edition_id text PRIMARY KEY REFERENCES editions (id) ON DELETE CASCADE,
    position   integer NOT NULL,
    price_bdt  integer NOT NULL
);

CREATE TABLE bundles (
    id          text PRIMARY KEY,
    position    integer NOT NULL,
    title       text NOT NULL,
    edition_ids text[] NOT NULL,
    price_bdt   integer NOT NULL
);

CREATE TABLE preorders (
    edition_id text PRIMARY KEY REFERENCES editions (id) ON DELETE CASCADE,
    position   integer NOT NULL
);

CREATE TABLE wishlist_items (
    user_id  text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    book_id  text NOT NULL REFERENCES books (id) ON DELETE CASCADE,
    added_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, book_id)
);
CREATE INDEX wishlist_items_user_idx ON wishlist_items (user_id, added_at DESC);

-- A share link of one reader wishlist; ownerName is the name friends see.
CREATE TABLE wishlist_shares (
    id         text PRIMARY KEY,
    user_id    text NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    owner_name text NOT NULL
);

CREATE TABLE alerts (
    id               text PRIMARY KEY,
    user_id          text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    position         bigserial NOT NULL,
    kind             text NOT NULL CHECK (kind IN ('backInStock', 'priceDrop')),
    book_id          text NOT NULL REFERENCES books (id) ON DELETE CASCADE,
    edition_id       text NOT NULL REFERENCES editions (id) ON DELETE CASCADE,
    target_price_bdt integer,
    fired_at         timestamptz
);
CREATE INDEX alerts_user_idx ON alerts (user_id, position);
CREATE INDEX alerts_unfired_idx ON alerts (edition_id) WHERE fired_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS wishlist_shares;
DROP TABLE IF EXISTS wishlist_items;
DROP TABLE IF EXISTS preorders;
DROP TABLE IF EXISTS bundles;
DROP TABLE IF EXISTS flash_sale_items;
DROP TABLE IF EXISTS cart_lines;
