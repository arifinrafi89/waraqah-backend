# Waraqah Backend — Rules for Coding Agents

Read this first, then **`BACKEND_PLAN.md`** (the full plan). If they disagree, the plan wins; fix this file in the same PR.

## What this repo is

The Go + PostgreSQL backend for the Waraqah Flutter app. The app is finished and runs on a fake API inside it. **This backend replaces that fake API without the app noticing** (plan §1, §4).

- Frontend repo: <https://github.com/arifinrafi89/waraqah-frontend>, checked out read-only next to this repo at `../waraqah-frontend/` (`FRONTEND_DIR`). Never commit to it from here.
- The contract is the fake API: `../waraqah-frontend/lib/features/<feature>/data/sources/*_fake_api.dart`. Every endpoint is listed in plan **Appendix A**.

## Golden rules

1. **The fake API is the spec.** Same method, same path under `/v1`, same query and body keys, same JSON field names (camelCase, Dart model names), same enum strings, same `null` meaning. When in doubt, open the fake API file and its model. (Plan §4.)
2. **Refusals and lookup misses answer `200` with body `null`** and an `X-Waraqah-Error: <code>` header. `401`/`403` only for missing or wrong tokens on protected endpoints; `404` only for unknown paths. (Plan §4.2.)
3. **Per-viewer fields come from the token** (`isMine`, `isMyDeal`, `role`, `unread`, …), never from the request.
4. **Port, don't reinvent.** Business rules are already written in Dart (plan Appendix B); port them with their tests. The fake stores (Appendix C) show the logic each endpoint needs.
5. **Layers:** handler (HTTP) → service (rules, transactions, side effects) → store (SQL via sqlc). Features talk to each other only through interfaces wired in `internal/app/deps.go`. `internal/platform` never imports a feature. (Plan §6.)
6. **Names match across repos:** package = frontend feature folder without underscores; handler method = fake API constant name (`P2pFakeApi.save` → `Save`); `routes.go` comments each route with its fake API constant.
7. **Small files:** aim for ≤ 200 lines, hard limit 300. Split handlers by area.
8. **SQL only in `db/queries/*.sql`** (sqlc). Schema changes only as new goose migrations; never edit a merged one.
9. **Config only from env.** New variable → `config.go` and `.env.example` in the same PR. No secrets, tokens, OTPs or base64 images in code, tests or logs.
10. **Every refusal has a code** in `internal/platform/httpx/errors.go` and `docs/error-codes.md`.
11. **No new dependency** without a note in `docs/decisions/` and the PR description.

## Workflow for one ticket

1. Read the ticket (plan §21) and the endpoints' rows in Appendix A.
2. Open the frontend files the ticket names (in `../waraqah-frontend/`): fake API, fake store, models, rules class and its test.
3. Write the migration (if any) → SQL queries → `make sqlc` → store → rules (+ ported tests) → service → handlers → routes line.
4. `make contract-export` if goldens are missing, then make the contract tests for these endpoints pass.
5. `make check` must pass. Then walk the screens in the app against the local backend (`flutter run --dart-define=API_BASE_URL=http://localhost:8080/v1`).
6. Update the feature's `README.md`, and this plan if anything differs.

## Commands

`make dev` · `make test` · `make check` · `make migrate` · `make sqlc` · `make seed` · `make frontend-sync` · `make contract-export` · `make smoke` (plan §22.4).

## Branches, commits, PRs

One branch per ticket (`feature/<kebab-name>`), small PRs, "Create a merge commit", nobody merges their own PR. Commit messages: a clear title, then what and why, with `Committed by:` and `Feature:` lines. **No AI attribution lines** in commits or PRs.

## Ownership

Same as the frontend: Rahinur (accounts, notifications, catalog, home, Bites, reviews, readers), Farhan (cart, checkout, orders, wallet, points, deals, wishlist, alerts, donate, inbox), Arifin (platform foundations, listings, moderation, reports, scan, requests, handled sales, Sell Back, shelves, AI, dashboard, donation places admin). Appendix A names the owner of every endpoint.
