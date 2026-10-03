# moderation

The Moderation Center: listing queue, reports, strikes and bans, and the audit log. All endpoints need the moderate permission (moderator, super admin).

- **Frontend files:** `lib/features/moderation/data/sources/{moderation_fake_api,moderation_fake_store,moderation_fake_reports,moderation_subjects}.dart`, `moderation_rules.dart` (`rules.go`).
- **Endpoints:** `GET /moderation/listings`, `POST /moderation/listings/decide`, `GET /moderation/reports`, `POST /moderation/reports/act`, `GET /moderation/log`.
- **Tables:** `moderation_log` (migration `0008`); strikes and bans live on `users`. A reader at 3 strikes is banned and can no longer sign in.
- **Decide:** approve, request changes or reject (a reason is needed for the last two). The seller is told, and the log records the staff name from the token; the `by` in the body is ignored.
- **Act:** dismiss, remove, warn or ban. Every open report about the same thing ends with it. Removing a listing takes it down and deletes its photos. Removing a message, Bite, comment or review goes through the `Subject` registry: inbox (T15), bites and reviews (T17) call `Register(kind, subject)` in `deps.go`. Until then a report on such a thing shows its id.
- `Service.IsBanned` is `moderation.Bans`.
