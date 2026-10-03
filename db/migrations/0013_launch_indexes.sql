-- +goose Up
-- Launch hardening (T19): every foreign key gets an index (plan §7.2). blocks.blocked_id serves the
-- "blocked either way" checks of feeds, chats and follows; alerts.book_id the sweep after a price or
-- stock change; reports.reporter_id the per-reader duplicate check; the rest keep account deletion
-- and joins cheap.
CREATE INDEX IF NOT EXISTS blocks_blocked_idx ON blocks (blocked_id);
CREATE INDEX IF NOT EXISTS alerts_book_idx ON alerts (book_id);
CREATE INDEX IF NOT EXISTS reports_reporter_idx ON reports (reporter_id, kind, target_id);
CREATE INDEX IF NOT EXISTS ratings_from_idx ON ratings (from_id);
CREATE INDEX IF NOT EXISTS listings_buyer_idx ON listings (buyer_id) WHERE buyer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS wishlist_items_book_idx ON wishlist_items (book_id);
CREATE INDEX IF NOT EXISTS messages_offer_idx ON messages (offer_id) WHERE offer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS book_questions_user_idx ON book_questions (user_id);
CREATE INDEX IF NOT EXISTS book_answers_user_idx ON book_answers (user_id);
CREATE INDEX IF NOT EXISTS coupons_created_by_idx ON coupons (created_by);

-- +goose Down
DROP INDEX IF EXISTS coupons_created_by_idx;
DROP INDEX IF EXISTS book_answers_user_idx;
DROP INDEX IF EXISTS book_questions_user_idx;
DROP INDEX IF EXISTS messages_offer_idx;
DROP INDEX IF EXISTS wishlist_items_book_idx;
DROP INDEX IF EXISTS listings_buyer_idx;
DROP INDEX IF EXISTS ratings_from_idx;
DROP INDEX IF EXISTS reports_reporter_idx;
DROP INDEX IF EXISTS alerts_book_idx;
DROP INDEX IF EXISTS blocks_blocked_idx;
