-- +goose Up
CREATE TABLE categories (
    id      text PRIMARY KEY,
    section text NOT NULL,
    name_en text NOT NULL,
    name_bn text NOT NULL
);

CREATE TABLE authors (
    id      text PRIMARY KEY,
    name    text NOT NULL,
    name_bn text,
    bio     text
);

CREATE TABLE publishers (
    id      text PRIMARY KEY,
    name    text NOT NULL,
    name_bn text
);

CREATE TABLE subjects (
    id      text PRIMARY KEY,
    name_en text NOT NULL,
    name_bn text NOT NULL
);

-- position keeps the storefront order: the order of /books when nothing sorts it.
CREATE TABLE books (
    id                text PRIMARY KEY,
    position          bigserial NOT NULL,
    title             text NOT NULL,
    title_bn          text,
    short_title       text,
    author            text NOT NULL,
    author_id         text NOT NULL,
    publisher_id      text NOT NULL,
    category_id       text NOT NULL,
    section           text NOT NULL,
    original_language text NOT NULL,
    added_at          timestamptz NOT NULL,
    rating            numeric(2, 1) NOT NULL DEFAULT 0,
    tags              text[] NOT NULL DEFAULT '{}',
    cover_seed        integer NOT NULL DEFAULT 0,
    hidden            boolean NOT NULL DEFAULT false,
    classes           integer[] NOT NULL DEFAULT '{}',
    exams             text[] NOT NULL DEFAULT '{}',
    subject_id        text
);
CREATE INDEX books_position_idx ON books (position);
CREATE INDEX books_author_idx ON books (author_id);
CREATE INDEX books_publisher_idx ON books (publisher_id);
CREATE INDEX books_category_idx ON books (category_id);
CREATE INDEX books_section_idx ON books (section);

CREATE TABLE editions (
    id             text PRIMARY KEY,
    book_id        text NOT NULL REFERENCES books (id) ON DELETE CASCADE,
    position       bigserial NOT NULL,
    format         text NOT NULL,
    language       text NOT NULL,
    price_bdt      integer NOT NULL,
    list_price_bdt integer,
    stock          integer NOT NULL,
    is_preorder    boolean NOT NULL DEFAULT false,
    isbn           text
);
CREATE INDEX editions_book_idx ON editions (book_id, position);
CREATE INDEX editions_isbn_idx ON editions (isbn) WHERE isbn IS NOT NULL;

CREATE TABLE book_details (
    book_id     text PRIMARY KEY REFERENCES books (id) ON DELETE CASCADE,
    description text NOT NULL,
    pages       integer NOT NULL
);

CREATE TABLE look_inside (
    book_id      text PRIMARY KEY REFERENCES books (id) ON DELETE CASCADE,
    contents     jsonb NOT NULL,
    sample_pages jsonb NOT NULL
);

-- A series lists every title, also the ones Waraqah does not sell, so entries are one jsonb value.
CREATE TABLE series (
    id       text PRIMARY KEY,
    position bigserial NOT NULL,
    name     text NOT NULL,
    entries  jsonb NOT NULL
);

CREATE TABLE book_questions (
    id         text PRIMARY KEY,
    position   bigserial NOT NULL,
    book_id    text NOT NULL REFERENCES books (id) ON DELETE CASCADE,
    user_id    text REFERENCES users (id) ON DELETE SET NULL,
    asker_name text NOT NULL,
    text       text NOT NULL,
    asked_at   timestamptz NOT NULL
);
CREATE INDEX book_questions_book_idx ON book_questions (book_id, position);

CREATE TABLE book_answers (
    id          text PRIMARY KEY,
    position    bigserial NOT NULL,
    question_id text NOT NULL REFERENCES book_questions (id) ON DELETE CASCADE,
    user_id     text REFERENCES users (id) ON DELETE SET NULL,
    author_name text NOT NULL,
    text        text NOT NULL,
    answered_at timestamptz NOT NULL,
    is_staff    boolean NOT NULL DEFAULT false
);
CREATE INDEX book_answers_question_idx ON book_answers (question_id, position);

CREATE TABLE experts (
    id            text PRIMARY KEY,
    position      bigserial NOT NULL,
    name          text NOT NULL,
    name_bn       text NOT NULL,
    credential_en text NOT NULL,
    credential_bn text NOT NULL,
    kind          text NOT NULL,
    verified      boolean NOT NULL DEFAULT false
);

CREATE TABLE collections (
    id       text PRIMARY KEY,
    position bigserial NOT NULL,
    title_en text NOT NULL,
    title_bn text NOT NULL,
    note_en  text NOT NULL,
    note_bn  text NOT NULL,
    section  text,
    expert_id text,
    book_ids text[] NOT NULL DEFAULT '{}'
);

-- owner_id is null for Staff lists; a reader own list belongs to the reader.
CREATE TABLE booklists (
    id         text PRIMARY KEY,
    position   bigserial NOT NULL,
    owner_id   text REFERENCES users (id) ON DELETE CASCADE,
    kind       text NOT NULL,
    title_en   text NOT NULL,
    title_bn   text NOT NULL,
    note_en    text,
    note_bn    text,
    book_ids   text[] NOT NULL DEFAULT '{}',
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX booklists_owner_idx ON booklists (owner_id);

-- The lowest price an Edition had in the last 30 days (the "lowest price" badge).
CREATE TABLE price_lows (
    edition_id text PRIMARY KEY REFERENCES editions (id) ON DELETE CASCADE,
    low_bdt    integer NOT NULL,
    since      timestamptz NOT NULL
);

-- Copies sold per Edition per month, for the bestselling sort and the dashboard.
CREATE TABLE sales_by_month (
    edition_id text NOT NULL REFERENCES editions (id) ON DELETE CASCADE,
    month      date NOT NULL,
    copies     integer NOT NULL DEFAULT 0,
    PRIMARY KEY (edition_id, month)
);

-- +goose Down
DROP TABLE IF EXISTS sales_by_month;
DROP TABLE IF EXISTS price_lows;
DROP TABLE IF EXISTS booklists;
DROP TABLE IF EXISTS collections;
DROP TABLE IF EXISTS experts;
DROP TABLE IF EXISTS book_answers;
DROP TABLE IF EXISTS book_questions;
DROP TABLE IF EXISTS series;
DROP TABLE IF EXISTS look_inside;
DROP TABLE IF EXISTS book_details;
DROP TABLE IF EXISTS editions;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS publishers;
DROP TABLE IF EXISTS authors;
DROP TABLE IF EXISTS categories;
