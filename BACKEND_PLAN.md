# Waraqah Backend — Plan

> **Read this before writing any backend code.** It is the single source of truth for the backend repository (`waraqah-backend`). It is written for the team (Rahinur, Farhan, Arifin) and for the coding agents working with them. When a decision here changes, update this file in the same PR.

**Status:** built (2026-10-04): all 174 endpoints, every contract golden passing, deployment setup ready (`render.yaml`, `docs/deploy.md`); see `TASKS.md`.
**Frontend:** <https://github.com/arifinrafi89/waraqah-frontend> (`main`, all 13 front-end handover items merged, PR #144).
**Backend:** <https://github.com/arifinrafi89/waraqah-backend>.
**Frontend checkout for reference:** `../waraqah-frontend/`, a sibling folder next to this repo, read-only (§3).

---

## Contents

1. [What we are building](#1-what-we-are-building)
2. [Stack and why](#2-stack-and-why)
3. [Repositories and local layout](#3-repositories-and-local-layout)
4. [The API contract (the most important section)](#4-the-api-contract-the-most-important-section)
5. [Accounts, tokens and roles](#5-accounts-tokens-and-roles)
6. [Backend architecture](#6-backend-architecture)
7. [Database](#7-database)
8. [Cross-feature services](#8-cross-feature-services)
9. [Live updates (SSE)](#9-live-updates-sse)
10. [Search](#10-search)
11. [Images (Cloudinary)](#11-images-cloudinary)
12. [Background jobs and demo behaviour](#12-background-jobs-and-demo-behaviour)
13. [AI assistant](#13-ai-assistant)
14. [Seed data and the contract export](#14-seed-data-and-the-contract-export)
15. [Testing](#15-testing)
16. [Debugging and observability](#16-debugging-and-observability)
17. [Security](#17-security)
18. [Configuration and secrets](#18-configuration-and-secrets)
19. [Deployment on free tiers](#19-deployment-on-free-tiers)
20. [Frontend integration tasks](#20-frontend-integration-tasks)
21. [Work plan, phases and owners](#21-work-plan-phases-and-owners)
22. [Rules for coding agents](#22-rules-for-coding-agents)
23. [Open questions](#23-open-questions)
- [Appendix A — Endpoint checklist (generated)](#appendix-a--endpoint-checklist-generated-from-the-frontend)
- [Appendix B — Business rules to port](#appendix-b--business-rules-to-port)
- [Appendix C — Where the fake backend logic lives](#appendix-c--where-the-fake-backend-logic-lives)

---

## 1. What we are building

Waraqah is a Bangladeshi book store app (new books sold by Waraqah, used books between readers, Book-Bites, reading life, an AI assistant). The Flutter app is finished and runs entirely on a **fake API** inside the app (`FakeApiInterceptor`). Every screen already talks to that fake API through Dio, with real paths, real JSON and real refusals.

**The backend's job is to replace the fake API without the app noticing.** The frontend's own rule (frontend `AGENTS.md` §2) is: *going live must only mean changing the API address.* So:

- The **fake API is the specification.** Its paths, methods, query and body fields, JSON responses and `null` answers (Appendix A) are what the Go server must serve.
- The **fake stores are the first draft of the business logic.** Each `*_fake_store.dart` (Appendix C) does in memory what the Go service does with PostgreSQL.
- The **rules classes are already written.** `ListingRules`, `OfferRules`, `SaleMath`, `SellBackRules` and the others (Appendix B) are pure Dart. Port them to Go line by line, with their tests.

Only a few small frontend changes are needed (tokens, base URL, Google ID token; §20). Everything else stays as it is, including the 562 frontend tests, which keep running on the fake API.

### Non-negotiables

1. **Contract compatibility** (§4). A response that parses in the fake API must parse from Go. Contract tests (§15) enforce it.
2. **Free tiers only, no card** (§19).
3. **Server-side rules.** Every check the fake backend makes (ownership, statuses, limits, blocks, bans, roles), Go makes too. The app's checks are only for UX.
4. **Agent-friendly code** (§22): small files, predictable names, one way to do each thing, everything greppable across both repos.

---

## 2. Stack and why

| Concern | Choice | Why |
|---|---|---|
| Language | **Go** (module `go 1.26`, the version installed locally) | Single static binary, fast cold start on free hosting, strong standard library, easy for agents to read. |
| HTTP | **Standard library `net/http`** with the method-and-path `ServeMux` | No framework to learn. Every route is `mux.HandleFunc("POST /v1/p2p/listings/save", h.Save)`, so it is greppable. |
| Database | **PostgreSQL 16+ on Neon** (free: 0.5 GB, 100 CU-hours/month, no expiry, no card) | Relational data (orders, money, threads) needs transactions. Neon sleeps when idle and wakes on connect. |
| DB access | **pgx v5** + **sqlc** (typed Go generated from SQL) | SQL stays visible in `db/queries/*.sql`; agents read and write SQL, not ORM magic. |
| Migrations | **goose** (SQL files in `db/migrations/`) | Plain numbered SQL files, up and down. |
| Auth | Our own **JWT** (access + refresh), **bcrypt** passwords, **email OTP**, **Google ID token** verification | Keeps the `/auth/*` contract the app already uses (§5). No SMS: it costs money. |
| Email | **Resend** (free plan) behind an `email.Sender` interface; **Brevo** as an alternative | OTP codes for sign-up and password reset. Check the current free limits when signing up. |
| Images | **Cloudinary** (free: 25 credits/month), signed **server-side** uploads | The app already sends photos as base64; the server uploads them. The server can also delete images (§11). |
| AI | **Gemini** via `google.golang.org/genai`, key on the server | The key never ships in the app (§13). |
| Live updates | **Server-sent events** from an in-process broker | The app already reads `/inbox/live`, `/sales/live` and `/notifications/live` this way (§9). |
| Config | Environment variables; `.env` loaded in development (`github.com/joho/godotenv`) | One place for every key (§18, `.env.example`). |
| Logging | **`log/slog`** JSON logs with request IDs | Easy to grep and to paste into an agent (§16). |
| Tests | `testing`, `net/http/httptest`, a local Postgres (Docker Compose), contract shape tests | §15. |
| Lint | `gofmt`, `go vet`, **golangci-lint** | Run in CI and before each PR. |
| Hosting | **Render** free web service (no card; sleeps after 15 min idle, ~1 min cold start) | §19. Do **not** use Render's free Postgres: it is deleted after 30 days. |

Not chosen: Firebase (no server code on the free plan, so money and moderation rules can't be enforced; see the discussion in the frontend session), an ORM (hides SQL from agents), a web framework (unneeded with Go 1.22+ routing).

---

## 3. Repositories and local layout

Two repositories, checked out side by side in one parent folder (any folder name works):

```
<parent folder>/
├── waraqah-backend/                  ← this repo (github.com/arifinrafi89/waraqah-backend)
│   ├── AGENTS.md                     rules for coding agents (short; points here)
│   ├── BACKEND_PLAN.md               this file
│   ├── TASKS.md                      the 19 build tasks and their status
│   ├── README.md                     how to run it
│   ├── .env.example                  every key, with placeholders (committed)
│   ├── .env                          your real values (git-ignored)
│   └── … (code, see §6)
└── waraqah-frontend/                 ← the Flutter app (github.com/arifinrafi89/waraqah-frontend), read-only from here
```

Set up the reference checkout once, from the parent folder:

```bash
git clone https://github.com/arifinrafi89/waraqah-frontend.git
```

Its location is `FRONTEND_DIR` in `.env` (default `../waraqah-frontend`). Keep it current with `git -C ../waraqah-frontend pull` (the Makefile has `make frontend-sync`). **Never commit to the frontend from this repo.** Frontend changes go through the frontend repo's own branches and PRs (§20).

Why side by side: agents working in the backend can open `../waraqah-frontend/lib/features/<feature>/data/sources/<x>_fake_api.dart` and the models next to the Go code they are writing, with no extra setup, and neither repo's git sees the other's files.

Branch and PR rules mirror the frontend: one branch per feature (`feature/<kebab-name>`), small PRs, "Create a merge commit", nobody merges their own PR, commit messages with `Committed by:` and `Feature:` lines, no AI attribution lines.

---

## 4. The API contract (the most important section)

### 4.1 Base URL and paths

- Base: `API_BASE_URL` = `https://<service>.onrender.com/v1` in production, `http://localhost:8080/v1` locally (Android emulator: `http://10.0.2.2:8080/v1`).
- **Paths are exactly the fake API paths under `/v1`.** `P2pFakeApi.save = '/p2p/listings/save'` becomes `POST /v1/p2p/listings/save`. No renaming, no path parameters (`/listing?id=…`, never `/listing/{id}`).
- **Methods:** reads are `GET` with query parameters; changes are `POST` with a JSON body. The frontend uses no `PUT`, `PATCH` or `DELETE`, so neither does the backend in v1.
- The complete list, with auth levels and the fake API's notes on bodies and `null`, is **Appendix A** (174 endpoints).

### 4.2 Responses

| Situation | Status | Body |
|---|---|---|
| Success | `200` | The same JSON value the fake handler returns: an object, a list, `{ "ok": true }`, or a number/string where the fake returns one. |
| Lookup miss (unknown id on a `GET`, e.g. `/p2p/listing?id=x`, `/books/detail`, `/donate/recipient`) | `200` | `null` |
| Refused change (breaks a rule, not yours, wrong state, wrong OTP) | `200` | `null`, plus header `X-Waraqah-Error: <code>` (§16) |
| Missing or invalid token on a `me` or staff endpoint (except `me` GETs, §5.4) | `401` | `{"error":{"code":"unauthorized","message":"…","requestId":"…"}}` |
| Signed in but wrong role | `403` | same error shape, code `forbidden` |
| Unknown path | `404` | same error shape, code `not_found` |
| Invalid JSON, body too large | `400` / `413` | same error shape |
| Server error | `500` | same error shape, code `internal` (details only in logs) |

**Why `200 null` for refusals and misses:** the remote sources decide on `response.data == null`. For example, `AuthRemoteSource.resetPassword` turns `null` into `AuthFailure.wrongCode` ("wrong code"), and `P2pRemoteSource.listing` turns `null` into "listing missing". A `4xx` would make Dio throw and show a generic error instead. A later contract version may move to `4xx` codes, but only together with a frontend change.

### 4.3 JSON shapes

- **Field names:** exactly the Dart model fields, camelCase, no renames (the frontend has no `@JsonKey(name:)` or `fieldRename`). Source of truth: `../waraqah-frontend/lib/features/*/data/models/*.dart` (52 model files) and `../waraqah-frontend/lib/core/models/{book,edition}.dart`.
- **Include every field** the model declares, even when it has a default. Unknown extra fields are ignored by the app, so additive fields are safe.
- **Enums:** the Dart enum value name as a string: `"inReview"`, `"likeNew"`, `"meetInPerson"`, `"wantToRead"`, `"superAdmin"`, `"saleRefund"`.
- **Dates:** RFC 3339 with an offset, e.g. `2026-10-03T14:05:00+06:00` (Dart `DateTime.parse` accepts it). "Today", streaks, seasons and "orders today" use `APP_TIMEZONE` (`Asia/Dhaka`).
- **Money:** whole taka as integers (`priceBdt`, `totalBdt`, `amountBdt`). Never floats.
- **IDs:** strings. Keep the frontend's readable prefixes so seeds, logs and the app agree: books `bk-…`, listings `p2p-…`, threads `th-…`, orders `WQ-100231` (a sequence), handled sales `HS-201`, places `rc-…`, users `u_<ulid>`.
- **Per-viewer fields** are computed from the token, never taken from the request: `isMine`, `isMyDeal`, a thread's `role`, `unread`, a review's "verified", etc. (frontend AGENTS.md §4.4: "the app never compares names").
- **Ordering:** as the fake API does ("newest first" etc., see Appendix A notes).
- **No pagination in v1:** where the fake caps a list (Bites: 30 newest), the server caps it the same way.

### 4.4 Requests

- Query parameter names and body keys exactly as the fake handler reads them (`options.queryParameters['id']`, `options.data['listingId']`). When in doubt, open the fake API file named in Appendix A.
- Photo uploads arrive as **base64 strings inside the JSON body**: listing `photoData: {slot: base64}`, return `photos: [base64]`, dispute `photos: [base64]`. The server decodes, checks the type and size, and uploads to Cloudinary (§11). Max request body: `MAX_REQUEST_BODY_MB` (default 12).
- Some staff bodies carry a `by` name (moderation decisions, dispute settlements). **Ignore it**: the audit log uses the signed-in staff member from the token. Accept the field so old app versions still work.
- `Accept-Language` is not sent; the AI assistant gets `lang` in its body or query.

### 4.5 Live endpoints (SSE)

| Path | Event `data` | Fired when |
|---|---|---|
| `GET /v1/inbox/live` | `{"seq": n, "threadId": "th-…", "listingId": "p2p-…"}` | any change to one of the user's threads or its Listing |
| `GET /v1/sales/live` | `{"seq": n, "saleId": "HS-…"}` | any move on a handled sale the user buys or sells, and disputes for moderators |
| `GET /v1/notifications/live` | `{"unread": n}` | the user's unread count changes |

Format: `Content-Type: text/event-stream`, one `data: {json}\n\n` per event, plus a comment line `: ping\n\n` every 25 s so proxies keep the stream open. The app sends its `Authorization` header (Dio) on these requests. See §9.

### 4.6 Additive endpoints (new in the backend, unknown to the fake)

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/v1/auth/refresh` | `{refreshToken}` → new `{accessToken, refreshToken, expiresAt}` |
| `POST` | `/v1/auth/logout` | revokes the refresh token |
| `GET` | `/healthz`, `/readyz`, `/version` | no `/v1` prefix; liveness, DB check, build info (§16) |

### 4.7 Versioning

The contract is **v1**: the fake API as of frontend `main` at PR #144. Any change to a path, field, enum value or `null` meaning needs a PR in **both** repos, the frontend's first if the app must change, and an entry in `docs/contract-changes.md` in this repo.

---

## 5. Accounts, tokens and roles

### 5.1 Users and roles

- `users` table (§7) with `role` ∈ `reader`, `moderator`, `catalogManager`, `support`, `superAdmin` (the frontend `UserRole` names).
- Permissions mirror the frontend's `UserRole` getters: `isStaff` (not reader), `canModerate` (moderator, superAdmin), `canManageCatalog` (catalogManager, superAdmin), `canManageOrders` (support, superAdmin). Implement once in `internal/platform/auth/roles.go` and port `../waraqah-frontend/lib/features/auth/domain/entities/user_role.dart` exactly.
- Seeded demo accounts (same emails as the frontend, password `SEED_DEMO_PASSWORD`): `admin@waraqah.test` (superAdmin), `moderator@`, `catalog@`, `support@`, and `reader@waraqah.test`, plus the demo sellers from the fake marketplace (Tanvir, Nabila, Rakib, …) as reader accounts so the marketplace is not empty.
- Only a super admin changes roles (an admin-only endpoint can come later; for now `make set-role EMAIL=… ROLE=…` runs a small Go command).

### 5.2 Tokens

- **Access token:** JWT HS256, signed with `JWT_SECRET` (≥ 32 random bytes), lifetime `JWT_ACCESS_TTL` (default 15 m), claims `sub` (user id), `role`, `iat`, `exp`, `iss` (`waraqah`).
- **Refresh token:** opaque random 32 bytes, stored **hashed** in `refresh_tokens`, lifetime `JWT_REFRESH_TTL` (default 30 days), rotated on every refresh, revoked on logout, password reset and account deletion.
- Every auth answer the app already parses (`/auth/login`, `/auth/google`, `/auth/signup/verify-otp`) returns the **`AppUserModel` fields** `{id, name, email, role}` **plus** `accessToken`, `refreshToken`, `expiresAt`. The extra fields are additive (§4.3); the app stores them after frontend task F2 (§20).

### 5.3 The auth endpoints

| Endpoint | Body (from `auth_remote_source.dart`) | Server behaviour |
|---|---|---|
| `POST /auth/login` | `{email, password}` | bcrypt check; refusal → `200 null` (the app shows "wrong email or password"). Rate-limited. |
| `POST /auth/signup/request-otp` | `{name, contact, password}` | `contact` is an email: store a pending sign-up (name, bcrypt hash), send a 6-digit code (`OTP_TTL`, hashed, `OTP_MAX_ATTEMPTS`). A phone number is refused (`X-Waraqah-Error: phone_not_supported`) because SMS costs money. Answers `{"ok": true}`. |
| `POST /auth/signup/verify-otp` | `{contact, otp}` | Wrong or expired code → `200 null`. Right code → create the user, answer the user + tokens. |
| `POST /auth/password/request-otp` | `{contact}` | Always `{"ok": true}` (don't reveal whether the account exists); send a code if it does. |
| `POST /auth/password/reset` | `{contact, otp, password}` | Wrong code → `200 null` (the app shows "wrong code"); else set the password, revoke refresh tokens, `{"ok": true}`. |
| `POST /auth/google` | `{idToken}` after task F3 (today the app sends no body) | Verify the Google ID token (signature, `aud` ∈ `GOOGLE_OAUTH_CLIENT_IDS`, `iss`, `exp`, verified email). Find or create the user. Answer user + tokens. Without `idToken` → `200 null`. |
| `POST /auth/delete` | `{}` | Soft-delete and anonymise the user, revoke tokens, hide their Listings and Bites. |

In development (`APP_ENV=development`), `OTP_DEV_CODE` (e.g. `123456`, the frontend's demo code) is also accepted, so you can sign up without an email account. **It must be empty in production.**

### 5.4 Guests

The fake backend answers every request as one user, "me", even when the app is signed out. The app only shows user pages to signed-in readers, but some screens still call user-scoped `GET` endpoints for guests (badges, strips). So:

- **`me` GET without a token** → `200` with the **empty value**: `[]` for lists, the empty object the fake returns for a new reader (empty cart, zero wallet, no shelves), `{"unread": 0}` style answers for counters. No `401`.
- **`me` POST or SSE without a token** → `401`.
- **Staff endpoints without a token** → `401`; with a reader token → `403`.

Phase 7 includes a "guest walk" (§21): open every tab signed out against the real backend; any error state is a bug.

---

## 6. Backend architecture

### 6.1 Folder layout

```
waraqah-backend/
├── cmd/
│   ├── api/main.go              wiring only: config → db → services → routes → server
│   ├── seed/main.go             loads seed/*.json into Postgres (idempotent)
│   └── admin/main.go            small ops commands: set-role, prune-tokens
├── internal/
│   ├── app/
│   │   ├── routes.go            mounts every feature's routes (one line each)
│   │   └── deps.go              builds services and passes cross-feature interfaces
│   ├── platform/                shared by all features; never imports a feature
│   │   ├── config/              Config struct, loaded from env (§18)
│   │   ├── db/                  pgx pool, WithTx helper, sqlc Queries
│   │   ├── httpx/               JSON read/write, Null(), Refuse(code), error shape, middleware
│   │   ├── auth/                JWT, refresh tokens, bcrypt, OTP, roles, Google ID token
│   │   ├── sse/                 broker (per-user topics), handler helper
│   │   ├── clock/               Clock interface (tests set the time), Dhaka "today"
│   │   ├── ids/                 id generators (bk-, p2p-, WQ-, HS-, u_)
│   │   ├── cloudinary/          Uploader interface + real/fake implementations
│   │   ├── email/               Sender interface + Resend/Brevo/log implementations
│   │   ├── gemini/              Client interface + real/fake implementations
│   │   └── logx/                slog setup, request-scoped logger
│   └── feature/                 one package per frontend feature (Appendix A)
│       └── p2p/
│           ├── routes.go        every endpoint of the feature, one line each, commented with the fake API constant
│           ├── handlers.go      decode → call service → encode (split by area if >250 lines)
│           ├── service.go       business logic, transactions, calls to other features' interfaces
│           ├── rules.go         ported Dart rules (pure functions) — Appendix B
│           ├── dto.go           JSON structs = the Dart models, with `json:"camelCase"` tags
│           ├── store.go         thin wrapper over sqlc queries
│           ├── *_test.go        rules tests (ported), handler tests, contract tests
│           └── README.md        5–10 lines: what it owns, its frontend files, its tables
├── db/
│   ├── migrations/              goose: 0001_users.sql, 0002_catalog.sql, …
│   ├── queries/                 sqlc input, one file per feature: p2p.sql, orders.sql, …
│   └── sqlc.yaml
├── seed/                        JSON exported from the frontend fixtures (§14)
├── testdata/contract/           golden JSON exported from the fake API (§14)
├── docs/
│   ├── contract-changes.md      every change to the v1 contract
│   ├── error-codes.md           every X-Waraqah-Error code and what causes it
│   └── decisions/               short ADRs (one file per decision)
├── scripts/                     export-contract.sh, smoke.sh
├── docker-compose.yml           local Postgres 16 (and a test DB)
├── Makefile                     every command (§22.4)
├── .golangci.yml
├── .github/workflows/ci.yml
├── go.mod / go.sum
```

### 6.2 Layers and dependency direction

```
handler (HTTP in/out, auth level) → service (rules, transactions, side effects) → store (SQL via sqlc)
```

- **Handlers** decode the query or body, call one service method, and write the result with `httpx.JSON(w, v)`, `httpx.Null(w, code)` or `httpx.Error(w, err)`. No SQL, no business rules.
- **Services** hold the logic ported from the fake stores. They run multi-step changes inside `db.WithTx(ctx, func(q *db.Queries) error {…})`, and publish SSE events and notifications **after** the commit.
- **Stores** are thin wrappers over sqlc's generated `Queries`.
- **Features never import each other's internals.** A feature that needs another one gets an **interface** in its constructor, built in `internal/app/deps.go`, e.g. `notifications.Sender`, `wallet.Ledger`, `catalog.Books`, `blocks.Checker`. This mirrors the frontend rule "use another feature only through its public providers, use cases or entities".
- `internal/platform` never imports `internal/feature`.

### 6.3 Naming that works across both repos

- Go package = frontend feature folder without underscores (`handled_sale` → `handledsale`, `ai_assistant` → `assistant`, `admin` dashboard → `dashboard`).
- Handler method = the fake API constant name: `P2pFakeApi.save` → `(*Handler).Save`, `InboxFakeApi.decide` → `Decide`. `routes.go` comments each line with the constant, so grepping `P2pFakeApi.save` finds both the Dart and the Go side.
- DTO type = Dart model name without `Model`: `P2pListingModel` → `P2pListing`.
- Rule functions keep the Dart names: `ListingRules.check` → `rules.CheckListing`, `SaleMath.sellerGets` → `SellerGets`.

---

## 7. Database

PostgreSQL, `snake_case` columns, `text` ids with prefixes (§4.3), money `integer` taka, timestamps `timestamptz`, enums as `text` with `CHECK` constraints (easier to evolve than Postgres enums), JSONB only for snapshots and small free-form data.

### 7.1 Tables by area

**Accounts and profile (P1)**
- `users(id, email citext unique, phone, name, password_hash, role, photo_url, area, district, member_since, strikes int, banned bool, deleted_at, created_at)`
- `pending_signups(contact, name, password_hash, created_at)`, `otp_codes(contact, purpose, code_hash, expires_at, attempts)`
- `refresh_tokens(id, user_id, token_hash, expires_at, revoked_at)`
- `profile_prefs(user_id pk, muted text[], profile_visible bool, reading_visible bool)`
- `addresses(id, user_id, label, recipient, phone, division, district, upazila, line, is_default)` — one default per user (partial unique index)
- `geo_*` seeded tables or a static JSON served from `seed/geo.json` (read-only, cached in memory)

**Notifications (P1)** — `notifications(id, user_id, kind, params jsonb, target jsonb, read_at, created_at)`

**Catalog (P2)**
- `categories(id, section, name_en, name_bn)`, `authors(id, name, name_bn, bio…)`, `publishers(…)`, `subjects(id, section, name_en, name_bn)`
- `books(id, title, title_bn, short_title, author_id, publisher_id, category_id, section, original_language, added_at, rating numeric(2,1), tags text[], cover_seed, hidden, classes int[], exams text[], subject_id, phonetic_keys text[])`
- `editions(id, book_id, format, language, price_bdt, list_price_bdt, stock, is_preorder, release_date, isbn)`
- `book_details(book_id pk, data jsonb)`, `look_inside(book_id pk, data jsonb)`, `series(id, title, entries jsonb)`
- `book_questions(id, book_id, user_id, text, created_at)`, `book_answers(id, question_id, user_id, text, created_at)`
- `collections(id, title, title_bn, note, section, expert_id, book_ids text[] ordered, position)`, `experts(…)`
- `booklists(id, owner_id null for staff, kind, title, book_ids text[], updated_at)`
- `banners(id, position, title, title_bn, subtitle, subtitle_bn, target jsonb, season)`, `app_config(key pk, value jsonb)` (forced Season)
- `price_lows(edition_id, low_bdt, since)` (30-day low badge), `ayahs(day_of_year pk, …)`
- `sales_by_month(book_id, month, copies)` (bestsellers)

**Buying new (P3)**
- `cart_lines(user_id, kind, ref_id, quantity, saved_for_later, added_at)` — kinds: edition, certifiedUsed, listing, bundle
- `wishlist_items(user_id, book_id, added_at)`, `wishlist_shares(id, user_id)`
- `alerts(id, user_id, book_id, edition_id, kind, target_price_bdt, fired_at)`
- `coupons(code pk, kind, amount, min_order_bdt, expires_at, created_by)`
- `flash_sales(…)`, `bundles(id, title, book_ids, price_bdt)`
- `orders(number pk 'WQ-…', user_id, status, placed_at, address jsonb snapshot, delivery_fee_bdt, payment, subtotal_bdt, discount_bdt, points_used, wallet_used_bdt, total_bdt, gift jsonb, gift_wrap_bdt, is_donation, donate_place_id)`
- `order_lines(order_number, book_id, edition_id, title, author, quantity, unit_price_bdt, kind)`, `order_history(order_number, status, at)`, `order_returns(order_number, reason, note, photos jsonb, status, requested_at, decided_at)`
- `wallet_entries(id, user_id, amount_bdt, reason, order_number, note, at)` — the balance is a sum, never a stored number
- `points_entries(id, user_id, points, reason, order_number, at)`
- `donate_places(id, name, kind, district, area, story, created_at, removed_at)`, `donate_needs(place_id, book_id, wanted, received)`

**Second-hand and moderation (P4)**
- `listings(id, seller_id, book_id, title, condition, flags text[], note, price_bdt, negotiable, handover, status, rejection_reason, cover_seed, category_id, section, new_price_bdt, buyer_id, created_at, updated_at)`
- `listing_photos(listing_id, slot, url, public_id)`
- `threads(id, listing_id, buyer_id, seller_id, read_by_buyer int, read_by_seller int, updated_at)` — unique `(listing_id, buyer_id)`
- `messages(id, thread_id, author_id null = system, at, text, event, amount_bdt, offer_id, removed_at)`
- `offers(id, thread_id, amount_bdt, handover, status, created_at)`
- `ratings(listing_id, from_id, to_id, stars, comment, at)` — unique `(listing_id, from_id)`
- `blocks(user_id, blocked_id, at)`
- `reports(id, reporter_id, kind, target_id, reason, note, status, created_at, closed_at)`
- `moderation_log(id, staff_id, staff_name, action, subject, why, at)`
- `book_requests(id, user_id, title, author, book_id, max_price_bdt, note, open, created_at)`
- `handled_sales(id 'HS-…', listing_id, buyer_id, seller_id, price_bdt, delivery_bdt, method, status, created_at, sent_at, completed_at, dispute jsonb)`, `payouts(id, user_id, amount_bdt, at)`
- `sell_backs(id, user_id, book_id, condition, flags, pickup_address jsonb, status, quote_bdt, grade, paid_bdt, created_at, …)`, `certified_used(id, book_id, edition_id, grade, price_bdt, sold)`

**Community and reading (P5)**
- `bites(id, author_id, text, book_id, spoiler, created_at, edited_at, deleted_at)`, `bite_likes(bite_id, user_id)`, `bite_comments(id, bite_id, parent_id, author_id, text, created_at, deleted_at)`
- `reviews(book_id, user_id, stars, text, verified, created_at, updated_at)` — primary key `(book_id, user_id)`; the book's `rating` is recomputed in the same transaction
- `follows(follower_id, followee_id, at)`
- `shelf_entries(user_id, book_id, shelf, added_at, finished_at, progress, pages_read, total_pages)`, `shelf_order_sync(user_id, book_id)` (a delivered Book lands on Want to Read only once), `reading_days(user_id, day date)`, `reading_goals(user_id, year, goal)`

**Admin (P6)** — `search_log(term, day date, count)`

### 7.2 Rules for migrations

- One migration per PR per feature, named `NNNN_<feature>_<what>.sql`, with `-- +goose Up` and `-- +goose Down`.
- Never edit a merged migration; add a new one.
- Every foreign key has an index. Every list query in Appendix A has an index that supports its `WHERE` and `ORDER BY`.
- `make migrate` runs locally; production runs them on start when `RUN_MIGRATIONS_ON_START=true` (free tier; there's no separate release step).

---

## 8. Cross-feature services

These are interfaces built once in `internal/app/deps.go` and passed to the features that need them. They correspond to the shared fake stores in the frontend's `app/fake_stores.dart`.

| Interface | Owner (package) | Used by | Frontend equivalent |
|---|---|---|---|
| `notifications.Sender` — `Send(ctx, userID, kind, params, target)` | notifications | orders, moderation, handled sales, Sell Back, alerts, book requests, bites, follows | `NotificationFakeStore.send`, `NotificationSends` |
| `wallet.Ledger` — `Credit`, `Spend`, `Balance` | wallet | orders, checkout, handled sales, Sell Back | `WalletFakeStore` |
| `points.Ledger` | loyalty | checkout, orders | `PointsFakeStore` |
| `catalog.Books` — find, editions, price and stock changes | catalog | almost everyone | `BookFixtures`, `bookRepositoryProvider` |
| `blocks.Checker` — `IsBlocked(viewer, other)` | report | p2p, inbox, bites, readers | `ReportFakeStore.isBlocked` |
| `moderation.Bans` — `IsBanned(user)` | moderation | p2p, bites, reviews | `ModerationFakeStore.isBanned` |
| `listings.Status` — reserve, sell, release | p2p | inbox, handled sales | `P2pFakeStore.setStatus` |
| `orders.Delivered` — delivered lines per user | orders | reviews (Verified Purchase), shelves | `OrderFakeStore` |
| `alerts.Sweeper` — runs after price/stock changes | alerts | catalog admin | `AlertFakeStore.sweep` |
| `search.Log` | dashboard | catalog `/books` | `SearchLog` |

Notification texts are **not** made on the server: the server stores `kind` and `params`, and the app builds the words from its ARB files (`notificationText`). Keep the same kinds and params as `../waraqah-frontend/lib/features/notifications/domain/entities/notification_kind.dart` and the `NotificationSends` helpers.

---

## 9. Live updates (SSE)

- `internal/platform/sse.Broker`: in-process, topics per user id (`inbox:<uid>`, `sales:<uid>`, `notifications:<uid>`, plus `sales:moderators`). One Render instance means in-process is enough.
- Services call `broker.Publish(topic, payload)` **after** the transaction commits.
- The handler (`sse.Serve(w, r, topic)`) sets the headers, flushes each event, sends `: ping` every 25 s, and ends cleanly when the client disconnects.
- `seq` is a per-process counter. The app only uses events as "something changed, reload" signals, so a lost event after a restart is harmless (the app reloads on its next open).
- Scaling later: replace the broker with Postgres `LISTEN/NOTIFY` behind the same interface.

---

## 10. Search

- `GET /books` keeps the fake API's behaviour: exact matches first, then phonetic matches (`BookSearchMatch`), then the filters and sorts in `CatalogFilters` (`../waraqah-frontend/lib/features/catalog/domain/entities/catalog_filters.dart`).
- **Port `PhoneticKey`** (`../waraqah-frontend/lib/features/catalog/data/sources/phonetic_key.dart`) to `internal/feature/catalog/phonetic.go` with its tests (`phonetic_key_test.dart`). Store keys for title, Bangla title, Author and Publisher in `books.phonetic_keys` and index them (GIN).
- Use the `pg_trgm` extension for "Did you mean…?" (`/books/did-you-mean`) and suggestions (`/books/suggest`), answering in the script the reader typed, as the fake does.
- Count reader searches like `SearchLog`: growing live-search terms from the same user within 2 minutes count once; Staff's `includeHidden` searches don't count.

---

## 11. Images (Cloudinary)

- The app sends photos as base64 (listings, returns, disputes). The server:
  1. decodes, checks it is a JPEG/PNG/WebP by its bytes (not the name), and enforces `MAX_IMAGE_MB`;
  2. uploads with the **signed** Admin API (`CLOUDINARY_API_KEY` / `CLOUDINARY_API_SECRET`) into `CLOUDINARY_FOLDER/<kind>/<id>/<slot>`;
  3. stores the `secure_url` and `public_id`.
- Listing responses keep today's `photos: ["front", "back", …]` (slot names), so the app keeps working. **Contract v1.1** adds `photoUrls: {"front": "https://res.cloudinary.com/…"}` together with a frontend task that shows the real photos (F7).
- Thumbnails come from URL transformations (`c_fill,w_300,q_auto,f_auto`), so bandwidth stays low.
- Removing a photo slot, deleting a draft, or a moderator removing a Listing deletes the Cloudinary asset (best effort, logged on failure).
- In tests and development without keys, `cloudinary.Fake` stores nothing and returns `https://example.invalid/<public_id>` URLs.

---

## 12. Background jobs and demo behaviour

The fake backend has demo-only behaviour: the other person in a thread answers after about 4 s, a demo seller sends a handled-sale book after about 4 s, a courier picks up a Sell Back book after 4 s, and some notifications are seeded. On a real backend the other people are real users. So:

| Fake behaviour | Real backend |
|---|---|
| Other person replies or rates in a thread | Real users do it. With `DEMO_MODE=true`, a "demo bot" acts **only for seeded demo accounts** (so one person can present the app). |
| Demo seller sends a handled-sale book | Real seller uses `POST /sales/step`. `DEMO_MODE` bot does it for demo sellers. |
| Courier picks up a Sell Back book | Simulated courier job: `scheduled` → `pickedUp` after `COURIER_PICKUP_DELAY`, always on until a real courier integration exists. |
| Seeded community notifications | Seed data only. |
| `AlertFakeStore.sweep` after Staff price/stock changes | `alerts.Sweeper` called by the catalog admin service in the same request. |
| Bestsellers over 30 days | `sales_by_month` updated when an order is placed; rankings computed at query time. |

Jobs run on an in-process ticker (`internal/platform/jobs`). Render's free service sleeps when idle, so every job must be **idempotent and catch up on wake** (e.g. "advance every pickup older than the delay", not "advance the one booked 4 s ago").

---

## 13. AI assistant

- Port `AssistantParser`, `assistantPicksFor` and `AssistantReplies` (`../waraqah-frontend/lib/features/ai_assistant/data/sources/`) to `internal/feature/assistant` with the tests in `ai_assistant_test.dart` and `smarter_ai_test.dart`. The endpoints, `/assistant/greeting` and `/assistant/ask`, keep their JSON, including `basket: {editionIds, totalBdt}`.
- With `GEMINI_API_KEY` set, the server asks Gemini to **word** the reply around the same picked Books (never to invent Books, prices or stock); if Gemini fails or is slow (`GEMINI_TIMEOUT`), the rule-based reply stands. Without a key, rule-based replies only.
- Rate-limit `/assistant/ask` per user (`AI_RATE_PER_MIN`) to stay inside Gemini's free tier.
- Frontend task F4 removes the app's direct Gemini call (`gemini_chatbot.dart`) so no key ships in the app.

---

## 14. Seed data and the contract export

One small frontend change (F5, §20) adds an exporter that runs on the fake backend and writes JSON files. It produces two things:

1. **`testdata/contract/`** — for every row in Appendix A, one golden response (`GET__p2p__listing.json`, `POST__p2p__listings__save.json`, …) from a sample request listed in the exporter. These are the **shape goldens** for the contract tests (§15).
2. **`seed/`** — the fixture lists as JSON: books with Editions, categories, authors, publishers, series, collections, experts, booklists, banners, subjects, seasons, ayahs, geo, demo people and their Listings, orders, threads, Bites, reviews, donation places, coupons, deals.

Commands:

```bash
make frontend-sync      # git -C frontend pull
make contract-export    # runs the exporter in ../waraqah-frontend and copies its output here
make seed               # loads seed/*.json into the database (idempotent upserts)
```

Rules: seeds never contain real people's data; seeded passwords come from `SEED_DEMO_PASSWORD`; `make seed` refuses to run against production unless `SEED_ALLOW_PRODUCTION=true`.

---

## 15. Testing

| Level | What | Where |
|---|---|---|
| Rules | Table tests ported from the frontend rules tests (`../waraqah-frontend/test/*_rules_test.dart`, `fair_price_test.dart`, `phonetic_key_test.dart`, …). Same inputs, same answers. | `internal/feature/<f>/rules_test.go` |
| Handlers | `httptest` against a real local Postgres (`docker compose up db-test`), each test in a transaction that rolls back. | `internal/feature/<f>/handlers_test.go` |
| Contract | For each golden in `testdata/contract`, call the same request and compare **shapes**: same keys, same JSON kinds (object/array/string/number/bool), `null` where the golden is `null`. Values may differ. | `internal/contract/contract_test.go` |
| End to end | Run the backend locally and the app with `--dart-define=API_BASE_URL=http://localhost:8080/v1`; walk the feature's screens. | manual checklist per PR |
| Frontend | Unchanged: the 562 widget and unit tests keep running on the fake API. | frontend repo |

`make test` runs rules, handlers and contract tests. CI runs them with a Postgres service container. **A PR that adds an endpoint must make its contract test pass.**

---

## 16. Debugging and observability

- **Request IDs:** every request gets `X-Request-ID` (kept if the client sent one), returned in the response and on every log line.
- **Structured logs:** `slog` JSON with `request_id`, `user_id`, `role`, `method`, `route`, `status`, `latency_ms`, and `refusal` for `200 null` refusals. `LOG_LEVEL` and `LOG_FORMAT` (`json` or `text` for local reading).
- **Refusal codes:** every `200 null` refusal sets `X-Waraqah-Error` to a stable code (`listing_not_editable`, `offer_below_minimum`, `blocked_reader`, `wrong_otp`, …). Each code lives in `internal/platform/httpx/errors.go` and is documented in `docs/error-codes.md` with the Dart rule it mirrors. An agent debugging "the app shows a generic error" reads the header and greps the code.
- **Health:** `/healthz` (process up), `/readyz` (database reachable), `/version` (git commit, build time, contract version).
- **Never log** tokens, passwords, OTP codes, base64 images or full request bodies. Log sizes and ids instead.
- `DEBUG_SQL=true` logs queries with timings in development only.
- `make logs` tails a local server; on Render use the dashboard log stream.

---

## 17. Security

- Passwords: bcrypt (`BCRYPT_COST`, default 12). OTPs and refresh tokens stored hashed (SHA-256).
- JWT: HS256 with `JWT_SECRET` ≥ 32 random bytes; short access tokens; rotated refresh tokens.
- Authorization in services, not only handlers: every change checks ownership (`seller_id = me`), state (Listing status), blocks and bans, exactly like the fake stores.
- Rate limits (in-memory, per IP and per user): auth endpoints, OTP requests (`OTP_RESEND_SECONDS`), `/assistant/ask`, report creation.
- Input limits: body size, string lengths from the rules classes, image type and size.
- SQL only through sqlc (parameterised). No string-built SQL.
- CORS: only `CORS_ALLOWED_ORIGINS` (the Flutter web origins and localhost ports in development).
- HTTPS is provided by Render. Cookies are not used (tokens travel in the `Authorization` header), so CSRF doesn't apply.
- Secrets only in environment variables; `.env` is git-ignored; CI uses GitHub secrets.
- Payments stay simulated (bKash / Nagad / card are recorded, not charged), as in the app. Real gateways are out of scope.

---

## 18. Configuration and secrets

All configuration comes from environment variables, loaded once into `config.Config` at start; missing required values stop the server with a clear message naming the variable. Locally, `.env` is loaded automatically in development. **`.env.example` lists every variable with a placeholder and a comment; keep it in sync with `config.go` in the same PR** (CI checks that every field has an entry).

Required in production: `DATABASE_URL`, `JWT_SECRET`, `CORS_ALLOWED_ORIGINS`, `EMAIL_API_KEY`, `EMAIL_FROM`, `CLOUDINARY_*`, `GOOGLE_OAUTH_CLIENT_IDS`. Optional: `GEMINI_API_KEY` (rule-based AI without it).

See `.env.example` for the full list.

---

## 19. Deployment on free tiers

| Piece | Service | Free limits to respect |
|---|---|---|
| API | Render free web service (Go native build or Docker) | 750 h/month, sleeps after 15 min idle, ~1 min cold start. Open the app a minute before a demo. |
| Database | Neon free | 0.5 GB, 100 CU-hours/month; use the **pooled** URL for the app (`DATABASE_URL`) and the direct URL for migrations (`DATABASE_URL_DIRECT`). |
| Images | Cloudinary free | 25 credits/month (storage + bandwidth + transformations). Thumbnails via transformations; delete removed photos. |
| Email | Resend (or Brevo) free | Daily and monthly email caps; OTP only. |
| AI | Gemini free tier | Per-minute and per-day caps; rate-limit per user. |
| Web app (optional) | Render static site or GitHub Pages | Serve `flutter build web` with `API_BASE_URL` set. |

CI (GitHub Actions, free for public repos and generous for private ones): `gofmt` check, `go vet`, `golangci-lint`, `sqlc diff`, migrations up/down on a fresh Postgres, `go test ./...`. Render auto-deploys `main` after CI passes.

---

## 20. Frontend integration tasks

Small PRs in the **frontend** repo. Each keeps the fake API as the default, so all frontend tests still pass.

| ID | Task | Frontend files | Owner |
|---|---|---|---|
| F1 | `API_BASE_URL` from `--dart-define`; when empty, keep the `FakeApiInterceptor` (today's behaviour); when set, no interceptor. | `core/network/api_config.dart`, `dio_client.dart`, `app/app_bootstrap.dart` | Rahinur (core) |
| F2 | Tokens: add `accessToken`, `refreshToken`, `expiresAt` to `AppUserModel` and the session store; an `AuthInterceptor` adds `Authorization: Bearer …`, refreshes on `401` via `/auth/refresh` once, then signs out. | `features/auth/data/…`, `core/network/` | Rahinur |
| F3 | Google sign-in sends `{idToken}` from `google_sign_in` to `/auth/google`. | `features/auth/…` | Rahinur |
| F4 | Remove the direct Gemini call (`gemini_chatbot.dart`); the server words replies. | `features/ai_assistant/…` | Arifin |
| F5 | Contract and seed exporter: a test-only tool that runs every Appendix A request on the fake backend and writes `build/contract/*.json` and `build/seed/*.json`. | `test/tool/export_contract_test.dart` | Arifin |
| F6 | SSE robustness: reconnect with backoff when a live stream ends; ignore `:` comment lines (heartbeats). | `inbox_live_source.dart`, `sale_live_source.dart`, notifications live source | Farhan (inbox), Arifin (sales), Rahinur (notifications) |
| F7 | (contract v1.1) Show real listing photos from `photoUrls` when present. | `features/p2p/…`, moderation photo strip | Arifin |
| F8 | Sign-up copy: email only (no SMS); keep phone as a profile field. | `features/auth/…`, ARB | Rahinur |

F1, F2 and F5 are needed before Phase 1 ends; the rest can follow their phase.

---

## 21. Work plan, phases and owners

Ownership mirrors the frontend (frontend `AGENTS.md` §7). Each phase lists its tickets; each ticket is one branch and one PR. Appendix A gives every endpoint's owner and phase.

### Phase 0 — Foundations (Arifin, with Rahinur for config review)
- **B0.1** Repo skeleton (§6.1), `go.mod`, Makefile, `.golangci.yml`, CI, `docker-compose.yml`, `.env.example`, README.
- **B0.2** `platform/config`, `platform/logx`, `platform/httpx` (JSON, `Null`, `Refuse`, error shape, request-ID, logging, recover, CORS, body limit middleware), `/healthz`, `/readyz`, `/version`.
- **B0.3** `platform/db` (pool, `WithTx`, sqlc setup, goose), first migration (`users`).
- **B0.4** `platform/auth` (JWT, refresh, bcrypt, OTP, roles, Google verifier) with tests; auth middleware levels `public`, `me`, `staff`, `staff:<perm>`.
- **B0.5** `platform/sse`, `platform/clock`, `platform/ids`, fakes for Cloudinary, email and Gemini.
- **B0.6** Contract test harness (§15) + frontend F5 exporter + `make contract-export`, `make seed`.
- **Done when:** `make test` passes, `/readyz` is green against Neon, a sample endpoint passes its contract test.

### Phase 1 — Accounts and notifications (Rahinur)
- **B1.1** `auth` endpoints (§5.3) + demo accounts seed. **F1, F2** in the frontend.
- **B1.2** `profile` (profile, prefs, addresses, geo, delete account).
- **B1.3** `notifications` (list, read, read all, live) + `notifications.Sender` for everyone else.

### Phase 2 — Catalog and storefront (Rahinur; scan: Arifin)
- **B2.1** Catalog reads: books (search, filters, sorts, phonetic), detail, details, look inside, questions, series, price lows, used options, suggest, did-you-mean.
- **B2.2** Records: categories, authors, publishers, series detail, subjects, collections, experts, booklists (+ a Reader's own).
- **B2.3** Home: banners, season, ayah.
- **B2.4** Catalog admin and tools (+ `alerts.Sweeper` hook stub).
- **B2.5** Scan lookup (Arifin).

### Phase 3 — Buying new (Farhan; donation places admin: Arifin)
- **B3.1** Cart, deals, wishlist (+ share), alerts.
- **B3.2** Checkout: coupons, `CheckoutTotals`, place order, points, wallet spend, gifts, `sales_by_month`.
- **B3.3** Orders: list, detail, cancel, return (with photos), reorder; admin orders and coupons; refunds to the wallet.
- **B3.4** Wallet and points pages.
- **B3.5** Donate (recipients, give) and Admin → Donation places (Arifin).

### Phase 4 — Second-hand and moderation (Arifin; inbox: Farhan)
- **B4.1** Listings: marketplace, mine, for book, detail, seller page, save (rules, photos to Cloudinary).
- **B4.2** Reports and blocks.
- **B4.3** Moderation Center: queue, decide, reports, act (remove deletes messages, Bites, comments, reviews), log, strikes and bans.
- **B4.4** Inbox (Farhan): threads, open, send, offer, decide, read, release, sold, rate, live; blocks enforced.
- **B4.5** Book requests (+ demand, wanted, seller notifications).
- **B4.6** Handled sales: buy, steps, disputes, settle, earnings, payouts, live; `saleRefund` wallet credits.
- **B4.7** Sell Back + Certified Used + Trade-ins + courier job.

### Phase 5 — Community and reading (Rahinur; shelves: Arifin)
- **B5.1** Bites (feeds, post, edit, delete, like, comments with one level of replies; blocks and bans).
- **B5.2** Reviews (+ Verified Purchase, rating recompute) and readers/follows.
- **B5.3** Shelves, progress, reading stats and goal (Arifin).

### Phase 6 — AI and dashboard (Arifin)
- **B6.1** Assistant (port parser, picks, basket, replies; optional Gemini). **F4**.
- **B6.2** Admin dashboard + search log.

### Phase 7 — Hardening and launch (everyone)
- Guest walk (§5.4), staff role walk (each demo account opens only its sections), full contract run, rate limits, indexes checked with `EXPLAIN`, free-tier budget check, Render deploy, Flutter web build pointing at production, **F6**, **F7**, **F8**, update both repos' AGENTS.md.

### Ticket template (copy into each issue / agent prompt)

```markdown
### B4.1 Listings
Owner: Arifin · Phase 4 · Branch: feature/listings
Endpoints (Appendix A → p2p): GET /p2p/listings, GET /p2p/listings/mine, GET /p2p/listings/for-book,
  GET /p2p/listing, GET /p2p/seller, POST /p2p/listings/save
Frontend reference: ../waraqah-frontend/lib/features/p2p/data/sources/{p2p_fake_api,p2p_fake_store,p2p_listing_writer}.dart,
  ../waraqah-frontend/lib/features/p2p/domain/entities/listing_rules.dart, ../waraqah-frontend/test/listing_*_test.dart
Tables: listings, listing_photos (migration NNNN_p2p_listings.sql)
Depends on: B0.*, B1.1, B2.1 (books), blocks.Checker (B4.2, stub until then)
Done when:
- [ ] Every endpoint above passes its contract test
- [ ] rules_test.go ports listing_rules_test.dart cases
- [ ] Refusals use X-Waraqah-Error codes listed in docs/error-codes.md
- [ ] App walk: sell a book, save a draft, send for review, see it in My Listings (local backend)
- [ ] BACKEND_PLAN.md / feature README updated if anything differs
```

---

## 22. Rules for coding agents

These are repeated, shorter, in `AGENTS.md`.

### 22.1 Before writing code
1. Read this plan's §4 (contract) and the ticket.
2. Open the frontend reference for the endpoint: the fake API file, the fake store, the model, the rules class and its test (paths in Appendix A and C).
3. If the fake API and this plan disagree, **the fake API wins** for v1; note the difference in the PR.

### 22.2 While writing code
- Follow §6: handler → service → store; features talk through interfaces.
- One endpoint = one route line + one handler method + one service method + its SQL in `db/queries/<feature>.sql`.
- Files stay small: aim for ≤ 200 lines, hard limit 300 (split by area: `handlers_listings.go`, `handlers_seller.go`).
- Names follow §6.3, so both repos grep to each other.
- Every refusal goes through `httpx.Refuse(w, code)` with a code in `errors.go` and `docs/error-codes.md`.
- No new dependency without a line in `docs/decisions/` and the PR description.
- No secrets, tokens or real emails in code, tests or logs.

### 22.3 Before opening a PR
- `make check` (format, vet, lint, sqlc diff, tests, contract tests) passes.
- New env variables are in `.env.example` and `config.go`.
- New migrations go up and down cleanly (`make migrate-down-up`).
- The feature's `README.md` and, if needed, this plan are updated.

### 22.4 Make targets

| Target | Does |
|---|---|
| `make dev` | start Postgres (Docker), run migrations, run the API with live `.env` |
| `make test` | unit, handler and contract tests |
| `make check` | `gofmt`, `go vet`, `golangci-lint`, `sqlc diff`, `make test` |
| `make migrate` / `make migrate-down-up` | apply migrations / prove the latest is reversible |
| `make sqlc` | regenerate the query code |
| `make seed` | load `seed/*.json` |
| `make frontend-sync` | update `../waraqah-frontend` (`git pull`) |
| `make contract-export` | refresh `testdata/contract` and `seed` from the frontend exporter |
| `make set-role EMAIL=… ROLE=…` | change a user's role (local or `DATABASE_URL` target) |
| `make smoke` | curl a handful of endpoints against `API_BASE_URL` |

### 22.5 Definition of done for an endpoint
Same method and path as Appendix A · same JSON shape as the golden · same `null` meaning · the same rules as the fake store (ported tests) · auth level enforced · refusals coded · indexed queries · logs without secrets.

---

## 23. Open questions

1. **Instructor sign-off** on Go + Neon + Render + Cloudinary + Resend, and on a separate `waraqah-backend` repo (the frontend AGENTS.md already says "A Go backend will come later, in a separate repository").
2. **Email-only sign-up** (no SMS): confirm with the team and the instructor (F8).
3. **Payments stay simulated** for the project: confirm.
4. **Who owns Phase 0** (proposed: Arifin) and the **reviewer rotation** for backend PRs (the frontend's rotation still mentions Niloy).
5. **Contract v2** (proper `4xx` errors instead of `200 null`) after launch, or never.

---

## Appendix A — Endpoint checklist (generated from the frontend)

Generated from `../waraqah-frontend/lib/features/*/data/sources/*_fake_api.dart` (paths and doc comments) and the remote sources (HTTP methods). **This list is the contract.** Every row must exist in Go with the same method and path under `/v1`. The doc comment is copied from the fake API: it describes the query or body and what `null` means.

Auth column: `public` = no token needed. `me` = needs a signed-in user (a GET without a token answers the empty value, see §5.4). `staff` = any staff role. `staff:catalog|orders|moderate` = that permission (super admin passes all).

### `auth` → `internal/feature/auth` · owner **Rahinur** · phase **P1**

Frontend reference: `../waraqah-frontend/lib/features/auth/data/sources/` (`auth_fake_api.dart`), models in `../waraqah-frontend/lib/features/auth/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| POST | `/auth/google` | public | `AuthFakeApi.google` | — |
| POST | `/auth/login` | public | `AuthFakeApi.login` | — |
| POST | `/auth/password/request-otp` | public | `AuthFakeApi.requestPasswordReset` | — |
| POST | `/auth/password/reset` | public | `AuthFakeApi.resetPassword` | — |
| POST | `/auth/signup/request-otp` | public | `AuthFakeApi.requestSignUpOtp` | — |
| POST | `/auth/signup/verify-otp` | public | `AuthFakeApi.verifySignUpOtp` | — |

### `notifications` → `internal/feature/notifications` · owner **Rahinur** · phase **P1**

Frontend reference: `../waraqah-frontend/lib/features/notifications/data/sources/` (`notification_fake_api.dart`), models in `../waraqah-frontend/lib/features/notifications/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/notifications` | me | `NotificationFakeApi.notifications` | The signed-in Reader's notifications, newest first. |
| GET (SSE) | `/notifications/live` | me | `NotificationFakeApi.live` | Server-sent events: one `data: {"unread": n}` line per change. |
| POST | `/notifications/read` | me | `NotificationFakeApi.read` | Body `{id}`. |
| POST | `/notifications/read-all` | me | `NotificationFakeApi.readAll` | — |

### `profile` → `internal/feature/profile` · owner **Rahinur** · phase **P1**

Frontend reference: `../waraqah-frontend/lib/features/profile/data/sources/` (`profile_fake_api.dart`), models in `../waraqah-frontend/lib/features/profile/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/addresses` | me | `ProfileFakeApi.addresses` | The saved addresses, default first. |
| POST | `/addresses/default` | me | `ProfileFakeApi.defaultAddress` | Body `{id}`: answers the addresses. |
| POST | `/addresses/delete` | me | `ProfileFakeApi.deleteAddress` | Body `{id}`: answers the addresses. |
| POST | `/addresses/save` | me | `ProfileFakeApi.saveAddress` | Body: an address; no `id` adds it. Answers the addresses. |
| POST | `/auth/delete` | me | `ProfileFakeApi.deleteAccount` | Ends the account. Under `/auth` on the real server; answered here because Profile's Settings asks for it. |
| GET | `/geo` | public | `ProfileFakeApi.geo` | Every division, its districts and their upazilas. |
| GET | `/profile` | me | `ProfileFakeApi.profile` | `{name, phone, photo?}`; the name is empty until first saved. |
| GET | `/profile/prefs` | me | `ProfileFakeApi.prefs` | `{muted: [group…], profileVisible, activityVisible}`. |
| POST | `/profile/prefs/save` | me | `ProfileFakeApi.savePrefs` | Body: the settings. Answers them saved. |
| POST | `/profile/save` | me | `ProfileFakeApi.saveProfile` | Body `{name, phone, photo?}`: answers the saved profile. |

### `catalog` → `internal/feature/catalog` · owner **Rahinur** · phase **P2**

Frontend reference: `../waraqah-frontend/lib/features/catalog/data/sources/` (`book_fake_api.dart`, `book_questions_fake_api.dart`, `book_suggest_fake_api.dart`, `booklist_fake_api.dart`, `collection_fake_api.dart`), models in `../waraqah-frontend/lib/features/catalog/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/authors/detail` | public | `BookFakeApi.author` | One Author: `?id=<authorId>`, or `null` when unknown. |
| GET | `/booklists` | public | `BooklistFakeApi.booklists` | Every Staff Booklist and the Reader's own. |
| GET | `/booklists/detail` | public | `BooklistFakeApi.detail` | One Booklist: `?id=`, or `null` when unknown. |
| POST | `/booklists/mine/delete` | me | `BooklistFakeApi.deleteMine` | Body `{id}` → `{id}`, or `null` when it isn't the Reader's own. |
| POST | `/booklists/mine/save` | me | `BooklistFakeApi.saveMine` | Body `{id?, name?, bookIds?}`: no `id` makes a new own list (`name` needed); with an `id`, renames it and/or replaces its books. Answers the list, or `null` when refused. |
| GET | `/books` | public | `BookFakeApi.books` | Books on the storefront, narrowed by the query. Hidden Books only with `includeHidden=true` (Staff's list). |
| GET | `/books/detail` | public | `BookFakeApi.book` | One Book, even a hidden one: `?id=<bookId>`, or `null` when unknown. |
| GET | `/books/details` | public | `BookFakeApi.bookDetails` | One book's summary, page count and reviews: `?id=<bookId>`. Answers `null` when there's nothing extra for that book. |
| GET | `/books/did-you-mean` | public | `BookSuggestFakeApi.didYouMean` | The one title closest to `?q=` (for "Did you mean…?"), or `null`. |
| GET | `/books/look-inside` | public | `BookFakeApi.lookInside` | Table of contents and sample pages: `?id=<bookId>`, or `null`. |
| GET | `/books/price-lows` | public | `BookFakeApi.priceLows` | Each Edition's lowest price in the last 30 days: `?id=<bookId>`. |
| GET | `/books/questions` | public | `BookQuestionsFakeApi.questions` | `?id=<bookId>`. |
| POST | `/books/questions/answer` | me | `BookQuestionsFakeApi.answer` | Body: `{bookId, questionId, text, name, isStaff}`. |
| POST | `/books/questions/ask` | me | `BookQuestionsFakeApi.ask` | Body: `{bookId, text, name}`. |
| GET | `/books/series` | public | `BookFakeApi.series` | The series a book is in, in reading order: `?id=<bookId>`, or `null`. |
| GET | `/books/suggest` | public | `BookSuggestFakeApi.suggest` | Up to 5 Book titles and Author names for `?q=`, best first. |
| GET | `/books/used-options` | public | `BookFakeApi.usedOptions` | Certified Used and reader copies of a book, and its resale estimate: `?id=<bookId>`. Answers `null` for an unknown book. |
| GET | `/categories` | public | `BookFakeApi.categories` | A Section's Categories: `?section=<section>`. |
| GET | `/collections` | public | `CollectionFakeApi.collections` | Every Collection. Narrow with `?section=<Section name>`, `?expert=<expertId>` and `?hasExpert=true\|false`. |
| GET | `/collections/detail` | public | `CollectionFakeApi.detail` | One Collection: `?id=<collectionId>`, or `null` when unknown. |
| GET | `/experts` | public | `CollectionFakeApi.experts` | Every Expert. |
| GET | `/experts/detail` | public | `CollectionFakeApi.expert` | One Expert with their `collections`: `?id=<expertId>`, or `null`. |
| GET | `/publishers/detail` | public | `BookFakeApi.publisher` | One Publisher: `?id=<publisherId>`, or `null` when unknown. |
| GET | `/series/detail` | public | `BookFakeApi.seriesDetail` | One Series: `?id=<seriesId>`, or `null` when unknown. |
| GET | `/subjects` | public | `BookFakeApi.subjects` | Every Subject, or only those with Books in `?section=<section>`. |

### `catalog_admin` → `internal/feature/catalogadmin` · owner **Rahinur** · phase **P2**

Frontend reference: `../waraqah-frontend/lib/features/catalog_admin/data/sources/` (`catalog_admin_fake_api.dart`, `catalog_tools_fake_api.dart`), models in `../waraqah-frontend/lib/features/catalog_admin/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/admin/catalog/banners` | staff:catalog | `CatalogAdminFakeApi.banners` | Every Banner, every Season's too, in display order. |
| POST | `/admin/catalog/banners/delete` | staff:catalog | `CatalogAdminFakeApi.deleteBanner` | Body `{id}` → every Banner. |
| POST | `/admin/catalog/banners/move` | staff:catalog | `CatalogAdminFakeApi.moveBanner` | Body `{id, by: -1 \| 1}` → every Banner. |
| POST | `/admin/catalog/banners/save` | staff:catalog | `CatalogAdminFakeApi.saveBanner` | Body: a Banner (empty `id` = new) → every Banner. |
| POST | `/admin/catalog/booklists/delete` | staff:catalog | `CatalogAdminFakeApi.deleteBooklist` | Body `{id}` → `{id}`; a Reader's own list is refused. |
| POST | `/admin/catalog/booklists/save` | staff:catalog | `CatalogAdminFakeApi.saveBooklist` | Body: a `ListDraft` with a Staff `kind` (no `id` = new) → `{id}`. |
| POST | `/admin/catalog/books/hide` | staff:catalog | `CatalogAdminFakeApi.hideBook` | Body `{id, hidden}` → the Book. |
| POST | `/admin/catalog/books/save` | staff:catalog | `CatalogAdminFakeApi.saveBook` | Body: a `BookDraft` (no `id` = new) → the saved Book. |
| POST | `/admin/catalog/collections/delete` | staff:catalog | `CatalogAdminFakeApi.deleteCollection` | Body `{id}` → `{id}`. |
| POST | `/admin/catalog/collections/save` | staff:catalog | `CatalogAdminFakeApi.saveCollection` | Body: a `ListDraft` without `kind` (no `id` = new) → `{id}`. |
| POST | `/admin/catalog/editions/stock` | staff:catalog | `CatalogToolsFakeApi.editionStock` | Body `{editionId, stock}` → the same; `null` for an eBook or a negative stock. |
| POST | `/admin/catalog/import` | staff:catalog | `CatalogToolsFakeApi.importBooks` | Body `{books: [{row, author, publisher, ...BookDraft}]}` → `{imported, skipped: [{row, reason}]}`. See [CatalogImportFake.run]. |
| GET | `/admin/catalog/isbn-lookup` | staff:catalog | `CatalogToolsFakeApi.isbnLookup` | `?isbn=<ISBN-13>` → `{inCatalog: true, bookId}` when an Edition has it, a Book from outside (`inCatalog: false`, title, author, publisher, language, format, listPriceBdt…), or `null`. |
| GET | `/admin/catalog/low-stock` | staff:catalog | `CatalogToolsFakeApi.lowStock` | Printed Editions (no eBooks or pre-orders) at or under `CatalogAdminRules.lowStock`, lowest first: `{bookId, title, coverSeed, editionId, format, language, stock}`. |
| GET | `/admin/catalog/season` | staff:catalog | `CatalogAdminFakeApi.season` | `{season: <name> \| null}`: the Season Staff forced on Home, `null` = automatic (by date). Add `/save` (same body) to change it. |
| POST | `/admin/catalog/season/save` | staff:catalog | `CatalogAdminFakeApi.season` + `/save` | Body `{season: <name> \| null}` → the same; `null` returns Home to automatic. |
| GET | `/admin/catalog/authors` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.author)` | Every one, with `bookCount`. |
| POST | `/admin/catalog/authors/save` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.author)` + `/save` | Body: the record (no `id` = new). |
| POST | `/admin/catalog/authors/delete` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.author)` + `/delete` | Body `{id}`; refused while a Book uses it. |
| GET | `/admin/catalog/categories` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.category)` | Every one, with `bookCount`. |
| POST | `/admin/catalog/categories/save` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.category)` + `/save` | Body: the record (no `id` = new). |
| POST | `/admin/catalog/categories/delete` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.category)` + `/delete` | Body `{id}`; refused while a Book uses it. |
| GET | `/admin/catalog/publishers` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.publisher)` | Every one, with `bookCount`. |
| POST | `/admin/catalog/publishers/save` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.publisher)` + `/save` | Body: the record (no `id` = new). |
| POST | `/admin/catalog/publishers/delete` | staff:catalog | `CatalogAdminFakeApi.records(RecordKind.publisher)` + `/delete` | Body `{id}`; refused while a Book uses it. |

### `home` → `internal/feature/home` · owner **Rahinur** · phase **P2**

Frontend reference: `../waraqah-frontend/lib/features/home/data/sources/` (`ayah_fake_api.dart`, `home_fake_api.dart`), models in `../waraqah-frontend/lib/features/home/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/home/banners` | public | `HomeFakeApi.banners` | The active Season's Banners first, then the all-year ones, in display order. Other Seasons' Banners are left out. |
| GET | `/home/season` | public | `HomeFakeApi.season` | The active Season's hero card, or `null` when none is on. |
| GET | `/islamic/ayah-of-the-day` | public | `AyahFakeApi.ayahOfTheDay` | — |

### `scan` → `internal/feature/scan` · owner **Arifin** · phase **P2**

Frontend reference: `../waraqah-frontend/lib/features/scan/data/sources/` (`scan_fake_api.dart`), models in `../waraqah-frontend/lib/features/scan/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/scan/lookup` | public | `ScanFakeApi.lookUp` | `?isbn=9789840001491`: the Book with an Edition of that ISBN, or `null` when Waraqah doesn't have it. |

### `alerts` → `internal/feature/alerts` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/alerts/data/sources/` (`alert_fake_api.dart`), models in `../waraqah-frontend/lib/features/alerts/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/alerts` | me | `AlertFakeApi.alerts` | — |
| POST | `/alerts/remove` | me | `AlertFakeApi.remove` | Body: `{id}`. |
| POST | `/alerts/set` | me | `AlertFakeApi.set` | Body: `{kind, bookId, editionId, targetPriceBdt?}`. |

### `cart` → `internal/feature/cart` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/cart/data/sources/` (`cart_fake_api.dart`), models in `../waraqah-frontend/lib/features/cart/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/cart` | me | `CartFakeApi.cart` | — |
| POST | `/cart/add` | me | `CartFakeApi.add` | Body: `{kind, id}`, e.g. `{kind: edition, id: bk-atomic-pb-en}`. |
| POST | `/cart/remove` | me | `CartFakeApi.remove` | Body: `{lineId}`. |
| POST | `/cart/update` | me | `CartFakeApi.update` | Body: `{lineId, quantity}`. |

### `checkout` → `internal/feature/checkout` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/checkout/data/sources/` (`checkout_fake_api.dart`, `coupon_admin_fake_api.dart`), models in `../waraqah-frontend/lib/features/checkout/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/admin/coupons` | staff:orders | `CouponAdminFakeApi.coupons` | Every coupon, newest first. |
| POST | `/admin/coupons/create` | staff:orders | `CouponAdminFakeApi.create` | Body: a coupon. Answers the updated list, or `null` if the code is already taken. |
| GET | `/coupons/check` | me | `CheckoutFakeApi.coupon` | `?code=EID100`; answers the coupon or `null`. |
| POST | `/orders/place` | me | `CheckoutFakeApi.placeOrder` | Body: `{addressId, payment, couponCode?, usePoints, useWallet, gift?}`. Works out the totals the same way the app does, spends the wallet, saves the order, empties the cart and answers a receipt; `null` when the cart is empty, the address unknown or a gift has no name. |

### `deals` → `internal/feature/deals` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/deals/data/sources/` (`deals_fake_api.dart`), models in `../waraqah-frontend/lib/features/deals/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/deals` | public | `DealsFakeApi.deals` | The flash sale, bundles and pre-orders running now. |

### `donate` → `internal/feature/donate` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/donate/data/sources/` (`donate_admin_fake_api.dart`, `donate_fake_api.dart`), models in `../waraqah-frontend/lib/features/donate/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| POST | `/admin/donate/places/remove` | staff:orders | `DonateAdminFakeApi.remove` | Body `{id}`. Answers every place left, or `null` for an unknown id. |
| POST | `/admin/donate/places/save` | staff:orders | `DonateAdminFakeApi.save` | Body `{id?, name, kind, district, area, story, needs: [{bookId, wanted}]}`. Answers every place, or `null` when it breaks the rules. |
| POST | `/donate/give` | me | `DonateFakeApi.give` | Body: `{recipientId, bookId, quantity, payment, note}`. Saves an order delivered free to the recipient and answers `{orderNumber, totalBdt}`; `null` for cash on delivery or more copies than they still need. |
| GET | `/donate/recipient` | public | `DonateFakeApi.recipient` | `?id=rc-aloghar`; answers the recipient or `null`. |
| GET | `/donate/recipients` | public | `DonateFakeApi.recipients` | — |

### `loyalty` → `internal/feature/loyalty` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/loyalty/data/sources/` (`points_fake_api.dart`), models in `../waraqah-frontend/lib/features/loyalty/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/points` | me | `PointsFakeApi.points` | The balance and its history, newest first. |

### `orders` → `internal/feature/orders` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/orders/data/sources/` (`order_admin_fake_api.dart`, `order_fake_api.dart`), models in `../waraqah-frontend/lib/features/orders/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/admin/orders` | staff:orders | `OrderAdminFakeApi.orders` | Every order, newest first. |
| POST | `/admin/orders/advance` | staff:orders | `OrderAdminFakeApi.advance` | Body: `{number, status}`, where status must be the order's next step. Answers the order, or `null` if that step isn't next any more. |
| POST | `/admin/orders/return` | staff:orders | `OrderAdminFakeApi.decideReturn` | Body: `{number, approve}`. Answers the order, or `null` if it has no waiting return. Approving refunds the books to the reader's wallet. |
| GET | `/orders` | me | `OrderFakeApi.orders` | The reader's orders, newest first. |
| POST | `/orders/cancel` | me | `OrderFakeApi.cancel` | Body: `{number}`. Answers the cancelled order, or `null` if it can't be. |
| GET | `/orders/details` | me | `OrderFakeApi.details` | `?number=WQ-100231`; answers the order or `null`. |
| POST | `/orders/reorder` | me | `OrderFakeApi.reorder` | Body: `{number}`. Puts the order's Editions back in the cart and answers `{added, skipped}` (books), or `null` for an unknown order. |
| POST | `/orders/return` | me | `OrderFakeApi.requestReturn` | Body: `{number, reason, note, photos}`, photos as base64 images. Answers the order, or `null` if a return can't be asked for. |

### `wallet` → `internal/feature/wallet` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/wallet/data/sources/` (`wallet_fake_api.dart`), models in `../waraqah-frontend/lib/features/wallet/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/wallet` | me | `WalletFakeApi.wallet` | The balance and its history, newest first. |

### `wishlist` → `internal/feature/wishlist` · owner **Farhan** · phase **P3**

Frontend reference: `../waraqah-frontend/lib/features/wishlist/data/sources/` (`wishlist_fake_api.dart`), models in `../waraqah-frontend/lib/features/wishlist/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/wishlist` | me | `WishlistFakeApi.wishlist` | — |
| POST | `/wishlist/remove` | me | `WishlistFakeApi.remove` | Body: `{bookId}`. |
| POST | `/wishlist/save` | me | `WishlistFakeApi.save` | Body: `{bookId}`. Saving a book twice keeps one copy, moved to the top. |
| POST | `/wishlist/share` | me | `WishlistFakeApi.share` | Body: `{ownerName}`, the name friends see (there are no auth tokens yet to tell the server who is asking). Answers `{id, ownerName, books}`. |
| GET | `/wishlist/shared` | public | `WishlistFakeApi.shared` | `?id=wl-nabila`; answers `{id, ownerName, books}` or `null`. |

### `book_request` → `internal/feature/bookrequest` · owner **Arifin** · phase **P4**

Frontend reference: `../waraqah-frontend/lib/features/book_request/data/sources/` (`book_request_fake_api.dart`), models in `../waraqah-frontend/lib/features/book_request/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| POST | `/requests` | me | `BookRequestFakeApi.create` | Body `{title, author?, bookId?, maxPriceBdt?, note?}`. |
| POST | `/requests/close` | me | `BookRequestFakeApi.close` | Body `{id}`: answers the reader's requests. |
| GET | `/requests/demand` | staff | `BookRequestFakeApi.demand` | Open requests per title, for admins. |
| GET | `/requests/mine` | me | `BookRequestFakeApi.mine` | — |
| GET | `/requests/wanted` | me | `BookRequestFakeApi.wanted` | Other readers' requests for books the reader is selling. |

### `handled_sale` → `internal/feature/handledsale` · owner **Arifin** · phase **P4**

Frontend reference: `../waraqah-frontend/lib/features/handled_sale/data/sources/` (`handled_sale_fake_api.dart`), models in `../waraqah-frontend/lib/features/handled_sale/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| POST | `/sales/buy` | me | `HandledSaleFakeApi.buy` | Body `{listingId, method}`. |
| GET | `/sales/detail` | me | `HandledSaleFakeApi.sale` | `?id=HS-101`. |
| POST | `/sales/dispute` | me | `HandledSaleFakeApi.dispute` | Body `{id, reason, note?, photos}`, photos as base64 images. |
| GET | `/sales/disputes` | staff:moderate | `HandledSaleFakeApi.disputes` | Open disputes, for moderators. |
| POST | `/sales/disputes/settle` | staff:moderate | `HandledSaleFakeApi.settle` | Body `{id, refund, by}`: answers the open disputes. |
| GET | `/sales/earnings` | me | `HandledSaleFakeApi.earnings` | — |
| GET (SSE) | `/sales/live` | me | `HandledSaleFakeApi.live` | Server-sent events: `data: {seq, saleId}` per change, so a sale's page follows the other side's moves. |
| GET | `/sales/mine` | me | `HandledSaleFakeApi.mine` | — |
| POST | `/sales/payout` | me | `HandledSaleFakeApi.payout` | — |
| POST | `/sales/step` | me | `HandledSaleFakeApi.step` | Body `{id, step}`: send, cancel or confirm. |

### `inbox` → `internal/feature/inbox` · owner **Farhan** · phase **P4**

Frontend reference: `../waraqah-frontend/lib/features/inbox/data/sources/` (`inbox_fake_api.dart`), models in `../waraqah-frontend/lib/features/inbox/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/inbox` | me | `InboxFakeApi.threads` | Threads with messages, newest first, each with its latest message. `?listingId=` keeps those about one listing. |
| POST | `/inbox/listing/release` | me | `InboxFakeApi.release` | Body `{threadId}`: the seller makes the book available again. |
| POST | `/inbox/listing/sold` | me | `InboxFakeApi.sold` | Body `{threadId}`: the seller marks it sold to this buyer. |
| GET (SSE) | `/inbox/live` | me | `InboxFakeApi.live` | A stream of server-sent events, one `data: {seq, threadId, listingId}` per change, kept open while the app listens. |
| POST | `/inbox/offer` | me | `InboxFakeApi.offer` | Body `{listingId, amountBdt, handover}`. |
| POST | `/inbox/offer/decide` | me | `InboxFakeApi.decide` | Body `{threadId, offerId, accept}`. |
| POST | `/inbox/open` | me | `InboxFakeApi.open` | Body `{listingId}`: the buyer's thread, started if needed. |
| POST | `/inbox/rate` | me | `InboxFakeApi.rate` | Body `{threadId, stars, comment?}`: after the sale, once each. |
| POST | `/inbox/read` | me | `InboxFakeApi.read` | Body `{threadId}`. |
| POST | `/inbox/send` | me | `InboxFakeApi.send` | Body `{threadId, text}`. |
| GET | `/inbox/thread` | me | `InboxFakeApi.thread` | `?id=`: the whole thread. |

### `moderation` → `internal/feature/moderation` · owner **Arifin** · phase **P4**

Frontend reference: `../waraqah-frontend/lib/features/moderation/data/sources/` (`moderation_fake_api.dart`), models in `../waraqah-frontend/lib/features/moderation/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/moderation/listings` | staff:moderate | `ModerationFakeApi.queue` | Listings waiting for approval. |
| POST | `/moderation/listings/decide` | staff:moderate | `ModerationFakeApi.decide` | Body `{listingId, decision, reason?, by}`. |
| GET | `/moderation/log` | staff:moderate | `ModerationFakeApi.log` | The audit log, newest first. |
| GET | `/moderation/reports` | staff:moderate | `ModerationFakeApi.reports` | Open reports, one per reported thing. |
| POST | `/moderation/reports/act` | staff:moderate | `ModerationFakeApi.act` | Body `{reportId, action, by}`. |

### `p2p` → `internal/feature/p2p` · owner **Arifin** · phase **P4**

Frontend reference: `../waraqah-frontend/lib/features/p2p/data/sources/` (`p2p_fake_api.dart`), models in `../waraqah-frontend/lib/features/p2p/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/p2p/listing` | public | `P2pFakeApi.listing` | `?id=p2p-1`; answers the listing or `null`. |
| GET | `/p2p/listings` | public | `P2pFakeApi.listings` | Listings on sale or reserved, newest first. `?available=true` keeps only those others can buy now; `?limit=4` caps the list. |
| GET | `/p2p/listings/for-book` | public | `P2pFakeApi.forBook` | `?bookId=bk-cleancode`: copies of a catalog book others are selling. |
| GET | `/p2p/listings/mine` | me | `P2pFakeApi.mine` | The signed-in reader's own listings, in any status. |
| POST | `/p2p/listings/save` | me | `P2pFakeApi.save` | Body: the form's fields, `photos` (slots kept) and `photoData` (`{slot: base64}` picked since), `submit` to send for review. Answers the Listing, or `null` when it breaks `ListingRules`. |
| GET | `/p2p/seller` | public | `P2pFakeApi.seller` | `?id=p-nabila`: a reader's seller page, or `null`. |

### `report` → `internal/feature/report` · owner **Arifin** · phase **P4**

Frontend reference: `../waraqah-frontend/lib/features/report/data/sources/` (`report_fake_api.dart`), models in `../waraqah-frontend/lib/features/report/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/blocks` | me | `ReportFakeApi.blocked` | The signed-in reader's blocked readers, newest first. |
| POST | `/blocks/add` | me | `ReportFakeApi.block` | Body `{readerId}`: answers the blocked list. |
| POST | `/blocks/remove` | me | `ReportFakeApi.unblock` | Body `{readerId}`: answers the blocked list. |
| POST | `/reports` | me | `ReportFakeApi.report` | Body `{kind, targetId, reason, note?}`: answers the report. |

### `sell_back` → `internal/feature/sellback` · owner **Arifin** · phase **P4**

Frontend reference: `../waraqah-frontend/lib/features/sell_back/data/sources/` (`sell_back_fake_api.dart`), models in `../waraqah-frontend/lib/features/sell_back/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| POST | `/sell-back` | me | `SellBackFakeApi.create` | Body `{bookId, condition, flags, pickupAddress}`. |
| GET | `/sell-back/book` | me | `SellBackFakeApi.book` | `?id=bk-zero`. |
| GET | `/sell-back/books` | me | `SellBackFakeApi.books` | `?q=`: catalog Books Waraqah buys back. |
| POST | `/sell-back/grade` | staff:catalog | `SellBackFakeApi.grade` | Body `{id, condition, accept, by}`: answers the queue. |
| GET | `/sell-back/mine` | me | `SellBackFakeApi.mine` | — |
| GET | `/sell-back/queue` | staff:catalog | `SellBackFakeApi.queue` | Picked-up books waiting to be graded, for staff. |

### `bites` → `internal/feature/bites` · owner **Rahinur** · phase **P5**

Frontend reference: `../waraqah-frontend/lib/features/bites/data/sources/` (`bite_fake_api.dart`), models in `../waraqah-frontend/lib/features/bites/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/bites` | public | `BiteFakeApi.feed` | `?feed=forYou\|following&bookId=&authorId=`: newest first, at most 30. |
| POST | `/bites/comments/delete` | me | `BiteFakeApi.deleteComment` | Body `{id}`: own only; answers the detail. |
| POST | `/bites/comments/post` | me | `BiteFakeApi.comment` | Body `{biteId, text, parentId?}`: answers the detail. |
| POST | `/bites/delete` | me | `BiteFakeApi.delete` | Body `{id}`: own only; answers `{id}`. |
| GET | `/bites/detail` | public | `BiteFakeApi.detail` | `?id=`: one Bite with its comments. |
| POST | `/bites/edit` | me | `BiteFakeApi.edit` | Body `{id, text, bookId?, spoiler}`: own only; answers the Bite. |
| POST | `/bites/like` | me | `BiteFakeApi.like` | Body `{id, liked}`: answers the Bite. |
| POST | `/bites/post` | me | `BiteFakeApi.post` | Body `{text, bookId?, spoiler}`: answers the Bite. |

### `readers` → `internal/feature/readers` · owner **Rahinur** · phase **P5**

Frontend reference: `../waraqah-frontend/lib/features/readers/data/sources/` (`reader_fake_api.dart`), models in `../waraqah-frontend/lib/features/readers/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/readers/detail` | public | `ReaderFakeApi.detail` | `?id=`: one Reader's page. |
| POST | `/readers/follow` | me | `ReaderFakeApi.follow` | Body `{id, follow}`: answers the Reader's page. |

### `reviews` → `internal/feature/reviews` · owner **Rahinur** · phase **P5**

Frontend reference: `../waraqah-frontend/lib/features/reviews/data/sources/` (`review_fake_api.dart`), models in `../waraqah-frontend/lib/features/reviews/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/reviews` | public | `ReviewFakeApi.reviews` | `?bookId=`. |
| POST | `/reviews/delete` | me | `ReviewFakeApi.delete` | Body `{bookId}`: deletes "me"'s review. |
| POST | `/reviews/save` | me | `ReviewFakeApi.save` | Body `{bookId, stars, text}`. |

### `shelves` → `internal/feature/shelves` · owner **Arifin** · phase **P5**

Frontend reference: `../waraqah-frontend/lib/features/shelves/data/sources/` (`shelf_fake_api.dart`), models in `../waraqah-frontend/lib/features/shelves/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| POST | `/reading/goal` | me | `ShelfFakeApi.goal` | Body `{goal}` (1–365). Answers the stats, or `null`. |
| GET | `/reading/stats` | me | `ShelfFakeApi.stats` | The reader's year in books: goal, streak, per month, Categories. |
| GET | `/shelves` | me | `ShelfFakeApi.mine` | The signed-in reader's shelves, newest first. |
| POST | `/shelves/move` | me | `ShelfFakeApi.move` | Body `{bookId, shelf}`; no `shelf` takes the Book off. Answers the shelves, or `null` for a Book the catalog doesn't have. |
| POST | `/shelves/progress` | me | `ShelfFakeApi.progress` | Body `{bookId, percent, pagesRead?, totalPages?}`. Answers the shelves, or `null` when it breaks `ProgressRules`. |

### `admin` → `internal/feature/dashboard` · owner **Arifin** · phase **P6**

Frontend reference: `../waraqah-frontend/lib/features/admin/data/sources/` (`dashboard_fake_api.dart`), models in `../waraqah-frontend/lib/features/admin/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| GET | `/admin/dashboard` | staff | `DashboardFakeApi.dashboard` | Today's numbers, what's waiting, top searches and most requested. |

### `ai_assistant` → `internal/feature/assistant` · owner **Arifin** · phase **P6**

Frontend reference: `../waraqah-frontend/lib/features/ai_assistant/data/sources/` (`assistant_fake_api.dart`), models in `../waraqah-frontend/lib/features/ai_assistant/data/models/`.

| Method | Path | Auth | Fake API constant | Notes from the fake API |
|---|---|---|---|---|
| POST | `/assistant/ask` | me | `AssistantFakeApi.ask` | Body `{prompt, history: [text], lang}`. Answers `{id, text, bookIds}`. |
| GET | `/assistant/greeting` | public | `AssistantFakeApi.greeting` | `?lang=bn`: the opening line. |


## Appendix B — Business rules to port

Pure Dart rules classes. Port each to the named Go file as pure functions, and port its frontend tests as table tests. Paths are relative to `../waraqah-frontend/`.

| Dart (frontend) | Go (backend) | Frontend tests to port |
|---|---|---|
| `lib/features/auth/domain/entities/user_role.dart` | `internal/platform/auth/roles.go` | `test/admin_access_test.dart` (roles part) |
| `lib/features/profile/domain/entities/profile_rules.dart` | `internal/feature/profile/rules.go` | `test/profile_rules_test.dart` |
| `lib/features/profile/domain/entities/address_rules.dart` | `internal/feature/profile/address_rules.go` | `test/address_rules_test.dart` |
| `lib/features/catalog/data/sources/phonetic_key.dart` | `internal/feature/catalog/phonetic.go` | `test/phonetic_key_test.dart`, `test/search_bangla_test.dart` |
| `lib/features/catalog/domain/entities/catalog_filters.dart` | `internal/feature/catalog/filters.go` | `test/search_filters_test.dart`, `test/search_sort_test.dart` |
| `lib/features/catalog/domain/entities/delivery_estimate.dart` | `internal/feature/catalog/delivery.go` | `test/delivery_estimate_test.dart` |
| `lib/features/catalog_admin/domain/entities/catalog_admin_rules.dart`, `list_rules.dart` | `internal/feature/catalogadmin/rules.go` | `test/catalog_admin_rules_test.dart`, `test/catalog_admin_record_rules_test.dart` |
| `lib/features/home/data/sources/season_picker.dart` | `internal/feature/home/season.go` | `test/season_picker_test.dart` |
| `lib/features/checkout/domain/entities/checkout_totals.dart`, `coupon.dart` | `internal/feature/checkout/totals.go`, `coupons.go` | `test/checkout_totals_test.dart`, `test/coupon_rules_test.dart`, `test/gift_rules_test.dart` |
| `lib/features/loyalty/domain/entities/loyalty_rules.dart` | `internal/feature/loyalty/rules.go` | `test/loyalty_test.dart` |
| `lib/features/orders/domain/entities/order_refunds.dart` (`OrderRefunds`), `lib/features/wallet/domain/entities/wallet.dart` | `internal/feature/orders/refunds.go`, `internal/feature/wallet/rules.go` | `test/wallet_rules_test.dart` |
| `lib/features/donate/domain/entities/recipient.dart`, `donate_place_draft.dart` (`PlaceRules`) | `internal/feature/donate/rules.go` | `test/donate_rules_test.dart`, `test/donation_places_test.dart` |
| `lib/features/p2p/domain/entities/listing_rules.dart` | `internal/feature/p2p/rules.go` | `test/listing_rules_test.dart` |
| `lib/features/p2p/domain/entities/fair_price.dart` | `internal/feature/p2p/fair_price.go` | `test/fair_price_test.dart` |
| `lib/features/inbox/domain/entities/offer_rules.dart`, `rating_rules.dart` | `internal/feature/inbox/rules.go` | `test/inbox_rules_test.dart`, `test/seller_ratings_test.dart` |
| `lib/features/report/domain/entities/report_rules.dart` | `internal/feature/report/rules.go` | `test/report_rules_test.dart` |
| `lib/features/moderation/domain/entities/moderation_rules.dart` | `internal/feature/moderation/rules.go` | `test/moderation_rules_test.dart` |
| `lib/features/book_request/domain/entities/request_rules.dart` | `internal/feature/bookrequest/rules.go` | `test/book_request_test.dart` (rules part) |
| `lib/features/handled_sale/domain/entities/sale_math.dart` | `internal/feature/handledsale/math.go` | `test/handled_sale_test.dart` |
| `lib/features/sell_back/domain/entities/sell_back_rules.dart` | `internal/feature/sellback/rules.go` | `test/sell_back_test.dart` |
| `lib/features/finished_it/domain/entities/finished_it_offers.dart` | `internal/feature/sellback/finished.go` | `test/finished_it_test.dart` (unit part) |
| `lib/features/bites/domain/entities/bite_rules.dart` | `internal/feature/bites/rules.go` (count graphemes, not bytes: `github.com/rivo/uniseg`) | `test/bite_rules_test.dart` |
| `lib/features/reviews/domain/entities/review_rules.dart` | `internal/feature/reviews/rules.go` | `test/reviews_api_test.dart` (rules part) |
| `lib/features/shelves/domain/entities/progress_rules.dart` | `internal/feature/shelves/rules.go` | `test/reading_stats_test.dart`, `test/shelves_test.dart` |
| `lib/features/ai_assistant/data/sources/assistant_parser.dart`, `assistant_catalog.dart`, `assistant_replies.dart`, `assistant_brain.dart` | `internal/feature/assistant/` | `test/ai_assistant_test.dart`, `test/smarter_ai_test.dart` |

## Appendix C — Where the fake backend logic lives

What each fake store does is what the Go service must do. Paths are `../waraqah-frontend/lib/features/<feature>/data/sources/`. The shared wiring is `../waraqah-frontend/lib/app/fake_stores.dart` (which stores talk to each other) and `../waraqah-frontend/lib/app/fake_api_routes.dart` (which routes exist).

| Feature | Fake backend files | Shares state with |
|---|---|---|
| auth | `auth_fake_api.dart`, `auth_fixtures.dart` | — |
| profile | `profile_fake_api.dart`, `profile_fake_store.dart`, `address_fake_store.dart`, `geo/*.dart` | checkout (addresses), notifications (muted groups) |
| notifications | `notification_fake_api.dart`, `notification_fake_store.dart`, `notification_seed.dart`, `notification_sends.dart`, `notification_sale_sends.dart` | everyone who sends |
| catalog | `book_fake_api.dart`, `book_suggest_fake_api.dart`, `book_questions_fake_api.dart`, `collection_fake_api.dart`, `booklist_fake_api.dart`, `*_fixtures.dart`, `seed/`, `phonetic_key.dart` | catalog admin edits the fixtures in place |
| catalog_admin | `catalog_admin_fake_api.dart`, `catalog_admin_fake_store.dart`, `catalog_admin_fake_*.dart`, `catalog_tools_fake_api.dart`, `catalog_import_fake.dart`, `isbn_lookup_fixtures.dart` | alerts (sweep on change) |
| home | `home_fake_api.dart`, `ayah_fake_api.dart`, `season_picker.dart`, `*_fixtures.dart` | catalog admin (forced Season, banners) |
| cart | `cart_fake_api.dart`, `cart_fake_store.dart` | deals, checkout |
| deals | `deals_fake_api.dart`, `deals_fake_store.dart` | cart |
| wishlist | `wishlist_fake_api.dart` | — |
| alerts | `alert_fake_api.dart`, `alert_fake_store.dart` | notifications, catalog admin |
| checkout | `checkout_fake_api.dart`, `coupon_fake_store.dart`, `coupon_admin_fake_api.dart`, `checkout_fixtures.dart` | cart, orders, points, wallet, addresses |
| orders | `order_fake_api.dart`, `order_admin_fake_api.dart`, `order_fake_store.dart`, `order_fixtures.dart` | wallet, points, cart, notifications, reviews, shelves |
| wallet | `wallet_fake_api.dart`, `wallet_fake_store.dart` | orders, handled sales, Sell Back |
| loyalty | `points_fake_api.dart`, `points_fake_store.dart` | checkout |
| donate | `donate_fake_api.dart`, `donate_admin_fake_api.dart`, `donate_places_store.dart`, `donate_fixtures.dart`, `donation_order.dart` | orders |
| p2p | `p2p_fake_api.dart`, `p2p_fake_store.dart`, `p2p_listing_writer.dart`, `p2p_catalog_link.dart`, `p2p_people.dart`, `p2p_ratings.dart`, `p2p_*seed.dart` | inbox, moderation, reports, handled sales, requests |
| inbox | `inbox_fake_api.dart`, `inbox_fake_store.dart`, `inbox_fake_actions.dart`, `inbox_fake_selling.dart`, `inbox_fake_rating.dart`, `inbox_fake_replies.dart`, `inbox_fake_json.dart`, `inbox_fake_seed.dart` | p2p, reports (blocks) |
| report | `report_fake_api.dart`, `report_fake_store.dart` | p2p, bites, follows, inbox |
| moderation | `moderation_fake_api.dart`, `moderation_fake_store.dart`, `moderation_fake_reports.dart`, `moderation_subjects.dart`, `moderation_seed.dart` | p2p, reports, inbox, bites, reviews, notifications |
| book_request | `book_request_fake_api.dart`, `book_request_fake_store.dart`, `book_request_demand.dart`, `book_request_seed.dart` | p2p, notifications |
| handled_sale | `handled_sale_fake_api.dart`, `handled_sale_fake_store.dart`, `handled_sale_fake_steps.dart`, `handled_sale_fake_money.dart`, `fake_sale.dart`, `sale_changes.dart`, `handled_sale_seed.dart` | p2p, wallet, moderation, notifications |
| sell_back | `sell_back_fake_api.dart`, `sell_back_fake_store.dart`, `sell_back_seed.dart` | wallet, notifications, Certified Used stock |
| bites | `bite_fake_api.dart`, `bite_fake_store.dart`, `bite_fake_json.dart`, `bite_records.dart`, `bite_fixtures.dart` | reports, moderation, follows, notifications |
| reviews | `review_fake_api.dart`, `review_fake_store.dart`, `review_seed.dart` | orders (Verified Purchase), moderation |
| readers | `reader_fake_api.dart`, `follow_fake_store.dart` | profile, bites, p2p, notifications |
| shelves | `shelf_fake_api.dart`, `shelf_fake_store.dart`, `shelf_seed.dart`, `reading_log.dart` | orders |
| ai_assistant | `assistant_fake_api.dart`, `assistant_brain.dart`, `assistant_parser.dart`, `assistant_catalog.dart`, `assistant_replies.dart` | catalog |
| admin (dashboard) | `dashboard_fake_api.dart`, `dashboard_fake_store.dart`, `search_log.dart` | orders, p2p, moderation, handled sales, requests, catalog search |
| scan | `scan_fake_api.dart` | catalog |
