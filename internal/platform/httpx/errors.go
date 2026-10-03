package httpx

// Refusal codes: sent in the X-Waraqah-Error header of a `200 null` answer.
// Features add their codes here and in docs/error-codes.md (AGENTS.md rule 10).
const (
	// ErrRateLimited is sent as a 429 error body code, not as a refusal header.
	ErrRateLimited = "rate_limited"
)

// Error body codes for non-200 answers (BACKEND_PLAN.md §4.2).
const (
	CodeUnauthorized = "unauthorized"
	CodeForbidden    = "forbidden"
	CodeNotFound     = "not_found"
	CodeBadRequest   = "bad_request"
	CodeTooLarge     = "too_large"
	CodeInternal     = "internal"
)
