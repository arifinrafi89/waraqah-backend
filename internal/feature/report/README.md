# report (reports and blocks)

- **Frontend files:** `lib/features/report/data/sources/{report_fake_api,report_fake_store}.dart`, `report_rules.dart` (`Check`).
- **Endpoints (me):** `POST /reports` (limited per reader to `REPORT_RATE_PER_HOUR`), `GET /blocks`, `POST /blocks/add|remove`.
- **Tables:** `reports`, `blocks` (migration `0008`).
- A report is of a listing, a reader, a message, a Bite, a comment or a review. The reader earlier open report on the same thing is returned instead of a second one. Listings and readers must exist and not be the reader own; the other kinds are checked when a moderator acts.
- `Service.IsBlocked` is `blocks.Checker`: true when either of the two readers blocked the other.
