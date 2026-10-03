-- +goose Up
-- A profile photo is kept as the base64 JPEG the app sends, because the app reads it back
-- from /profile as base64 (ProfileDetailsModel.photo).
ALTER TABLE users ADD COLUMN photo_data text;

CREATE TABLE profile_prefs (
    user_id          text PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    muted            text[] NOT NULL DEFAULT '{}',
    profile_visible  boolean NOT NULL DEFAULT true,
    activity_visible boolean NOT NULL DEFAULT true
);

CREATE TABLE addresses (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    position   bigserial NOT NULL,
    label      text NOT NULL,
    recipient  text NOT NULL,
    phone      text NOT NULL,
    line       text NOT NULL,
    upazila    text NOT NULL,
    district   text NOT NULL,
    division   text NOT NULL,
    is_default boolean NOT NULL DEFAULT false
);
-- One default address per reader.
CREATE UNIQUE INDEX addresses_one_default_idx ON addresses (user_id) WHERE is_default;
CREATE INDEX addresses_user_position_idx ON addresses (user_id, position);

CREATE TABLE notifications (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text NOT NULL,
    params     jsonb NOT NULL DEFAULT '{}',
    target     jsonb,
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS profile_prefs;
ALTER TABLE users DROP COLUMN IF EXISTS photo_data;
