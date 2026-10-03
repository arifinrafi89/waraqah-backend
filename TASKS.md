# Waraqah Backend — Tasks

> The backend in **19 tasks**, from an empty repo to a deployed API that the app uses in place of its fake API. Work through them in order, because each task builds on the ones before it. **Update this file whenever a task changes state** (see [How to update this file](#how-to-update-this-file)).

Read with: [`AGENTS.md`](AGENTS.md) (rules), [`BACKEND_PLAN.md`](BACKEND_PLAN.md) (the plan; § numbers below point into it). If a task and the plan disagree, the plan wins; fix this file in the same PR.

**Frontend paths.** Every frontend path below is relative to the frontend checkout, `FRONTEND_DIR` (default `../waraqah-frontend`, the sibling folder next to this repo; plan §3). `lib/…` and `test/…` mean `$FRONTEND_DIR/lib/…` and `$FRONTEND_DIR/test/…`. The frontend is read-only from here; frontend changes (the **F** tasks, plan §20) are separate PRs in the frontend repo.

**Endpoint count.** Appendix A lists **174 endpoints**. Each one belongs to exactly one task below: T07 6 · T08 14 · T09 25 · T10 29 · T11 13 · T12 14 · T13 5 · T14 15 · T15 16 · T16 16 · T17 18 · T18 3 = **174**.

---

## Progress

| # | Task | Plan tickets | Owner | Status |
|---|---|---|---|---|
✅ done
✅ done
✅ done
✅ done
✅ done
✅ done
🟡 in progress
✅ done
✅ done
✅ done
✅ done
| T12 | Checkout, orders, wallet and points | B3.2, B3.3, B3.4 | Farhan | ⬜ todo |
| T13 | Donate and donation places admin | B3.5 | Farhan (admin: Arifin) | ⬜ todo |
| T14 | Listings, reports and blocks, moderation | B4.1, B4.2, B4.3 | Arifin | ⬜ todo |
| T15 | Inbox and book requests | B4.4, B4.5 | Farhan (requests: Arifin) | ⬜ todo |
| T16 | Handled sales, Sell Back and Certified Used | B4.6, B4.7 | Arifin | ⬜ todo |
| T17 | Bites, reviews, readers and shelves | B5.1, B5.2, B5.3 | Rahinur (shelves: Arifin) | ⬜ todo |
| T18 | AI assistant and admin dashboard | B6.1, B6.2, F4 | Arifin | ⬜ todo |
| T19 | Hardening, deployment and launch | Phase 7, F6, F7 | everyone | ⬜ todo |

Status values: ⬜ todo · 🟡 in progress · 🔵 in review (PR open) · ✅ done.

---

## How to update this file

- **Starting a task:** set its status to 🟡 in the table and in the task's `Status:` line, add the branch name, and commit the change on that branch.
- **Opening the PR:** set 🔵 and add the PR link to the task's `Status:` line.
- **After merge:** set ✅, tick the task's **Done when** boxes, and write 1–3 lines under **Notes** about anything that differs from the plan (a renamed refusal code, a deferred endpoint, a new env variable). If something is deferred, name the task that picks it up.
- Large tasks may be split across several PRs (one per feature package is a good size). List each PR under **Notes**; the task stays 🟡 until the last one merges.
- Never delete a task. If the plan changes, edit the task and say so under **Notes**.

---

## Conventions every feature task follows

These apply to T07–T18 and are not repeated in each task. The details are in AGENTS.md's "Workflow for one ticket" and plan §6, §15 and §22.

1. **Read the contract first:** the feature's rows in plan Appendix A, its fake API file, its fake store (Appendix C), its Dart models in `lib/features/<feature>/data/models/`, and its rules class and tests (Appendix B).
2. **Build in layers:** migration `db/migrations/NNNN_<feature>_<what>.sql` → queries `db/queries/<feature>.sql` → `make sqlc` → `store.go` → `rules.go` and ported `rules_test.go` → `service.go` → `handlers*.go` → `routes.go` (one line per endpoint, commented with the fake API constant) → mount the feature in `internal/app/routes.go` and wire its interfaces in `internal/app/deps.go`.
3. **DTOs mirror the Dart models:** every field, camelCase `json` tags, enums as Dart value names, money as `int` taka, RFC 3339 dates in `APP_TIMEZONE` (§4.3). The type name is the model name without `Model`.
4. **Auth level per row** (Appendix A): `public`, `me` (GET without a token answers the empty value, §5.4), `staff`, `staff:catalog|orders|moderate`. Public endpoints still read an optional token so per-viewer fields (`isMine`, `isMyDeal`, …) are correct.
5. **Refusals and misses:** `200 null` via `httpx.Null` / `httpx.Refuse(w, code)`. Every code is added to `internal/platform/httpx/errors.go` and `docs/error-codes.md` with the Dart rule it mirrors.
6. **Seed:** extend `cmd/seed` to load the feature's `seed/*.json` (idempotent upserts) so the app shows the same data it shows on the fake API.
7. **Contract tests:** every endpoint in the task passes its golden in `testdata/contract/` (T06).
8. **App walk:** run the app against the local backend (`flutter run --dart-define=API_BASE_URL=http://localhost:8080/v1`, Android emulator `http://10.0.2.2:8080/v1`) and use the feature's screens signed in **and** signed out. An error state is a bug.
9. **Finish:** `make check` passes, the feature's `README.md` (5–10 lines: what it owns, its frontend files, its tables) is written, and this file is updated.

---

## T01 — Repo skeleton, tooling and CI

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/3 · **Owner:** Arifin · **Plan:** B0.1, §2, §6.1, §19, §22.4 · **Branch:** `feature/repo-skeleton` · **Depends on:** nothing

**Goal:** an empty but complete project layout where `make check` runs green in CI, so every later task only adds code.

**Steps**
1. Create the folders from plan §6.1: `cmd/{api,seed,admin}`, `internal/{app,platform,feature}`, `db/{migrations,queries}`, `seed/`, `testdata/contract/`, `docs/decisions/`, `scripts/`. Add `.gitkeep` files where a folder is still empty.
2. `cmd/api/main.go`: a minimal server on `PORT` answering `GET /healthz` with `{"ok":true}` (T02 replaces it with the full wiring).
3. Add the dependencies the plan names to `go.mod`: `github.com/jackc/pgx/v5`, `github.com/pressly/goose/v3`, `github.com/joho/godotenv`, `github.com/golang-jwt/jwt/v5`, `golang.org/x/crypto` (bcrypt), `github.com/oklog/ulid/v2`, `github.com/rivo/uniseg`, `google.golang.org/genai`. Pin sqlc and golangci-lint as tools (`go tool` directives or pinned versions in the Makefile). **Write one ADR per dependency** in `docs/decisions/NNNN-<name>.md` (AGENTS.md rule 11), each a few lines: what it is for, and why the standard library isn't enough.
4. `docker-compose.yml`: `db` (Postgres 16, port 5432, user/password/db `waraqah`) and `db-test` (port 5433, db `waraqah_test`, to match `DATABASE_URL_TEST` in `.env.example`). Both enable `citext` and `pg_trgm` (an init script in `scripts/db-init.sql`).
5. `Makefile` with every target from plan §22.4: `dev`, `test`, `check`, `migrate`, `migrate-down-up`, `sqlc`, `seed`, `frontend-sync`, `contract-export`, `set-role`, `smoke`, plus `logs`. Targets whose code comes later print `not yet: see TASKS.md T0x` instead of failing silently. `FRONTEND_DIR` is read from `.env` with a default of `../waraqah-frontend`.
6. `.golangci.yml` with `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`, `gofmt`, `misspell`, plus a file-length check (`lll` doesn't do this: use a short `scripts/check-file-length.sh` that fails on any non-generated `.go` file over 300 lines, as AGENTS.md rule 7 requires).
7. `.github/workflows/ci.yml`: Postgres 16 service container, then `gofmt -l` (empty), `go vet ./...`, golangci-lint, the file-length check, `sqlc diff` (once T03 adds sqlc), migrations up/down, `go test ./...`.
8. `docs/error-codes.md` (a header and an empty table: code · meaning · Dart rule mirrored · endpoints) and `docs/contract-changes.md` (header plus "v1 = fake API at frontend PR #144").
9. Update `README.md` "Getting started" if a command changed.

**Done when**
- [x] `make check` passes locally and in CI on the PR
- [x] `docker compose up -d db db-test` starts both databases with `citext` and `pg_trgm`
- [x] Every dependency has an ADR in `docs/decisions/`

**Notes:** Layout, stub server, Makefile, lint config, docker-compose (two databases), CI and one ADR per dependency. Migrations are embedded from a package in `db/` and run with `go run ./cmd/api -migrate up`. Gemini will use REST, so no genai dependency (ADR 0008). sqlc and golangci-lint are installed as tools, not in go.mod.

---

## T02 — Config, logging and the HTTP toolkit

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/4 · **Owner:** Arifin (config review: Rahinur) · **Plan:** B0.2, §4.2, §4.6, §16, §17, §18 · **Branch:** `feature/http-platform` · **Depends on:** T01

**Goal:** the shared HTTP behaviour every endpoint relies on: the `200 null` refusal, the error shape, request IDs, logs, CORS, body limits, and health endpoints.

**Steps**
1. **`internal/platform/config`**: a `Config` struct with one field per variable in `.env.example` (all of them: App, Logging, Database, Auth, Email, Cloudinary, Gemini, Rate limits, Demo/jobs, Seeding, Local tools). Load `.env` with godotenv only when `APP_ENV=development`. Parse durations (`15m`, `720h`), ints and booleans, and comma lists (`CORS_ALLOWED_ORIGINS`, `GOOGLE_OAUTH_CLIENT_IDS`). Fail at start with a message naming the variable when a production-required value (§18) is missing. In production, refuse to start when `OTP_DEV_CODE` is non-empty or `JWT_SECRET` is shorter than 32 bytes.
2. A test, `config_env_test.go`, that parses `.env.example` and fails if a struct field has no entry or an entry has no field (plan §18: "CI checks that every field has an entry").
3. **`internal/platform/logx`**: a `slog` logger, JSON or text by `LOG_FORMAT`, level by `LOG_LEVEL`; `logx.From(ctx)` returns the request-scoped logger carrying `request_id`, and once T04 lands, `user_id` and `role` too.
4. **`internal/platform/httpx`**:
   - `JSON(w, v)` (status 200, `application/json`), `Null(w)` (writes `null`), `Refuse(w, r, code)` (sets `X-Waraqah-Error: <code>`, writes `null`, logs `refusal=<code>`).
   - `Error(w, r, status, code, msg)` writes `{"error":{"code","message","requestId"}}` with the codes `unauthorized` 401, `forbidden` 403, `not_found` 404, `bad_request` 400, `too_large` 413, `internal` 500 (details only in logs).
   - `Decode(r, &v)` for JSON bodies (invalid JSON → 400) and small query helpers (`Query(r, "id")`, `QueryBool`, `QueryInt`).
   - `errors.go`: typed refusal code constants, empty for now; features add theirs.
   - Middleware, outermost first: recover (500 + stack in the log) → request ID (keep the client's `X-Request-ID` or make one; echo it back) → access log (`method`, `route`, `status`, `latency_ms`, `refusal`) → CORS (only `CORS_ALLOWED_ORIGINS`; allow the `Authorization`, `Content-Type` and `X-Request-ID` headers; expose `X-Waraqah-Error` and `X-Request-ID`) → body limit (`MAX_REQUEST_BODY_MB` → 413).
   - Unknown paths under `API_BASE_PATH` answer `404` in the error shape, not Go's plain-text 404.
5. **`internal/app/routes.go`**: one `http.ServeMux`. Features mount with `mux.HandleFunc("POST /v1/…", h.X)` using `API_BASE_PATH` as the prefix. `GET /healthz`, `GET /readyz` (a DB ping once T03 lands; until then always OK) and `GET /version` (`{commit, builtAt, contract: "v1"}` from `-ldflags`) have no prefix.
6. **`cmd/api/main.go`**: config → logger → (db, T03) → deps → routes → `http.Server` with sensible timeouts (ReadHeaderTimeout; no WriteTimeout because of SSE) and graceful shutdown on SIGINT/SIGTERM.
7. Tests: `Null` and `Refuse` bodies and headers, error shape, request-ID echo, 404 shape, 413 on a big body, CORS preflight.

**Done when**
- [x] `curl localhost:8080/healthz`, `/version` work; an unknown `/v1/x` answers the 404 error shape
- [x] `.env.example` ⇄ `Config` test passes
- [x] No token, password or body is logged (checked by reading the access log code)

**Notes:** Config struct with one field per .env.example variable (checked by a test), production safety checks, slog setup with a request-scoped logger, httpx (JSON, Null, Refuse, Error, Decode, middleware chain, Router), /healthz /readyz /version, unknown paths answer the 404 error shape. Notes: Decode takes the ResponseWriter and writes its own 400 or 413, returning false; logx.Annotate lets later middleware add user_id and role to the request log.

---

## T03 — Database layer, migrations and sqlc

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/5 · **Owner:** Arifin · **Plan:** B0.3, §7 · **Branch:** `feature/db-platform` · **Depends on:** T02

**Goal:** a pgx pool, a transaction helper, goose migrations and sqlc code generation, plus the first tables, which auth needs.

**Steps**
1. **`internal/platform/db`**: `Open(ctx, cfg)` builds a `pgxpool.Pool` from `DATABASE_URL` with `DB_MAX_CONNS`; `WithTx(ctx, func(q *Queries) error) error` runs commit/rollback; `DEBUG_SQL=true` adds a pgx tracer logging the SQL and its timing (development only, never with argument values).
2. **sqlc**: `db/sqlc.yaml` (engine `postgresql`, `sql_package: pgx/v5`, schema `db/migrations`, queries `db/queries`, output `internal/platform/db/sqlc`, `emit_json_tags: false`; DTOs are written by hand to match Dart). `make sqlc` generates it and CI runs `sqlc diff`.
3. **goose**: migrations embedded with `embed.FS`. `make migrate` runs `up` against `DATABASE_URL_DIRECT` (falling back to `DATABASE_URL` locally), and `make migrate-down-up` runs down-one then up-one. With `RUN_MIGRATIONS_ON_START=true`, `cmd/api` runs `up` before it listens.
4. **Migration `0001_platform_users.sql`**: `CREATE EXTENSION IF NOT EXISTS citext; … pg_trgm;`, then `users` (all columns from §7.1, `role` text with a `CHECK` over the five `UserRole` names, `email citext unique`), `pending_signups`, `otp_codes` (`contact`, `purpose` ∈ `signup|reset`, `code_hash`, `expires_at`, `attempts`), and `refresh_tokens` (`token_hash` unique, index on `user_id`). Add a `Down` section.
5. **ID sequences** in the same migration: `order_number_seq` (start 100231, for `WQ-…`) and `handled_sale_seq` (start 201, for `HS-…`). T05's `ids` package uses them.
6. **Test helper** `internal/platform/db/dbtest`: connects to `DATABASE_URL_TEST`, migrates once per test run, and gives each test a transaction that is rolled back at the end. Handler tests in every feature use it.
7. `/readyz` now pings the pool (`503` + error shape when the database is down).

**Done when**
- [x] `make migrate` and `make migrate-down-up` work on the local DB; CI runs them on a fresh Postgres
- [x] `make sqlc` generates code and `sqlc diff` is clean
- [x] `/readyz` is green locally and against a Neon branch

**Notes:** pgx pool with an optional SQL timing tracer, InTx and WithTx helpers, goose migrations embedded from db/ and run with the -migrate flag (guarded by an advisory lock), sqlc config and generated code, migration 0001 with users, pending sign-ups, OTP codes, refresh tokens and the two id sequences, dbtest helper running each test in a rolled-back transaction, /readyz now pings the database. Notes: sqlc is pinned to v1.29.0 and installed with CGO_ENABLED=0 go install; dbtest loads the repo .env so a plain go test works locally.

---

## T04 — Auth platform: tokens, OTP, roles, middleware

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/6 · **Owner:** Arifin · **Plan:** B0.4, §5, §17, Appendix B (roles) · **Branch:** `feature/auth-platform` · **Depends on:** T03

**Goal:** everything identity-related that features need, without any endpoints yet (those come in T07).

**Frontend reference:** `lib/features/auth/domain/entities/user_role.dart`, `test/admin_access_test.dart` (the roles part).

**Steps**
1. **`roles.go`**: port `UserRole` exactly: values `reader`, `moderator`, `catalogManager`, `support`, `superAdmin`; `IsStaff`, `CanModerate`, `CanManageCatalog`, `CanManageOrders`. Port the role cases of `admin_access_test.dart` as a table test.
2. **`jwt.go`**: HS256 access tokens with claims `sub`, `role`, `iat`, `exp`, `iss`=`JWT_ISSUER`, lifetime `JWT_ACCESS_TTL`. Parsing checks the algorithm, issuer and expiry.
3. **`refresh.go`**: 32 random bytes, base64url to the client, SHA-256 hash in `refresh_tokens`; `Issue`, `Rotate` (revokes the old token and issues a new one; reusing a revoked token revokes all of that user's tokens), `RevokeAll(userID)`. Lifetime `JWT_REFRESH_TTL`.
4. **`password.go`**: bcrypt at `BCRYPT_COST`.
5. **`otp.go`**: a 6-digit code from `crypto/rand`, stored as a SHA-256 hash with `OTP_TTL`; `OTP_MAX_ATTEMPTS` before the code dies; `OTP_RESEND_SECONDS` between sends per contact. `OTP_DEV_CODE` is accepted **only** when `APP_ENV=development`.
6. **`google.go`**: a `GoogleVerifier` interface. The real implementation fetches Google's JWKS (cache it by `Cache-Control`) and checks the signature, `iss` (`accounts.google.com` or `https://accounts.google.com`), `aud` ∈ `GOOGLE_OAUTH_CLIENT_IDS`, `exp` and `email_verified`. A fake implementation is used in tests.
7. **`middleware.go`**: one wrapper per level:
   - `Public(h)`: reads the token if present (bad token = treated as a guest, never a 401) so per-viewer fields work.
   - `Me(h)`: GET without a token reaches the handler with no user (the handler answers the empty value, §5.4). POST and SSE without a token → 401. Banned or deleted users → 401.
   - `Staff(h)` / `StaffCan(perm, h)`: no token → 401; reader or wrong role → 403; `superAdmin` passes all.
   - `auth.UserFrom(ctx)` returns `(User{ID, Role}, ok)`; also add `user_id` and `role` to the request logger.
8. Tests for each file, including expired and tampered JWTs, refresh reuse detection, the dev OTP refused outside development, and every middleware level × token state.

**Done when**
- [x] Roles behave exactly like `user_role.dart` (ported tests pass)
- [x] Middleware matrix test passes (public/me/staff/staff:perm × no token/reader/each staff role/bad token)
- [x] No secret or token appears in any log line (checked in tests with a captured logger)

**Notes:** Roles ported from user_role.dart, HS256 access tokens, rotating refresh tokens with theft detection, bcrypt, OTP with expiry, attempt limit and resend delay (dev code only in development), Google ID token verifier with a cached JWKS and a fake, and the middleware levels Public, Me, Authed (for SSE), Staff and StaffCan. Notes: the role used by middleware comes from the database, so a role change or ban applies on the next request; an expired or bad token on a Me endpoint answers 401 so the app can refresh, while a missing token on a GET answers the empty value. CI scripts are now executable.

---

## T05 — Shared services: SSE, clock, ids, images, email, AI, jobs, limits

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/7 · **Owner:** Arifin · **Plan:** B0.5, §4.5, §9, §11, §12, §13, §17 · **Branch:** `feature/shared-services` · **Depends on:** T02 (T03 for ids)

**Goal:** every platform service a feature might need, each behind an interface with a fake, so feature tasks never touch an outside service directly.

**Steps**
1. **`platform/clock`**: `Clock` interface (`Now()`), a real one and a settable one for tests; `Today(clock)` and `DayOf(t)` in `APP_TIMEZONE` (Asia/Dhaka) for streaks, seasons and "orders today".
2. **`platform/ids`**: generators for every prefix in §4.3: `u_<ulid>`, `bk-…`, `p2p-…`, `th-…`, `rc-…`, plus `WQ-<n>` and `HS-<n>` from the T03 sequences. Look at the fake stores for each prefix's exact format (e.g. `lib/features/p2p/data/sources/p2p_listing_writer.dart` for new listing ids) and match it.
3. **`platform/sse`**: a `Broker` with `Publish(topic, payload)` and `Subscribe(topic)`, topics `inbox:<uid>`, `sales:<uid>`, `sales:moderators`, `notifications:<uid>`. `Serve(w, r, topic)` sets `Content-Type: text/event-stream`, `Cache-Control: no-cache`, writes `data: {json}\n\n` per event and `: ping\n\n` every 25 s, flushes each write, and returns when the client goes away. A per-process `seq` counter is added to inbox and sales events. Test it with `httptest` and a cancelled context.
4. **`platform/cloudinary`**: an `Uploader` interface: `Upload(ctx, kind, id, slot, base64) (url, publicID, error)` and `Destroy(ctx, publicID)`. `DecodeImage` decodes base64, enforces `MAX_IMAGE_MB`, and checks the bytes are JPEG/PNG/WebP (`http.DetectContentType`, not the name). The real implementation makes a signed upload to `CLOUDINARY_FOLDER/<kind>/<id>/<slot>`. `Fake` (used when `CLOUDINARY_FAKE=true`) returns `https://example.invalid/<public_id>`. Add a `Thumb(url)` helper for the `c_fill,w_300,q_auto,f_auto` transformation.
5. **`platform/email`**: a `Sender` interface (`SendOTP(ctx, to, code, purpose)`), with `resend`, `brevo` and `log` (prints the code, development only; refuse `log` in production) chosen by `EMAIL_PROVIDER`.
6. **`platform/gemini`**: a `Client` interface (`Word(ctx, prompt) (string, error)`) with a `GEMINI_TIMEOUT` deadline, the real implementation on `google.golang.org/genai` with `GEMINI_MODEL`, and a fake. With no `GEMINI_API_KEY`, the constructor returns nil and callers use rule-based text.
7. **`platform/jobs`**: a ticker every `JOBS_TICK` that runs registered `func(ctx) error` jobs one at a time and logs failures. Jobs must be idempotent and catch up after sleep (§12).
8. **`platform/ratelimit`**: an in-memory token bucket keyed by IP or user, with middleware helpers for `AUTH_RATE_PER_MIN`, `AI_RATE_PER_MIN` and `REPORT_RATE_PER_HOUR`. A limited request answers `429` in the error shape with code `rate_limited` (add it to `docs/error-codes.md`).
9. Wire all of them in `internal/app/deps.go` from config.

**Done when**
- [x] Each service has a fake and unit tests; nothing under `internal/platform` imports `internal/feature`
- [x] The SSE test receives an event and a ping, and the handler exits on disconnect
- [x] An image test rejects a renamed non-image and an oversize image

**Notes:** Clock with Dhaka days, ids for every prefix with the two sequences, the SSE broker with pings and per-process seq, Cloudinary uploader (real signed upload, fake) with strict image checks and Thumb, email senders (Resend, Brevo, log), Gemini REST client, a jobs ticker, an in-memory rate limiter with 429 middleware, and everything wired in deps.go. Notes: Cloudinary Upload takes a checked Image from DecodeImage instead of raw base64 so handlers can refuse bad images first; Gemini uses REST with the key in a header (ADR 0008).

---

## T06 — Contract export, contract tests and the seed loader

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/9 · **Owner:** Arifin · **Plan:** B0.6, §14, §15, F5 · **Branch:** `feature/contract-harness` (plus a frontend PR for F5) · **Depends on:** T03, T04

**Goal:** a machine check that every Go endpoint answers in the same shape as the fake API, and the fake API's data loaded into Postgres.

**Frontend reference:** `lib/app/fake_api_routes.dart` (`FakeApiRoutes.interceptor`), `lib/app/fake_stores.dart`, `lib/core/network/fake_api_interceptor.dart`, every `*_fixtures.dart` / `*_seed.dart` under `lib/features/*/data/sources/`.

**Steps**
1. **Frontend F5 (a frontend PR):** add `test/tool/export_contract_test.dart`. It builds `FakeApiRoutes.interceptor(FakeStores())`, sends one sample request per Appendix A row (all 174, including the catalog admin records and `season/save`), and writes `build/contract/<METHOD>__<path with / as __>.json`, for example `POST__p2p__listings__save.json`. Each file holds `{"request": {method, path, query, body}, "response": <json>}`. Use samples that hit a real record (`bk-cleancode`, `p2p-1`, `WQ-100231`, `HS-101`, `rc-aloghar`, `p-nabila`, …) so goldens aren't `null` where the endpoint usually answers data. Add a second golden for an important `null` case where it helps (`…__miss.json`). It also writes `build/seed/*.json`: books with Editions, categories, authors, publishers, subjects, series, collections, experts, booklists, banners, seasons, ayahs, geo, demo people, their Listings, threads, orders, Bites, reviews, donation places, coupons, deals, ISBN lookup fixtures, Certified Used, Sell Back seeds and notifications. Keep the frontend's 562 tests green.
2. **`scripts/export-contract.sh`** (`make contract-export`): runs `flutter test test/tool/export_contract_test.dart` in `$FRONTEND_DIR`, then copies `build/contract/*` → `testdata/contract/` and `build/seed/*` → `seed/`. Commit both folders.
3. **`internal/contract/contract_test.go`**: for each golden, replay the request against the router (`httptest`, a DB seeded from `seed/`, signed in as `reader@waraqah.test`, or as the right staff account for `staff` rows), then compare **shapes**: same keys (recursively), same JSON kinds, `null` where the golden is `null`. Values may differ. Arrays compare their first element's shape. Report each mismatch with its JSON path.
4. **Pending list:** endpoints not built yet are listed in `internal/contract/pending.txt` and skipped with a count. Each feature task removes its lines. T19 requires the file to be empty.
5. **Route coverage test:** parses Appendix A in `BACKEND_PLAN.md` and checks that every method + path is registered on the mux or listed in `pending.txt`, so a row can't be forgotten.
6. **`cmd/seed`** (`make seed`): reads `seed/*.json` and upserts it in dependency order. Each feature task adds its own loader file (`cmd/seed/<feature>.go`). Rules:
   - **People ids:** demo people keep their fixture ids (`p-nabila`, `p-tanvir`, `p-arif`, …), so goldens, deep links and seeds agree. The fake's signed-in user, id `me`, becomes the account `reader@waraqah.test`; replace every `me` reference with that user's id while loading. Staff accounts: `admin@` (superAdmin), `moderator@`, `catalog@`, `support@waraqah.test`. Each seeded account's password is `SEED_DEMO_PASSWORD`. New sign-ups get `u_<ulid>`.
   - Refuse to run when `APP_ENV=production` unless `SEED_ALLOW_PRODUCTION=true`.
   - Running it twice changes nothing.
7. **`cmd/admin`**: `set-role` (`make set-role EMAIL=… ROLE=…`, validates the role name) and `prune-tokens`.
8. **`scripts/smoke.sh`** (`make smoke`): curls `/healthz`, `/readyz`, `/version` and a few public GETs against `API_BASE_URL`; T19 extends it.

**Done when**
- [x] F5 merged in the frontend; `make contract-export` fills `testdata/contract/` (174+ files) and `seed/`
- [x] `make test` runs the contract test: every endpoint is pending, plus `/healthz` as a sample passing check
- [x] `make seed` loads users and is idempotent; the route coverage test passes

**Notes:** Contract exporter (tools/contract-export, Dart) that runs 186 sample requests against the fake API and writes goldens for all 174 endpoints plus refusal variants, a Go replay test that compares shapes, a pending list, a route coverage test that reads Appendix A, the seed loader (users first), and the admin commands set-role and prune-tokens. Notes: the exporter lives in this repo and is copied into the frontend test/tool folder only while it runs, so no frontend PR (F5) is needed; goldens carry an as field naming the account that sends the request; the seed folder grows with each feature task.

---

## T07 — Auth endpoints and the app's switch to the real API

**Status:** ✅ done · backend PR https://github.com/arifinrafi89/waraqah-backend/pull/10, frontend PR https://github.com/arifinrafi89/waraqah-frontend/pull/145 · **Owner:** Rahinur · **Plan:** B1.1, §5, F1, F2, F3, F8 · **Branch:** `feature/auth` (+ frontend PRs F1, F2, F3, F8) · **Depends on:** T04, T05, T06

**Endpoints (6 + 2 additive):** `POST /auth/login`, `/auth/google`, `/auth/signup/request-otp`, `/auth/signup/verify-otp`, `/auth/password/request-otp`, `/auth/password/reset`, and the additive `POST /auth/refresh`, `/auth/logout` (§4.6). (`POST /auth/delete` belongs to T08, Profile.)

**Frontend reference:** `lib/features/auth/data/sources/{auth_fake_api,auth_fixtures,auth_remote_source}.dart`, `lib/features/auth/data/models/app_user_model.dart` (`{id, name, email, role}`).

**Steps**
1. Package `internal/feature/auth`. Every successful sign-in answers the `AppUserModel` fields **plus** `accessToken`, `refreshToken`, `expiresAt` (§5.2).
2. Behaviour per §5.3:
   - `login`: an email is matched case-insensitively and the bcrypt hash checked; a wrong pair → `Refuse(wrong_credentials)`.
   - `signup/request-otp`: an email contact stores a pending sign-up and sends an OTP (`{"ok":true}`). A phone number is refused with `phone_not_supported`. An already-used email is refused with `email_taken`.
   - `signup/verify-otp`: a wrong or expired code → `wrong_otp`; otherwise create the user (`reader`) and answer user + tokens.
   - `password/request-otp`: always `{"ok":true}`.
   - `password/reset`: a wrong code → `wrong_otp`; otherwise set the hash, revoke all refresh tokens, `{"ok":true}`.
   - `google`: no `idToken` → `Refuse(google_token_missing)`; verify it, find or create the user by email, answer user + tokens.
   - `refresh`: rotate; an unknown/expired/reused token → `401`. `logout`: revoke the token, `{"ok":true}`.
3. Rate-limit every `/auth/*` route by IP (`AUTH_RATE_PER_MIN`) and OTP sends per contact (`OTP_RESEND_SECONDS`).
4. Seed loader for the demo accounts (T06 rules).
5. **Frontend PRs** (Rahinur, plan §20), each keeping the fake API as the default so the frontend tests still pass:
   - **F1:** `API_BASE_URL` from `--dart-define`. Empty = install `FakeApiInterceptor` as today; set = point Dio's base URL there with no interceptor (`lib/core/network/{api_config,dio_client,dio_provider}.dart`, `lib/app/app_bootstrap.dart`).
   - **F2:** add `accessToken`, `refreshToken`, `expiresAt` to `AppUserModel` and the session store, and add an `AuthInterceptor` that sends `Authorization: Bearer …`. On a `401` it calls `/auth/refresh` once and retries, then signs out.
   - **F3:** Google sign-in sends `{idToken}`.
   - **F8:** sign-up copy says email only; phone stays a profile field.
6. App walk: sign up with an email (dev code `OTP_DEV_CODE`), sign out, sign in, reset the password, and Google sign-in once F3 lands. Each staff demo account sees only its admin sections.

**Done when**
- [x] 6 contract tests pass; refresh/logout have handler tests
- [x] F1 and F2 merged in the frontend; the app signs in against the local backend and keeps the session across a token refresh
- [x] Refusal codes `wrong_credentials`, `wrong_otp`, `phone_not_supported`, `email_taken`, `google_token_missing` documented

**Notes:** Backend half merged (PR for feature/auth): all eight endpoints, rate limits, dev OTP code, refresh rotation with theft detection, handler tests. Frontend PR (F1, F2, F3, F8 together) merged: API_BASE_URL, tokens with a refresh-once interceptor, expiry signs the reader out, null answers from login, Google and sign-up show messages. F3 is partial: the body accepts an idToken, but the `google_sign_in` package and OAuth client setup are not wired (needs the team OAuth client IDs). The app walk against a running backend was not done on a device here; the flows are covered by handler tests and the interceptor tests. Extra refusal codes: `contact_invalid`, `password_invalid`, `google_token_invalid`.

---

## T08 — Profile and notifications

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/11 · **Owner:** Rahinur · **Plan:** B1.2, B1.3, §8 · **Branch:** `feature/profile`, `feature/notifications` · **Depends on:** T07

**Endpoints (14)**
- profile (10): `GET /profile`, `POST /profile/save`, `GET /profile/prefs`, `POST /profile/prefs/save`, `GET /addresses`, `POST /addresses/save`, `/addresses/default`, `/addresses/delete`, `GET /geo` (public), `POST /auth/delete`
- notifications (4): `GET /notifications`, `POST /notifications/read`, `/notifications/read-all`, `GET /notifications/live` (SSE)

**Frontend reference:** `lib/features/profile/data/sources/{profile_fake_api,profile_fake_store,address_fake_store}.dart`, `lib/features/profile/data/sources/geo/*.dart`; rules `profile_rules.dart`, `address_rules.dart` (tests `test/profile_rules_test.dart`, `test/address_rules_test.dart`). Notifications: `lib/features/notifications/data/sources/{notification_fake_api,notification_fake_store,notification_seed,notification_sends,notification_sale_sends,notification_live_source}.dart`, `lib/features/notifications/domain/entities/notification_kind.dart`, test `test/notification_sale_senders_test.dart`.

**Steps**
1. **Migration:** `profile_prefs`, `addresses` (a partial unique index for one default per user), `notifications` (index `(user_id, created_at desc)`).
2. **Profile:** port `ProfileRules` and `AddressRules` with their tests. The address endpoints answer the whole list, default first; deleting the default promotes the next one, as the fake store does. Guests get `{name:"", phone:"", photo:null}`-style empties, `[]` addresses and the fake's default prefs.
3. **Geo:** serve `seed/geo.json` (exported from `geo/*.dart`), parsed once and cached in memory; no table needed.
4. **`/auth/delete`:** soft-delete, anonymise name/email/phone, revoke all refresh tokens, and hide the user's Listings and Bites (those tables come later; call the hide hooks through interfaces that are no-ops until T14/T17).
5. **Notifications:** list newest first; `read` / `read-all` publish `{"unread": n}` on `notifications:<uid>`; `live` serves that topic. Guests: `[]`.
6. **`notifications.Sender`** (§8): `Send(ctx, userID, kind, params, target)`. Store `kind` and `params` only, never text (the app builds the words from ARB). Respect the muted groups in `profile_prefs`, exactly as the fake store does. Port the helper functions from `notification_sends.dart` and `notification_sale_sends.dart` as Go functions (`SendOrderShipped(…)` etc.) with the same kinds and params, and port `notification_sale_senders_test.dart`. Wire it in `deps.go` for later features.
7. Seed: the demo reader's notifications from `notification_seed.dart`.

**Done when**
- [x] 14 contract tests pass; ported rules tests pass
- [x] App walk: edit the profile, add/default/delete addresses, mute a group, see notifications and the unread badge update live
- [x] Guest walk on the Profile and Notifications tabs shows no errors

**Notes:** Profile, settings, saved addresses, geography list and account deletion; notifications list, read, read-all and the live unread stream; the notifications.Sender with mute handling and one helper per Dart sender. Seeds the demo reader addresses and notifications (times count back from seeding). Notes: the profile photo is kept as the base64 the app sends (users.photo_data) because the app reads it back as base64; account deletion runs hooks that later features register; the geography list is embedded in the binary.

---

## T09 — Catalog reads, search and records

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/12 · **Owner:** Rahinur · **Plan:** B2.1, B2.2, §10 · **Branch:** `feature/catalog` (split into reads / search / records PRs) · **Depends on:** T06

**Endpoints (25), all public except two:** `GET /books`, `/books/detail`, `/books/details`, `/books/look-inside`, `/books/price-lows`, `/books/series`, `/books/used-options`, `/books/suggest`, `/books/did-you-mean`, `/books/questions`, `POST /books/questions/ask` (me), `POST /books/questions/answer` (me), `GET /categories`, `/subjects`, `/authors/detail`, `/publishers/detail`, `/series/detail`, `/collections`, `/collections/detail`, `/experts`, `/experts/detail`, `/booklists`, `/booklists/detail`, `POST /booklists/mine/save` (me), `POST /booklists/mine/delete` (me)

**Frontend reference:** `lib/features/catalog/data/sources/{book_fake_api,book_suggest_fake_api,book_questions_fake_api,collection_fake_api,booklist_fake_api,phonetic_key}.dart`, `*_fixtures.dart`, `seed/`; models in `lib/features/catalog/data/models/` and `lib/core/models/{book,edition}.dart`; rules `catalog_filters.dart`, `delivery_estimate.dart`; tests `phonetic_key_test.dart`, `search_bangla_test.dart`, `search_filters_test.dart`, `search_sort_test.dart`, `delivery_estimate_test.dart`, `search_isbn_test.dart`.

**Steps**
1. **Migration:** every catalog table in §7.1 (categories, authors, publishers, subjects, books, editions, book_details, look_inside, series, book_questions, book_answers, collections, experts, booklists, price_lows, sales_by_month). Add a GIN index on `books.phonetic_keys` and trigram indexes on titles and author names.
2. **Port the rules:** `PhoneticKey` → `phonetic.go`, `CatalogFilters` → `filters.go` (every filter and sort), `DeliveryEstimate` → `delivery.go`, each with its tests as tables.
3. **`/books`:** exact matches first, then phonetic matches, then filters and sort, exactly like `BookFakeApi.books`. Hidden books only with `includeHidden=true` **and** a catalog staff token. Record searches through a `search.Log` interface (a no-op until T18), counting reader searches only (§10).
4. **Suggest and did-you-mean:** `pg_trgm` similarity, answering in the script the reader typed, up to 5 suggestions, best first.
5. **Details:** `details`, `look-inside` and `series` answer `null` when the fake does. `used-options` combines Certified Used and reader listings (both come later: use interfaces returning empty lists until T14/T16) plus the resale estimate from the fake.
6. **Questions:** `ask` and `answer` take the author and the staff flag from the token. The body's `name` / `isStaff` are accepted and ignored (§4.4).
7. **Booklists:** `/booklists` = staff lists + the caller's own; `mine/save` and `mine/delete` refuse lists that aren't the caller's (`booklist_not_yours`).
8. **`catalog.Books` interface** (§8): find a book, its editions, price and stock changes. Every later feature uses this, never the tables directly.
9. **Seed** all catalog data from `seed/`, computing `phonetic_keys` while loading.

**Done when**
- [x] 25 contract tests pass; phonetic, filter, sort and delivery tests pass
- [x] App walk: search in English and Bangla (phonetic), filters, sorts, book page tabs, series, authors, collections, experts, booklists, asking a question
- [x] `EXPLAIN` on `/books` with a term and filters uses the indexes

**Notes:** All 25 catalog endpoints. Search, suggestions and did-you-mean are ports of the Dart code (phonetic key, ranking, edition filters, sorts) running over an in-memory snapshot of the catalog; delivery estimate ported; booklists with a reader own lists; questions that take the poster from the token. Migration 0003, seeds for the whole catalog, shared test environment (internal/testenv), and the contract test now maps ids the fake API made up. Notes: the plan suggested pg_trgm and GIN indexes for search; the Dart ranking needs the whole catalog, so search runs in memory (Store.Snapshot, invalidated on admin changes) and no trigram index was added. Extra refusal codes: booklist_invalid, book_unknown, question_invalid. Certified Used stock is a stand-in until T16.

---

## T10 — Home, catalog admin and scan

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/13 · **Owner:** Rahinur (scan: Arifin) · **Plan:** B2.3, B2.4, B2.5 · **Branch:** `feature/home`, `feature/catalog-admin`, `feature/scan` · **Depends on:** T09

**Endpoints (29)**
- home (3, public): `GET /home/banners`, `/home/season`, `/islamic/ayah-of-the-day`
- catalog admin (25, all `staff:catalog`): `POST /admin/catalog/books/save`, `/books/hide`; `GET /admin/catalog/categories`, `POST …/categories/save`, `…/categories/delete`; the same three for `/authors` and `/publishers`; `GET /admin/catalog/banners`, `POST …/banners/save`, `…/banners/delete`, `…/banners/move`; `GET /admin/catalog/season`, `POST …/season/save`; `POST …/collections/save`, `…/collections/delete`, `…/booklists/save`, `…/booklists/delete`; `GET …/isbn-lookup`, `GET …/low-stock`, `POST …/editions/stock`, `POST …/import`
- scan (1, public): `GET /scan/lookup`

**Frontend reference:** `lib/features/home/data/sources/{home_fake_api,ayah_fake_api,season_picker}.dart` + fixtures (test `season_picker_test.dart`, `season_api_test.dart`); `lib/features/catalog_admin/data/sources/{catalog_admin_fake_api,catalog_admin_fake_store,catalog_admin_fake_*,catalog_tools_fake_api,catalog_import_fake,isbn_lookup_fixtures}.dart`, rules `catalog_admin_rules.dart`, `list_rules.dart` (tests `catalog_admin_rules_test.dart`, `catalog_admin_record_rules_test.dart`); `lib/features/scan/data/sources/scan_fake_api.dart`.

**Steps**
1. **Migration:** `banners`, `app_config` (the forced Season), `ayahs`, and an `isbn_lookup` table for outside books (from `isbn_lookup_fixtures.dart`).
2. **Home:** port `SeasonPicker` → `season.go` with its tests, using Dhaka dates. `/home/season` answers the forced Season from `app_config` if set, else the date's Season, else `null`. Banners: the active Season's first, then the all-year ones, in `position` order.
3. **Catalog admin:** port `CatalogAdminRules` and `ListRules` with tests. Note the record routes are built in Dart by `CatalogAdminFakeApi.records(kind)`: `/admin/catalog/{categories|authors|publishers}` plus `/save` and `/delete`; the list includes `bookCount`, and delete is refused while a Book uses the record (`record_in_use`). Banner `move` swaps positions (`by: -1|1`). Season `save` with `null` returns Home to automatic.
4. **Tools:** `low-stock` (printed Editions only, no eBooks or pre-orders, at or under `CatalogAdminRules.lowStock`, lowest first); `editions/stock` refuses eBooks and negative values; `import` returns `{imported, skipped: [{row, reason}]}` following `CatalogImportFake.run` row by row; `isbn-lookup` answers `{inCatalog: true, bookId}` or an outside book or `null`.
5. **`alerts.Sweeper` hook:** after any price or stock change, call an `alerts.Sweeper` interface (a no-op until T11 wires the real one).
6. **Scan** (Arifin): `/scan/lookup?isbn=` → the Book having an Edition with that ISBN, or `null`.
7. Body `by` fields are ignored; the audit uses the token (§4.4).

**Done when**
- [x] 29 contract tests pass; season and catalog admin rules tests pass
- [x] App walk as `catalog@waraqah.test`: add and edit a book, hide it, edit records, banners, force a Season, import, low stock; Home changes accordingly
- [x] A reader token on any `/admin/catalog/*` → 403; no token → 401

**Notes:** Home (banners, Season hero card, Ayah of the day), all 25 Admin → Catalog endpoints behind the catalog permission, and the scan lookup. Ports of SeasonPicker, CatalogAdminRules, ListRules and Isbn with their tests; each admin change drops the catalog cache and calls the alerts sweeper hook; a price change keeps the 30-day low. Migration 0004, seeds for banners, ayahs and the ISBN lookup, and the contract replay now pins the clock to the day of the export. Notes: hidden books are not offered to the scanner; the sweeper is a no-op until T11; extra refusal codes are listed in docs/error-codes.md.

---

## T11 — Cart, deals, wishlist and alerts

**Status:** ✅ done · PR https://github.com/arifinrafi89/waraqah-backend/pull/14 · **Owner:** Farhan · **Plan:** B3.1 · **Branch:** `feature/cart`, `feature/wishlist-alerts` · **Depends on:** T08, T09, T10

**Endpoints (13):** `GET /cart`, `POST /cart/add`, `/cart/update`, `/cart/remove`; `GET /deals` (public); `GET /wishlist`, `POST /wishlist/save`, `/wishlist/remove`, `/wishlist/share`, `GET /wishlist/shared` (public); `GET /alerts`, `POST /alerts/set`, `/alerts/remove`

**Frontend reference:** `lib/features/cart/data/sources/{cart_fake_api,cart_fake_store}.dart`, `lib/features/deals/data/sources/{deals_fake_api,deals_fake_store}.dart`, `lib/features/wishlist/data/sources/wishlist_fake_api.dart`, `lib/features/alerts/data/sources/{alert_fake_api,alert_fake_store}.dart`, plus each feature's models.

**Steps**
1. **Migration:** `cart_lines`, `flash_sales`, `bundles`, `wishlist_items`, `wishlist_shares`, `alerts`.
2. **Cart:** line kinds `edition`, `certifiedUsed`, `listing`, `bundle`. Resolve each kind through a `cart.ItemResolver` interface: editions and bundles now, listings in T14 and Certified Used in T16 (until then those resolvers answer "not found" and `add` refuses with `cart_item_unknown`). Answer the cart JSON exactly as `CartFakeStore` builds it (prices from the catalog at read time, deal prices applied).
3. **Deals:** the flash sale, bundles and pre-orders running now (Dhaka time), as `DealsFakeStore` decides.
4. **Wishlist:** saving twice keeps one copy, moved to the top. `share` creates or reuses a share id **for the signed-in user** (the body's `ownerName` is the display name only). `shared?id=` answers `{id, ownerName, books}` or `null`.
5. **Alerts:** `set` / `remove` / list. Implement **`alerts.Sweeper`** (port `AlertFakeStore.sweep`): price-drop and back-in-stock alerts fire once, send a notification via `notifications.Sender`, and set `fired_at`. Wire it in `deps.go`, replacing the T10 no-op.
6. Guests: empty cart, `[]` wishlist, `[]` alerts, exactly as the fake answers a new reader.
7. Seed deals, bundles and demo wishlists (`wl-nabila`).

**Done when**
- [x] 13 contract tests pass
- [x] App walk: add editions and a bundle, change quantities, wishlist and share, set a price alert, then lower the price as catalog staff and receive the notification
- [x] Guest walk on Cart and Wishlist shows no errors

**Notes:** All 13 endpoints. The cart stores only what was added and prices it from the catalog and the deals at read time (flash price, bundles, caps per order); deals run all day with a countdown to midnight; the wishlist keeps one copy newest first and makes a share link for the signed-in reader; alerts are checked on every read and the sweeper now fires them once and notifies through notifications.Sender, replacing the T10 stand-in. Migration 0005, seeds for the demo people, deals and the friend wishlist, and the exporter now refuses two samples that write the same golden file. Notes: an item the cart or alerts refuse still answers the full cart or list (the app always expects one) with the code in X-Waraqah-Error; certified used cart lines wait for Sell Back (T16).

---

## T12 — Checkout, orders, wallet and points

**Status:** ⬜ todo · **Owner:** Farhan · **Plan:** B3.2, B3.3, B3.4, §8 · **Branch:** `feature/checkout`, `feature/orders`, `feature/wallet-points` · **Depends on:** T11

**Endpoints (14)**
- checkout (4): `GET /coupons/check`, `POST /orders/place`, `GET /admin/coupons` (staff:orders), `POST /admin/coupons/create` (staff:orders)
- orders (8): `GET /orders`, `/orders/details`, `POST /orders/cancel`, `/orders/return`, `/orders/reorder`; `GET /admin/orders`, `POST /admin/orders/advance`, `/admin/orders/return` (staff:orders)
- wallet (1): `GET /wallet` · loyalty (1): `GET /points`

**Frontend reference:** `lib/features/checkout/data/sources/{checkout_fake_api,coupon_fake_store,coupon_admin_fake_api,checkout_fixtures}.dart`, rules `checkout_totals.dart`, `coupon.dart` (tests `checkout_totals_test.dart`, `coupon_rules_test.dart`, `gift_rules_test.dart`, `gift_test.dart`); `lib/features/orders/data/sources/{order_fake_api,order_admin_fake_api,order_fake_store,order_fixtures}.dart`, `order_refunds.dart`; `lib/features/wallet/…` (`wallet.dart`, tests `wallet_rules_test.dart`, `wallet_test.dart`, `wallet_sale_refund_test.dart`); `lib/features/loyalty/…` (`loyalty_rules.dart`, test `loyalty_test.dart`).

**Steps**
1. **Migration:** `coupons`, `orders`, `order_lines`, `order_history`, `order_returns`, `wallet_entries`, `points_entries` (indexes on `user_id, at desc`). Order numbers come from `order_number_seq` → `WQ-<n>`.
2. **Port the rules with tests:** `CheckoutTotals`, coupon rules, gift rules → `checkout/totals.go`, `coupons.go`; `LoyaltyRules` → `loyalty/rules.go`; `OrderRefunds` → `orders/refunds.go`; wallet rules → `wallet/rules.go`.
3. **Ledgers** (§8): `wallet.Ledger` (`Credit`, `Spend`, `Balance`; the balance is always a sum of entries, never stored; reasons use the Dart enum names, e.g. `saleRefund`) and `points.Ledger`. Wire both in `deps.go`.
4. **`/orders/place`** in **one transaction**: load the cart and address, compute totals exactly as the app does, apply the coupon, spend points and the wallet, write the order, its lines, history and the address/gift snapshots, empty the cart, update stock through `catalog.Books`, add to `sales_by_month`, and award points. After the commit, send notifications. Refusals: empty cart, unknown address, gift without a name (`cart_empty`, `address_unknown`, `gift_name_missing`).
5. **Orders:** cancel and return follow the fake store's allowed states. Return photos go through `cloudinary.Uploader`. `reorder` answers `{added, skipped}`. Admin `advance` accepts only the next status (otherwise `null`) and notifies the reader. Admin `return` approving refunds the books to the wallet.
6. **`orders.Delivered`** interface (§8): delivered lines per user, for reviews (Verified Purchase) and shelves (T17).
7. Coupons admin: newest first; `create` refuses a taken code (`coupon_code_taken`).
8. Guests: `[]` orders, the zero wallet and zero points answers.
9. Seed demo orders, coupons (`EID100`, …), wallet and points history.

**Done when**
- [ ] 14 contract tests pass; totals, coupon, gift, loyalty and wallet tests pass
- [ ] App walk: check out with a coupon, points and wallet; cancel; ask for a return with a photo; as `support@` advance an order and approve the return; the wallet shows the refund
- [ ] Placing an order is atomic (a test forces a failure mid-way and checks nothing was written)

**Notes:** —

---

## T13 — Donate and donation places admin

**Status:** ⬜ todo · **Owner:** Farhan (admin: Arifin) · **Plan:** B3.5 · **Branch:** `feature/donate` · **Depends on:** T12

**Endpoints (5):** `GET /donate/recipients`, `/donate/recipient` (public), `POST /donate/give` (me), `POST /admin/donate/places/save`, `/admin/donate/places/remove` (staff:orders)

**Frontend reference:** `lib/features/donate/data/sources/{donate_fake_api,donate_admin_fake_api,donate_places_store,donate_fixtures,donation_order}.dart`, rules `recipient.dart`, `donate_place_draft.dart` (`PlaceRules`); tests `donate_rules_test.dart`, `donation_places_test.dart`, `donate_test.dart`.

**Steps**
1. **Migration:** `donate_places` (soft-remove with `removed_at`), `donate_needs`.
2. Port the recipient rules and `PlaceRules` with tests.
3. **`give`:** through the orders service (an interface), create an order delivered free to the recipient (`is_donation`, `donate_place_id`), increase `received`, and answer `{orderNumber, totalBdt}`. Cash on delivery and more copies than still needed are refused (`donate_cod_not_allowed`, `donate_too_many`).
4. **Admin:** `save` validates with `PlaceRules` and answers every place; `remove` answers the rest or `null` for an unknown id.
5. Seed the places (`rc-aloghar`, …) and their needs.

**Done when**
- [ ] 5 contract tests pass; donate rules tests pass
- [ ] App walk: donate a book (it shows in Orders), then as `support@` add and remove a place

**Notes:** —

---

## T14 — Listings, reports and blocks, moderation

**Status:** ⬜ todo · **Owner:** Arifin · **Plan:** B4.1, B4.2, B4.3, §8, §11 · **Branch:** `feature/listings`, `feature/reports`, `feature/moderation` · **Depends on:** T08, T09, T05 (images)

**Endpoints (15)**
- p2p (6): `GET /p2p/listings`, `/p2p/listing`, `/p2p/listings/for-book`, `/p2p/seller` (public), `GET /p2p/listings/mine`, `POST /p2p/listings/save` (me)
- report (4): `GET /blocks`, `POST /blocks/add`, `/blocks/remove`, `POST /reports` (me)
- moderation (5, staff:moderate): `GET /moderation/listings`, `POST /moderation/listings/decide`, `GET /moderation/reports`, `POST /moderation/reports/act`, `GET /moderation/log`

**Frontend reference:** `lib/features/p2p/data/sources/{p2p_fake_api,p2p_fake_store,p2p_listing_writer,p2p_catalog_link,p2p_people,p2p_ratings,p2p_*seed}.dart`, rules `listing_rules.dart`, `fair_price.dart` (tests `listing_rules_test.dart`, `fair_price_test.dart`); `lib/features/report/data/sources/{report_fake_api,report_fake_store}.dart`, `report_rules.dart` (test `report_rules_test.dart`); `lib/features/moderation/data/sources/{moderation_fake_api,moderation_fake_store,moderation_fake_reports,moderation_subjects,moderation_seed}.dart`, `moderation_rules.dart` (test `moderation_rules_test.dart`).

**Steps**
1. **Migration:** `listings`, `listing_photos`, `ratings`, `blocks`, `reports`, `moderation_log`; strikes and bans live on `users`.
2. **Port the rules with tests:** `ListingRules` → `p2p/rules.go`, `FairPrice` → `p2p/fair_price.go`, `ReportRules`, `ModerationRules`.
3. **Listings:** `save` follows `p2p_listing_writer.dart`: a new listing or an edit of your own; status changes only as the rules allow; `submit` sends it for review. `photos` lists the slots kept and `photoData` `{slot: base64}` the new ones: upload each through `cloudinary.Uploader`, and destroy the asset of a removed slot. Responses keep `photos: [slot…]` (contract v1). Refuse with `ListingRules` codes (`listing_not_editable`, …). `isMine` comes from the token. Banned users can't save (`reader_banned`).
4. **Public lists:** on sale or reserved, newest first, `available=true` and `limit=` as the fake does; hide listings of users the viewer blocked or who blocked the viewer.
5. **Interfaces** (§8): implement `listings.Status` (reserve, sell, release, for inbox and handled sales), `blocks.Checker`, and `moderation.Bans`, and wire them in `deps.go`. Replace the T08 account-delete hook and the T09 `used-options` / T11 cart `listing` resolvers with the real ones.
6. **Reports:** create with a per-user rate limit (`REPORT_RATE_PER_HOUR`); blocks answer the blocked list, newest first.
7. **Moderation:** queue = listings waiting for review; `decide` approves or rejects with a reason, notifies the seller, and logs it with the **token's** staff name (the body `by` is ignored). `reports` = open reports grouped one per reported thing. `act` follows `moderation_fake_reports.dart`: strikes, bans, and removal of the reported thing. Removal of messages, Bites, comments and reviews goes through a `moderation.Remover` registry that each later feature (T15, T17) registers into. Removing a listing destroys its Cloudinary photos.
8. Seed demo people (`p-…`), their listings, ratings and the moderation seed.

**Done when**
- [ ] 15 contract tests pass; listing, fair price, report and moderation rules tests pass
- [ ] App walk: sell a book with photos, save a draft, submit; as `moderator@` approve one and reject one; report a listing and act on it; block a seller and see their listings disappear
- [ ] Photos reach Cloudinary in a dev run with real keys, and removed slots are deleted

**Notes:** —

---

## T15 — Inbox and book requests

**Status:** ⬜ todo · **Owner:** Farhan (book requests: Arifin) · **Plan:** B4.4, B4.5, §9, §12 · **Branch:** `feature/inbox`, `feature/book-requests` · **Depends on:** T14

**Endpoints (16)**
- inbox (11, me): `GET /inbox`, `/inbox/thread`, `/inbox/live` (SSE), `POST /inbox/open`, `/inbox/send`, `/inbox/offer`, `/inbox/offer/decide`, `/inbox/read`, `/inbox/listing/release`, `/inbox/listing/sold`, `/inbox/rate`
- book requests (5): `POST /requests`, `/requests/close`, `GET /requests/mine`, `/requests/wanted` (me), `GET /requests/demand` (staff)

**Frontend reference:** `lib/features/inbox/data/sources/{inbox_fake_api,inbox_fake_store,inbox_fake_actions,inbox_fake_selling,inbox_fake_rating,inbox_fake_replies,inbox_fake_json,inbox_fake_seed,inbox_live_source}.dart`, rules `offer_rules.dart`, `rating_rules.dart` (tests `inbox_rules_test.dart`, `seller_ratings_test.dart`, `inbox_blocking_test.dart`, `inbox_buying_test.dart`, `inbox_selling_test.dart`, `inbox_live_test.dart`); `lib/features/book_request/data/sources/{book_request_fake_api,book_request_fake_store,book_request_demand,book_request_seed}.dart`, `request_rules.dart` (test `book_request_test.dart`).

**Steps**
1. **Migration:** `threads` (unique `(listing_id, buyer_id)`), `messages`, `offers`, `book_requests`.
2. **Port the rules with tests:** `OfferRules`, `RatingRules`, `RequestRules`.
3. **Inbox:** port `inbox_fake_json.dart` for the JSON (the thread's `role` and `unread` come from the token). `open` starts or reuses the buyer's thread. `offer`, `decide`, `sold` and `release` change the listing through `listings.Status`. `rate` is allowed once each, after the sale. Blocks are enforced on every write (`blocked_reader`). Each change publishes `{seq, threadId, listingId}` on `inbox:<buyer>` and `inbox:<seller>` **after** the commit.
4. **Register** a message remover in the `moderation.Remover` registry (T14).
5. **Demo bot** (`DEMO_MODE=true`, §12): when a real user writes to a seeded demo seller, the demo side answers like `inbox_fake_replies.dart` / `inbox_fake_rating.dart` after `DEMO_BOT_DELAY`, as a job that only acts for seeded demo accounts.
6. **Book requests:** `create`, `close` (answers the reader's requests) and `mine`; `wanted` = other readers' open requests for books the caller is selling; `demand` = open requests per title for staff (port `book_request_demand.dart`). When a listing for a requested book is approved (hook from T14's moderation `decide`), notify the requesters, as the fake store does.
7. Seed threads, messages and requests.

**Done when**
- [ ] 16 contract tests pass; offer, rating and request rules tests pass
- [ ] App walk with two signed-in devices (or one plus the demo bot): open a thread, chat, offer, accept, mark sold, rate; the other side updates live
- [ ] Blocking a reader stops new messages in both directions

**Notes:** —

---

## T16 — Handled sales, Sell Back and Certified Used

**Status:** ⬜ todo · **Owner:** Arifin · **Plan:** B4.6, B4.7, §9, §12 · **Branch:** `feature/handled-sales`, `feature/sell-back` · **Depends on:** T12 (wallet), T14, T15

**Endpoints (16)**
- handled sales (10): `POST /sales/buy`, `/sales/step`, `/sales/dispute`, `/sales/payout`, `GET /sales/detail`, `/sales/mine`, `/sales/earnings`, `/sales/live` (SSE) (me); `GET /sales/disputes`, `POST /sales/disputes/settle` (staff:moderate)
- sell back (6): `POST /sell-back`, `GET /sell-back/mine`, `/sell-back/books`, `/sell-back/book` (me); `GET /sell-back/queue`, `POST /sell-back/grade` (staff:catalog)

**Frontend reference:** `lib/features/handled_sale/data/sources/{handled_sale_fake_api,handled_sale_fake_store,handled_sale_fake_steps,handled_sale_fake_money,fake_sale,sale_changes,handled_sale_seed,sale_live_source}.dart`, `sale_math.dart` (tests `handled_sale_test.dart`, `handled_sale_flow_test.dart`, `live_handled_sales_test.dart`, `wallet_sale_refund_test.dart`); `lib/features/sell_back/data/sources/{sell_back_fake_api,sell_back_fake_store,sell_back_seed}.dart`, `sell_back_rules.dart` (test `sell_back_test.dart`); `lib/features/finished_it/domain/entities/finished_it_offers.dart` (test `finished_it_test.dart`).

**Steps**
1. **Migration:** `handled_sales` (ids `HS-<n>` from `handled_sale_seq`), `payouts`, `sell_backs`, `certified_used`.
2. **Port the rules with tests:** `SaleMath` → `handledsale/math.go`, `SellBackRules` → `sellback/rules.go`, `FinishedItOffers` → `sellback/finished.go`.
3. **Handled sales:** `buy` reserves the listing via `listings.Status`. `step` (`send`, `cancel`, `confirm`) follows `handled_sale_fake_steps.dart`. Money follows `handled_sale_fake_money.dart`: the seller's earnings by `SaleMath.sellerGets`, and `saleRefund` wallet credits on cancellation or a refund. `dispute` uploads its photos; moderators `settle` (`refund` true/false) and get the open disputes back. `payout` moves earnings into `payouts`. Every move publishes `{seq, saleId}` on `sales:<buyer>`, `sales:<seller>` and, for disputes, `sales:moderators`, and sends the notifications from `notification_sale_sends.dart`.
4. **Demo bot:** with `DEMO_MODE=true`, a seeded demo seller sends the book after `DEMO_BOT_DELAY` (job, idempotent).
5. **Sell Back:** `books?q=` lists the catalog books Waraqah buys back; `book` gives the quote; `create` books a pickup. A **courier job** (always on) moves `pickupBooked` → `checking` for every pickup older than `COURIER_PICKUP_DELAY`, catching up after sleep. Staff `grade` (accept or not, with the condition) pays the reader through `wallet.Ledger`, adds a `certified_used` copy, notifies the reader and answers the queue.
6. **Certified Used:** plug the real resolvers into the cart (`certifiedUsed` kind, T11) and the catalog's `used-options` (T09).
7. Seed the sale and Sell Back seeds.

**Done when**
- [ ] 16 contract tests pass; sale math, Sell Back and finished-it tests pass
- [ ] App walk: buy a handled sale, seller sends, buyer confirms, earnings and payout; open and settle a dispute as `moderator@`; sell back a book, the courier job moves it, `catalog@` grades it, and the wallet is credited
- [ ] The jobs are idempotent (a test runs them twice)

**Notes:** —

---

## T17 — Bites, reviews, readers and shelves

**Status:** ⬜ todo · **Owner:** Rahinur (shelves: Arifin) · **Plan:** B5.1, B5.2, B5.3 · **Branch:** `feature/bites`, `feature/reviews-readers`, `feature/shelves` · **Depends on:** T12 (`orders.Delivered`), T14 (blocks, bans, remover)

**Endpoints (18)**
- bites (8): `GET /bites`, `/bites/detail` (public); `POST /bites/post`, `/bites/edit`, `/bites/delete`, `/bites/like`, `/bites/comments/post`, `/bites/comments/delete` (me)
- reviews (3): `GET /reviews` (public); `POST /reviews/save`, `/reviews/delete` (me)
- readers (2): `GET /readers/detail` (public), `POST /readers/follow` (me)
- shelves (5, me): `GET /shelves`, `POST /shelves/move`, `/shelves/progress`, `GET /reading/stats`, `POST /reading/goal`

**Frontend reference:** `lib/features/bites/data/sources/{bite_fake_api,bite_fake_store,bite_fake_json,bite_records,bite_fixtures}.dart`, `bite_rules.dart` (tests `bite_rules_test.dart`, `bites_api_test.dart`, `bite_detail_test.dart`); `lib/features/reviews/data/sources/{review_fake_api,review_fake_store,review_seed}.dart`, `review_rules.dart` (test `reviews_api_test.dart`); `lib/features/readers/data/sources/{reader_fake_api,follow_fake_store}.dart`; `lib/features/shelves/data/sources/{shelf_fake_api,shelf_fake_store,shelf_seed,reading_log}.dart`, `progress_rules.dart` (tests `shelves_test.dart`, `reading_stats_test.dart`).

**Steps**
1. **Migration:** `bites`, `bite_likes`, `bite_comments`, `reviews`, `follows`, `shelf_entries`, `shelf_order_sync`, `reading_days`, `reading_goals`.
2. **Port the rules with tests:** `BiteRules` (count graphemes with `rivo/uniseg`, not bytes), `ReviewRules`, `ProgressRules`.
3. **Bites:** `forYou` / `following` feeds, `bookId` / `authorId` filters, newest first, **at most 30**. Comments have one level of replies. Edit and delete are own-only. Blocks hide both ways; banned users can't post. Answers follow `bite_fake_json.dart`. Register Bite and comment removers with moderation.
4. **Reviews:** one per user per book. "Verified" comes from `orders.Delivered`. Saving or deleting recomputes `books.rating` in the same transaction. Register a review remover with moderation.
5. **Readers:** the reader page respects `profileVisible` / `activityVisible` from prefs and blocks; `follow` answers the page and notifies the followed reader.
6. **Shelves:** `move` (no `shelf` takes the book off; an unknown book → `null`); `progress` per `ProgressRules`; stats per year in Dhaka days (streak, per month, categories); `goal` 1–365. A delivered order puts its book on Want to Read **once** (`shelf_order_sync`), as `shelf_fake_store.dart` does.
7. Replace the T08 account-delete hook for Bites with the real one.
8. Seed Bites, reviews, follows and shelves.

**Done when**
- [ ] 18 contract tests pass; Bite, review and progress rules tests pass
- [ ] App walk: post, edit, like, comment and reply to a Bite; review a delivered book (Verified) and see the rating change; follow a reader; move books between shelves, log progress, set a goal and see the stats
- [ ] A moderator removing a Bite or review through a report works end to end

**Notes:** —

---

## T18 — AI assistant and admin dashboard

**Status:** ⬜ todo · **Owner:** Arifin · **Plan:** B6.1, B6.2, §10, §13, F4 · **Branch:** `feature/assistant`, `feature/dashboard` (+ frontend PR F4) · **Depends on:** T09, T12, T14, T15, T16

**Endpoints (3):** `GET /assistant/greeting` (public), `POST /assistant/ask` (me), `GET /admin/dashboard` (staff)

**Frontend reference:** `lib/features/ai_assistant/data/sources/{assistant_fake_api,assistant_brain,assistant_parser,assistant_catalog,assistant_replies,gemini_chatbot}.dart` (tests `ai_assistant_test.dart`, `smarter_ai_test.dart`); `lib/features/admin/data/sources/{dashboard_fake_api,dashboard_fake_store,search_log}.dart`.

**Steps**
1. **Assistant:** port `AssistantParser`, `assistant_catalog.dart` (`assistantPicksFor`), `AssistantReplies` and `assistant_brain.dart` to `internal/feature/assistant`, with both test files as tables. `ask` answers `{id, text, bookIds}` (plus `basket: {editionIds, totalBdt}` where the fake gives one), using `lang`. Books, prices and stock always come from `catalog.Books`.
2. **Gemini (optional):** with `GEMINI_API_KEY`, ask Gemini only to **word** the reply around the already-picked books. Never let it add books or prices. On error or `GEMINI_TIMEOUT`, keep the rule-based text. Rate-limit `ask` per user (`AI_RATE_PER_MIN`).
3. **Frontend F4 (a frontend PR):** remove `gemini_chatbot.dart` and its provider wiring so no key ships in the app.
4. **Search log:** implement `search.Log` (port `search_log.dart`: growing live-search terms from the same user within 2 minutes count once; staff `includeHidden` searches don't count) into `search_log(term, day, count)`, replacing the T09 no-op.
5. **Dashboard:** port `dashboard_fake_store.dart`: today's numbers (Dhaka), what's waiting (listings in review, open reports, disputes, returns, Sell Back queue, low stock), top searches and most requested books. Each number comes from its feature through a small read interface; the dashboard never queries other features' tables directly.

**Done when**
- [ ] 3 contract tests pass; both assistant test files pass in Go
- [ ] App walk: ask the assistant in English and Bangla (with and without a Gemini key); the dashboard as `admin@` matches what the other screens show
- [ ] F4 merged; `grep -ri gemini lib/` in the frontend finds no API call

**Notes:** —

---

## T19 — Hardening, deployment and launch

**Status:** ⬜ todo · **Owner:** everyone · **Plan:** Phase 7, §5.4, §16, §17, §19, F6, F7 · **Branch:** `feature/launch` (+ frontend PRs F6, F7) · **Depends on:** T01–T18

**Steps**
1. **Full contract run:** `internal/contract/pending.txt` is empty, all 174 goldens pass, and the route coverage test passes. Re-run `make frontend-sync && make contract-export` first to catch any frontend drift, and record any differences in `docs/contract-changes.md`.
2. **Guest walk** (§5.4): open every tab signed out against the local backend. Any error state is a bug: fix the `me` GET that answered 401 instead of the empty value.
3. **Staff role walk:** sign in as each demo staff account; each opens only its sections, and every other admin endpoint answers 403.
4. **Security pass:** rate limits on auth, OTP, assistant and reports; body and image limits; no secrets in logs (grep a full app walk's log for `Bearer`, `password`, `otp`, `base64`); `OTP_DEV_CODE` empty and `EMAIL_PROVIDER` not `log` in production; CORS limited to the real web origin.
5. **Performance:** `EXPLAIN ANALYZE` every list query on seeded data; add missing indexes in a new migration; check the Neon size against the 0.5 GB limit.
6. **Frontend F6:** the three live sources reconnect with backoff and ignore `:` comment lines (owners per plan §20). **F7** (contract v1.1): add `photoUrls` to listing responses (backend) and show real photos (frontend), recorded in `docs/contract-changes.md`.
7. **Deploy:** Neon project (pooled `DATABASE_URL`, direct `DATABASE_URL_DIRECT`), a Render free web service from `main` (Go build with `-ldflags` for `/version`), every production env var set from `.env.example`, `RUN_MIGRATIONS_ON_START=true`, then `make seed` with `SEED_ALLOW_PRODUCTION=true` once. CI must pass before Render deploys.
8. **Smoke:** extend `scripts/smoke.sh` to sign in as the demo reader and hit one endpoint per feature; run `make smoke API_BASE_URL=https://<service>.onrender.com/v1`.
9. **Apps:** build the Flutter app (and optionally `flutter build web`) with `--dart-define=API_BASE_URL=https://<service>.onrender.com/v1`, and do a final app walk on a phone.
10. **Docs:** README "Status" and "Getting started" updated; both repos' AGENTS.md describe the live setup; `BACKEND_PLAN.md` status line updated; every task here ✅.

**Done when**
- [ ] 174/174 contract tests pass and `pending.txt` is empty
- [ ] Guest walk and staff role walk are clean
- [ ] The production URL passes `make smoke`, and the app works end to end against it
- [ ] Free-tier budgets checked (Render hours, Neon size and CU-hours, Cloudinary credits, email and Gemini caps)

**Notes:** —
