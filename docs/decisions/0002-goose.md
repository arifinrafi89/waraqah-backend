# goose v3

**What:** Runs the numbered SQL migrations in db/migrations, embedded in the binary (github.com/pressly/goose/v3).

**Why not the standard library:** The standard library has no migration runner; goose keeps plain SQL files with Up and Down sections.
