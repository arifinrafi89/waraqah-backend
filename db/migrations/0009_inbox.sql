-- +goose Up
-- A conversation of one buyer with the seller of a listing. The read markers are the position of
-- the last message each side has seen, so removing a message never shifts them.
CREATE TABLE threads (
    id          text PRIMARY KEY,
    listing_id  text NOT NULL REFERENCES listings (id),
    buyer_id    text NOT NULL REFERENCES users (id),
    seller_id   text NOT NULL REFERENCES users (id),
    buyer_read  bigint NOT NULL DEFAULT 0,
    seller_read bigint NOT NULL DEFAULT 0,
    bot_replied boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (listing_id, buyer_id)
);
CREATE INDEX threads_buyer_idx ON threads (buyer_id);
CREATE INDEX threads_seller_idx ON threads (seller_id);

CREATE TABLE offers (
    id         text PRIMARY KEY,
    position   bigserial NOT NULL,
    thread_id  text NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    amount_bdt integer NOT NULL CHECK (amount_bdt > 0),
    handover   text NOT NULL CHECK (handover IN ('meetup', 'courier')),
    status     text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'declined', 'closed'))
);
CREATE INDEX offers_thread_idx ON offers (thread_id);

-- Exactly one of text, offer_id and event is set. author_id is a reader or 'system'.
CREATE TABLE messages (
    id         text PRIMARY KEY,
    position   bigserial NOT NULL,
    thread_id  text NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    author_id  text NOT NULL,
    at         timestamptz NOT NULL,
    text       text,
    offer_id   text REFERENCES offers (id) ON DELETE SET NULL,
    event      text CHECK (event IN ('offerAccepted', 'offerDeclined', 'reservedElsewhere', 'madeAvailable', 'sold', 'soldElsewhere')),
    amount_bdt integer
);
CREATE INDEX messages_thread_idx ON messages (thread_id, position);

-- What a reader asked for: "I am looking for this book".
CREATE TABLE book_requests (
    id            text PRIMARY KEY,
    position      bigserial NOT NULL,
    requester_id  text NOT NULL REFERENCES users (id),
    title         text NOT NULL,
    author        text,
    book_id       text,
    max_price_bdt integer,
    note          text,
    is_open       boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL
);
CREATE INDEX book_requests_requester_idx ON book_requests (requester_id);

-- +goose Down
DROP TABLE IF EXISTS book_requests;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS offers;
DROP TABLE IF EXISTS threads;
