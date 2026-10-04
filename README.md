# Waraqah Backend

Go + PostgreSQL API for the [Waraqah](https://github.com/arifinrafi89/waraqah-frontend) Flutter app. It serves the same endpoints the app's fake API answers today, so the app switches over by setting one address.

- **Plan:** [`BACKEND_PLAN.md`](BACKEND_PLAN.md) — stack, API contract, database, phases, owners.
- **Tasks:** [`TASKS.md`](TASKS.md) — the 19 tasks that build the backend, in order, with their status.
- **Agents:** [`AGENTS.md`](AGENTS.md) — rules for coding agents (and a good summary for people).
- **Settings:** [`.env.example`](.env.example) — every environment variable with placeholders.

## Status

All 174 endpoints of the app's fake API are built (tasks T01–T18), and every contract golden replays against them (`make test`). It is deployed on Render with a Neon database (`render.yaml`, [`docs/deploy.md`](docs/deploy.md)), seeded, and `make smoke` passes against it. Still to do on the frontend side: F4, F6 and the frontend half of F7.

## Getting started

```bash
git clone https://github.com/arifinrafi89/waraqah-backend.git
git clone https://github.com/arifinrafi89/waraqah-frontend.git    # read-only reference, next to the backend
cd waraqah-backend
cp .env.example .env                                               # then fill in values (JWT_SECRET, SEED_DEMO_PASSWORD, ...)
docker compose up -d db db-test                                    # Postgres for the app (5432) and the tests (5433)
make dev                                                           # migrations, then the API on :8080
make seed                                                          # once: demo accounts and data (run again any time)
make check                                                         # what CI runs
make smoke                                                         # with the API running: one call per feature
```

Demo accounts: `reader@waraqah.test`, `admin@`, `moderator@`, `catalog@`, `support@waraqah.test`, all with `SEED_DEMO_PASSWORD`. In development the sign-up code `OTP_DEV_CODE` works and OTP emails go to the log.

Point the app at it (from `../waraqah-frontend`):

```bash
flutter run --dart-define=API_BASE_URL=http://localhost:8080/v1
```

(Android emulator: `http://10.0.2.2:8080/v1`.)
