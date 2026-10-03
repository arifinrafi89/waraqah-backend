-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users (
    id            text PRIMARY KEY,
    email         citext UNIQUE,
    phone         text NOT NULL DEFAULT '',
    name          text NOT NULL DEFAULT '',
    password_hash text,
    role          text NOT NULL DEFAULT 'reader'
                  CHECK (role IN ('reader', 'moderator', 'catalogManager', 'support', 'superAdmin')),
    photo_url     text,
    area          text NOT NULL DEFAULT '',
    district      text NOT NULL DEFAULT '',
    member_since  timestamptz NOT NULL DEFAULT now(),
    strikes       integer NOT NULL DEFAULT 0,
    banned        boolean NOT NULL DEFAULT false,
    deleted_at    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE pending_signups (
    contact       citext PRIMARY KEY,
    name          text NOT NULL,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE otp_codes (
    contact    citext NOT NULL,
    purpose    text NOT NULL CHECK (purpose IN ('signup', 'reset')),
    code_hash  text NOT NULL,
    expires_at timestamptz NOT NULL,
    attempts   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (contact, purpose)
);

CREATE TABLE refresh_tokens (
    id          text PRIMARY KEY,
    user_id     text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  text NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

-- Order numbers WQ-<n> and handled sale ids HS-<n> (BACKEND_PLAN.md section 4.3).
CREATE SEQUENCE order_number_seq START 100231;
CREATE SEQUENCE handled_sale_seq START 201;

-- +goose Down
DROP SEQUENCE IF EXISTS handled_sale_seq;
DROP SEQUENCE IF EXISTS order_number_seq;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS pending_signups;
DROP TABLE IF EXISTS users;
