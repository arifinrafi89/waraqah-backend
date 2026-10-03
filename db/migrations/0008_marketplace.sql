-- +goose Up
-- Books sold before the records of the app start, for the seller page of a reader.
ALTER TABLE users ADD COLUMN sold_before integer NOT NULL DEFAULT 0;

-- A used book a reader offers. position grows with every new listing, so the list is newest first.
CREATE TABLE listings (
    id               text PRIMARY KEY,
    position         bigserial NOT NULL,
    seller_id        text NOT NULL REFERENCES users (id),
    title            text NOT NULL,
    price_bdt        integer NOT NULL,
    condition        text NOT NULL CHECK (condition IN ('likeNew', 'veryGood', 'good', 'acceptable')),
    flags            text[] NOT NULL DEFAULT '{}',
    is_negotiable    boolean NOT NULL DEFAULT false,
    handover         text NOT NULL CHECK (handover IN ('meetInPerson', 'delivery')),
    status           text NOT NULL CHECK (status IN ('draft', 'inReview', 'changesRequested', 'rejected', 'live', 'reserved', 'sold')),
    rejection_reason text,
    book_id          text,
    cover_seed       integer NOT NULL DEFAULT 0,
    district         text,
    area             text,
    category_id      text,
    new_price_bdt    integer,
    note             text,
    buyer_id         text REFERENCES users (id),
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX listings_position_idx ON listings (position DESC);
CREATE INDEX listings_seller_idx ON listings (seller_id);
CREATE INDEX listings_status_idx ON listings (status);
CREATE INDEX listings_book_idx ON listings (book_id) WHERE book_id IS NOT NULL;

-- One row per photo slot kept; public_id is the Cloudinary asset to delete with it.
CREATE TABLE listing_photos (
    listing_id text NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    slot       text NOT NULL CHECK (slot IN ('front', 'back', 'spine', 'inside', 'damage')),
    position   integer NOT NULL,
    url        text NOT NULL DEFAULT '',
    public_id  text NOT NULL DEFAULT '',
    PRIMARY KEY (listing_id, slot)
);

-- What readers gave each other after a sale: 1 to 5 stars and a few words.
CREATE TABLE ratings (
    id         bigserial PRIMARY KEY,
    from_id    text NOT NULL REFERENCES users (id),
    to_id      text NOT NULL REFERENCES users (id),
    listing_id text,
    stars      integer NOT NULL CHECK (stars BETWEEN 1 AND 5),
    comment    text,
    at         timestamptz NOT NULL
);
CREATE INDEX ratings_to_idx ON ratings (to_id, at DESC);
CREATE UNIQUE INDEX ratings_once_idx ON ratings (listing_id, from_id) WHERE listing_id IS NOT NULL;

CREATE TABLE blocks (
    user_id    text NOT NULL REFERENCES users (id),
    blocked_id text NOT NULL REFERENCES users (id),
    at         timestamptz NOT NULL,
    PRIMARY KEY (user_id, blocked_id)
);

CREATE TABLE reports (
    id          text PRIMARY KEY,
    position    bigserial NOT NULL,
    kind        text NOT NULL CHECK (kind IN ('listing', 'user', 'message', 'bite', 'comment', 'review')),
    target_id   text NOT NULL,
    reason      text NOT NULL CHECK (reason IN ('spam', 'fake', 'photocopy', 'harassment', 'offensive', 'other')),
    note        text,
    reporter_id text NOT NULL REFERENCES users (id),
    status      text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'removed', 'dismissed', 'warned', 'banned')),
    created_at  timestamptz NOT NULL
);
CREATE INDEX reports_open_idx ON reports (created_at) WHERE status = 'open';

-- The audit log of the Moderation Center. by_name is the staff member name at the time.
CREATE TABLE moderation_log (
    id      bigserial PRIMARY KEY,
    at      timestamptz NOT NULL,
    by_name text NOT NULL,
    action  text NOT NULL,
    subject text NOT NULL,
    reason  text
);

-- +goose Down
DROP TABLE IF EXISTS moderation_log;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS ratings;
DROP TABLE IF EXISTS listing_photos;
DROP TABLE IF EXISTS listings;
ALTER TABLE users DROP COLUMN IF EXISTS sold_before;
