# Waraqah Backend

Go + PostgreSQL API for the [Waraqah](https://github.com/arifinrafi89/waraqah-frontend) Flutter app. It serves the same endpoints the app's fake API answers today, so the app switches over by setting one address.

- **Plan:** [`BACKEND_PLAN.md`](BACKEND_PLAN.md) — stack, API contract, database, phases, owners.
- **Agents:** [`AGENTS.md`](AGENTS.md) — rules for coding agents (and a good summary for people).
- **Settings:** [`.env.example`](.env.example) — every environment variable with placeholders.

## Status

Planning. Nothing is built yet; Phase 0 (foundations) is next (plan §21).

## Getting started (once Phase 0 lands)

```bash
git clone https://github.com/arifinrafi89/waraqah-backend.git
git clone https://github.com/arifinrafi89/waraqah-frontend.git    # read-only reference, next to the backend
cd waraqah-backend
cp .env.example .env                                               # then fill in values
make dev                                                          # Postgres in Docker, migrations, API on :8080
```

Point the app at it (from `../waraqah-frontend`):

```bash
flutter run --dart-define=API_BASE_URL=http://localhost:8080/v1
```

(Android emulator: `http://10.0.2.2:8080/v1`.)
