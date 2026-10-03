-- +goose Up
-- A used book bought through Waraqah: the buyer pays in the app and Waraqah holds the money
-- until the buyer confirms, or a moderator settles a dispute. Ids are HS-<n> (handled_sale_seq).
-- The fee is fixed when the sale is made, so a later change of SaleMath never changes old sales.
CREATE TABLE handled_sales (
    id             text PRIMARY KEY,
    listing_id     text NOT NULL REFERENCES listings (id),
    buyer_id       text NOT NULL REFERENCES users (id),
    seller_id      text NOT NULL REFERENCES users (id),
    price_bdt      integer NOT NULL CHECK (price_bdt > 0),
    fee_bdt        integer NOT NULL,
    delivery_bdt   integer NOT NULL,
    method         text NOT NULL CHECK (method IN ('bkash', 'nagad', 'card')),
    status         text NOT NULL DEFAULT 'paid'
                   CHECK (status IN ('paid', 'sent', 'completed', 'disputed', 'refunded', 'released', 'cancelled')),
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    dispute_reason text CHECK (dispute_reason IN ('notAsDescribed', 'damaged', 'photocopy', 'wrongBook', 'notReceived')),
    dispute_note   text,
    -- Dispute photos stay base64: the app decodes them from the sale (contract v1), like return photos.
    dispute_photos jsonb NOT NULL DEFAULT '[]'
);
CREATE INDEX handled_sales_buyer_idx ON handled_sales (buyer_id, created_at DESC);
CREATE INDEX handled_sales_seller_idx ON handled_sales (seller_id, created_at DESC);
CREATE INDEX handled_sales_listing_idx ON handled_sales (listing_id);
CREATE INDEX handled_sales_open_idx ON handled_sales (status, created_at) WHERE status IN ('paid', 'disputed');

-- Money sent from a seller's earnings to their bKash.
CREATE TABLE payouts (
    id         bigserial PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users (id),
    amount_bdt integer NOT NULL CHECK (amount_bdt > 0),
    at         timestamptz NOT NULL
);
CREATE INDEX payouts_user_idx ON payouts (user_id, at DESC);

-- A book a reader sells to Waraqah for an instant quote. The book is kept as it was quoted
-- (title, author, new price), so the reader's history does not change with the catalog.
-- Ids are SB-<n> (sell_back_seq); the demo seeds use SB-201 to SB-203.
CREATE SEQUENCE sell_back_seq START 301;
CREATE TABLE sell_backs (
    id               text PRIMARY KEY,
    user_id          text NOT NULL REFERENCES users (id),
    book_id          text NOT NULL,
    title            text NOT NULL,
    author           text NOT NULL,
    new_price_bdt    integer NOT NULL,
    cover_seed       integer NOT NULL DEFAULT 0,
    condition        text NOT NULL CHECK (condition IN ('likeNew', 'veryGood', 'good', 'acceptable')),
    flags            integer NOT NULL DEFAULT 0 CHECK (flags >= 0),
    quote_bdt        integer NOT NULL,
    status           text NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'pickedUp', 'paid', 'returned')),
    pickup_address   text NOT NULL,
    created_at       timestamptz NOT NULL,
    graded_condition text CHECK (graded_condition IN ('likeNew', 'veryGood', 'good', 'acceptable')),
    paid_bdt         integer
);
CREATE INDEX sell_backs_user_idx ON sell_backs (user_id, created_at DESC);
CREATE INDEX sell_backs_status_idx ON sell_backs (status, created_at);

-- Waraqah's Certified Used copies: Sell Back books graded and put on sale. A copy is one of a
-- kind; sold_at is set when an order takes it.
CREATE TABLE certified_used (
    id         text PRIMARY KEY,
    book_id    text NOT NULL,
    condition  text NOT NULL CHECK (condition IN ('likeNew', 'veryGood', 'good', 'acceptable')),
    price_bdt  integer NOT NULL CHECK (price_bdt > 0),
    created_at timestamptz NOT NULL,
    sold_at    timestamptz
);
CREATE INDEX certified_used_book_idx ON certified_used (book_id, price_bdt) WHERE sold_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS certified_used;
DROP TABLE IF EXISTS sell_backs;
DROP SEQUENCE IF EXISTS sell_back_seq;
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS handled_sales;
