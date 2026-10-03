-- +goose Up
-- A code the reader types at checkout. Staff create these in Admin -> Orders.
CREATE TABLE coupons (
    code             text PRIMARY KEY,
    position         bigserial NOT NULL,
    kind             text NOT NULL CHECK (kind IN ('percentOff', 'amountOff', 'freeDelivery')),
    value            integer NOT NULL DEFAULT 0,
    min_order_bdt    integer NOT NULL DEFAULT 0,
    max_discount_bdt integer,
    expires_at       timestamptz,
    created_by       text REFERENCES users (id) ON DELETE SET NULL
);

CREATE TABLE orders (
    number           text PRIMARY KEY,
    user_id          text NOT NULL REFERENCES users (id),
    status           text NOT NULL CHECK (status IN ('placed', 'confirmed', 'packed', 'shipped', 'delivered', 'cancelled')),
    placed_at        timestamptz NOT NULL,
    address_label    text NOT NULL,
    address_line     text NOT NULL,
    payment          text NOT NULL CHECK (payment IN ('bkash', 'nagad', 'cashOnDelivery', 'card')),
    subtotal_bdt     integer NOT NULL,
    delivery_fee_bdt integer NOT NULL,
    discount_bdt     integer NOT NULL,
    total_bdt        integer NOT NULL,
    needs_delivery   boolean NOT NULL DEFAULT true,
    points_used      integer NOT NULL DEFAULT 0,
    points_earned    integer NOT NULL DEFAULT 0,
    gift             jsonb,
    gift_wrap_bdt    integer NOT NULL DEFAULT 0,
    is_donation      boolean NOT NULL DEFAULT false,
    donate_place_id  text,
    wallet_used_bdt  integer NOT NULL DEFAULT 0,
    refunded_bdt     integer NOT NULL DEFAULT 0
);
CREATE INDEX orders_user_placed_idx ON orders (user_id, placed_at DESC);
CREATE INDEX orders_placed_idx ON orders (placed_at DESC);

CREATE TABLE order_lines (
    order_number   text NOT NULL REFERENCES orders (number) ON DELETE CASCADE,
    position       integer NOT NULL,
    book_id        text NOT NULL,
    edition_id     text,
    title          text NOT NULL,
    author         text NOT NULL,
    quantity       integer NOT NULL,
    unit_price_bdt integer NOT NULL,
    format         text,
    language       text,
    cover_seed     integer NOT NULL DEFAULT 0,
    PRIMARY KEY (order_number, position)
);

CREATE TABLE order_history (
    order_number text NOT NULL REFERENCES orders (number) ON DELETE CASCADE,
    position     integer NOT NULL,
    status       text NOT NULL,
    at           timestamptz NOT NULL,
    PRIMARY KEY (order_number, position)
);

-- Return photos stay base64: the app decodes them from the order (contract v1).
CREATE TABLE order_returns (
    order_number  text PRIMARY KEY REFERENCES orders (number) ON DELETE CASCADE,
    reason        text NOT NULL,
    status        text NOT NULL,
    requested_at  timestamptz NOT NULL,
    decided_at    timestamptz,
    note          text NOT NULL DEFAULT '',
    photos        jsonb NOT NULL DEFAULT '[]'
);

-- The wallet balance is always the sum of its entries, never a stored number.
CREATE TABLE wallet_entries (
    id           bigserial PRIMARY KEY,
    user_id      text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    amount_bdt   integer NOT NULL,
    reason       text NOT NULL CHECK (reason IN ('cancelRefund', 'returnRefund', 'saleRefund', 'sellBack', 'spent')),
    order_number text,
    note         text,
    at           timestamptz NOT NULL
);
CREATE INDEX wallet_entries_user_idx ON wallet_entries (user_id, at DESC, id DESC);

CREATE TABLE points_entries (
    id           bigserial PRIMARY KEY,
    user_id      text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    points       integer NOT NULL,
    reason       text NOT NULL,
    order_number text,
    at           timestamptz NOT NULL
);
CREATE INDEX points_entries_user_idx ON points_entries (user_id, at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS points_entries;
DROP TABLE IF EXISTS wallet_entries;
DROP TABLE IF EXISTS order_returns;
DROP TABLE IF EXISTS order_history;
DROP TABLE IF EXISTS order_lines;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS coupons;
