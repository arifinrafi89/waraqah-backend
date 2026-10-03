# notifications

What happened to a reader. The server stores a kind and its params, never the words: the app builds the text from its ARB files.

- **Frontend files:** `lib/features/notifications/data/sources/{notification_fake_api,notification_fake_store,notification_seed,notification_sends,notification_sale_sends,notification_community_sends}.dart`, `notification_kind.dart`.
- **Endpoints:** `GET /notifications`, `POST /notifications/read`, `POST /notifications/read-all`, `GET /notifications/live` (SSE, one `{"unread": n}` per change).
- **Table:** `notifications` (migration `0002`), indexed by `(user_id, created_at desc)`. Queries: `db/queries/notifications.sql`.
- **Other features use `Sender`** (`Service.Send`): it skips a group the reader muted in Settings (moderation cannot be muted) and publishes the unread count. `sends.go` has one function per event (`OrderChanged`, `SaleSettledNotice`, ...) with the same kinds and params as the Dart senders; `notifications_test.go` ports their tests.
- A guest gets an empty list; the live stream needs a token.
