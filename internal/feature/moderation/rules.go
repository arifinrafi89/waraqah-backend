// Package moderation is the Moderation Center: the listing queue, reports, strikes and bans (one
// system for every kind of content), and the audit log.
package moderation

import (
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// Port of ModerationRules (lib/features/moderation/domain/entities/moderation_rules.dart).
const (
	// MaxStrikes: the third strike bans the account.
	MaxStrikes = 3
	MaxReason  = 300
)

// Decisions on a listing (ListingDecision), actions on a report (ReportAction) and the audit
// actions (AuditAction).
const (
	Approve        = "approve"
	RequestChanges = "requestChanges"
	Reject         = "reject"

	ActRemove  = "remove"
	ActDismiss = "dismiss"
	ActWarn    = "warn"
	ActBan     = "ban"

	AuditApproved         = "approved"
	AuditChangesRequested = "changesRequested"
	AuditRejected         = "rejected"
	AuditRemoved          = "removed"
	AuditDismissed        = "dismissed"
	AuditWarned           = "warned"
	AuditBanned           = "banned"
)

// Problem is what is wrong with the reason of a decision.
type Problem string

// The problems CheckDecision finds.
const (
	ProblemNone           Problem = ""
	ProblemReasonRequired Problem = "reasonRequired"
	ProblemReasonTooLong  Problem = "reasonTooLong"
)

// CheckDecision: asking for changes and rejecting tell the seller why, in 300 characters or fewer.
func CheckDecision(decision, reason string) Problem {
	text := strings.TrimSpace(reason)
	if textutil.Len(text) > MaxReason {
		return ProblemReasonTooLong
	}
	if decision != Approve && text == "" {
		return ProblemReasonRequired
	}
	return ProblemNone
}

// AfterWarning is the strikes after a warning, and whether that bans the account.
func AfterWarning(strikes int) (next int, banned bool) {
	next = strikes + 1
	return next, next >= MaxStrikes
}
