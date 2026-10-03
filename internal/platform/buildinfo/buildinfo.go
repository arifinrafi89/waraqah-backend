// Package buildinfo holds values stamped in with -ldflags at build time.
package buildinfo

// Set with: -ldflags "-X github.com/arifinrafi89/waraqah-backend/internal/platform/buildinfo.Commit=<sha>".
var (
	Commit  = "dev"
	BuiltAt = "unknown"
)

// Contract is the API contract version served (BACKEND_PLAN.md §4.7).
const Contract = "v1"
