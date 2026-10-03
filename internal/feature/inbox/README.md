# inbox

Chat of buyers and sellers of used books: threads, offers, the deal that follows and the ratings after a sale, with live updates.

- **Frontend files:** `lib/features/inbox/data/sources/{inbox_fake_api,inbox_fake_store,inbox_fake_actions,inbox_fake_selling,inbox_fake_rating,inbox_fake_replies,inbox_fake_json,inbox_fake_seed,inbox_live_source}.dart`; rules `offer_rules.dart` and `rating_rules.dart` (`rules.go`, ported with their tests).
- **Endpoints (me):** `GET /inbox` (`listingId`), `GET /inbox/thread`, `GET /inbox/live` (SSE), `POST /inbox/open|send|offer|offer/decide|read|listing/release|listing/sold|rate`.
- **Tables:** `threads` (unique `(listing_id, buyer_id)`; read markers are message positions), `messages`, `offers` (migration `0009`).
- **Per viewer:** `role`, `unread`, `from` (me, them, system) and the ratings come from the token.
- **Deal:** accepting an offer reserves the book for that buyer (through `p2p.Status`) and tells the seller other threads; `release` puts it on sale again; `sold` closes the other waiting offers; `rate` is allowed once each, after the sale. Blocks (either way) stop messages, offers and accepting (`blocked_reader`); a blocked buyer's offer can still be declined.
- **Live:** every change publishes `{seq, threadId, listingId}` on `inbox:<buyer>` and `inbox:<seller>` after the commit.
- **Demo bot** (`DEMO_MODE`, `DEMO_BOT_DELAY`): when a real reader writes or makes an offer to a seeded demo reader (ids `p-...`), that reader answers once per thread after the delay; a demo buyer rates the seller after a sale. Accounts that signed up never answer. A delay of 0 answers at once (the tests use it).
- **Moderation:** `Messages` is registered as the `message` subject, so a moderator can look up and remove a reported message.
