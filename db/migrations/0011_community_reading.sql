-- +goose Up
-- Book-Bites: short posts that may tag a catalog Book. Deleting a Bite deletes its likes and
-- comments. position grows with every new Bite, so a feed is newest first.
CREATE TABLE bites (
    id         text PRIMARY KEY,
    position   bigserial NOT NULL,
    author_id  text NOT NULL REFERENCES users (id),
    text       text NOT NULL,
    book_id    text,
    spoiler    boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL,
    edited_at  timestamptz
);
CREATE INDEX bites_feed_idx ON bites (created_at DESC, position DESC);
CREATE INDEX bites_author_idx ON bites (author_id, created_at DESC);
CREATE INDEX bites_book_idx ON bites (book_id, created_at DESC) WHERE book_id IS NOT NULL;

CREATE TABLE bite_likes (
    bite_id text NOT NULL REFERENCES bites (id) ON DELETE CASCADE,
    user_id text NOT NULL REFERENCES users (id),
    PRIMARY KEY (bite_id, user_id)
);
CREATE INDEX bite_likes_user_idx ON bite_likes (user_id);

-- One level of replies: a reply's parent is always a top comment.
CREATE TABLE bite_comments (
    id         text PRIMARY KEY,
    position   bigserial NOT NULL,
    bite_id    text NOT NULL REFERENCES bites (id) ON DELETE CASCADE,
    parent_id  text REFERENCES bite_comments (id) ON DELETE CASCADE,
    author_id  text NOT NULL REFERENCES users (id),
    text       text NOT NULL,
    created_at timestamptz NOT NULL
);
CREATE INDEX bite_comments_bite_idx ON bite_comments (bite_id, position);
CREATE INDEX bite_comments_parent_idx ON bite_comments (parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX bite_comments_author_idx ON bite_comments (author_id);

-- One review per reader per Book. seed_verified is the Verified Purchase of a seeded reader;
-- everyone else's is worked out from their delivered orders.
CREATE TABLE reviews (
    id            text NOT NULL UNIQUE,
    book_id       text NOT NULL,
    user_id       text NOT NULL REFERENCES users (id),
    stars         integer NOT NULL CHECK (stars BETWEEN 1 AND 5),
    text          text NOT NULL DEFAULT '',
    seed_verified boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL,
    edited_at     timestamptz,
    PRIMARY KEY (book_id, user_id)
);
CREATE INDEX reviews_book_idx ON reviews (book_id, created_at DESC);
CREATE INDEX reviews_user_idx ON reviews (user_id);

CREATE TABLE follows (
    follower_id text NOT NULL REFERENCES users (id),
    followee_id text NOT NULL REFERENCES users (id),
    at          timestamptz NOT NULL,
    PRIMARY KEY (follower_id, followee_id)
);
CREATE INDEX follows_followee_idx ON follows (followee_id);

-- position keeps the order Books were put on a shelf, for Books added at the same moment.
CREATE TABLE shelf_entries (
    user_id     text NOT NULL REFERENCES users (id),
    position    bigserial NOT NULL,
    book_id     text NOT NULL,
    shelf       text NOT NULL CHECK (shelf IN ('wantToRead', 'reading', 'finished')),
    added_at    timestamptz NOT NULL,
    finished_at timestamptz,
    progress    integer NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
    pages_read  integer,
    total_pages integer,
    PRIMARY KEY (user_id, book_id)
);
CREATE INDEX shelf_entries_user_idx ON shelf_entries (user_id, added_at DESC, position);

-- A delivered Book goes on Want to Read once; taking it off later is respected.
CREATE TABLE shelf_order_sync (
    user_id text NOT NULL REFERENCES users (id),
    book_id text NOT NULL,
    PRIMARY KEY (user_id, book_id)
);

-- The days a reader read (Dhaka dates), for the streak.
CREATE TABLE reading_days (
    user_id text NOT NULL REFERENCES users (id),
    day     date NOT NULL,
    PRIMARY KEY (user_id, day)
);

CREATE TABLE reading_goals (
    user_id text NOT NULL REFERENCES users (id),
    year    integer NOT NULL,
    goal    integer NOT NULL CHECK (goal BETWEEN 1 AND 365),
    PRIMARY KEY (user_id, year)
);

-- +goose Down
DROP TABLE IF EXISTS reading_goals;
DROP TABLE IF EXISTS reading_days;
DROP TABLE IF EXISTS shelf_order_sync;
DROP TABLE IF EXISTS shelf_entries;
DROP TABLE IF EXISTS follows;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS bite_comments;
DROP TABLE IF EXISTS bite_likes;
DROP TABLE IF EXISTS bites;
