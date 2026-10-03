# Error codes

Every refusal answers `200` with body `null` and an `X-Waraqah-Error: <code>` header (BACKEND_PLAN.md §4.2, §16).
Each code lives in `internal/platform/httpx/errors.go`. Add a row here in the same PR that adds the code.

| Code | Meaning | Dart rule mirrored | Endpoints |
|---|---|---|---|
