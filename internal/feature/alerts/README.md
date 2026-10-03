# alerts

Price-drop and back-in-stock alerts. Each alert is checked against the catalog every time the list is read; the `Sweeper` runs after Staff change prices or stock and tells the reader once, when an alert first fires.

- **Frontend files:** `lib/features/alerts/data/sources/{alert_fake_api,alert_fake_store}.dart`.
- **Endpoints:** `GET /alerts`, `POST /alerts/set|remove`.
- **Table:** `alerts` with `fired_at` (migration `0005`). Setting an alert replaces one of the same kind on the same Edition; one that already fires when set is marked fired without a notification (the page shows it).
- `Service.Sweep` implements the sweeper that `catalogadmin` calls (wired in `internal/app/deps.go`) and sends the notification through `notifications.Sender`.
