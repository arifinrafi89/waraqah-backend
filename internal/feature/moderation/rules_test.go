package moderation

import (
	"strings"
	"testing"
)

// Ported from test/moderation_rules_test.dart.
func TestDecisionsNeedAShortReason(t *testing.T) {
	if CheckDecision(Approve, "") != ProblemNone {
		t.Error("approving needs no reason")
	}
	if CheckDecision(Reject, " ") != ProblemReasonRequired || CheckDecision(RequestChanges, "") != ProblemReasonRequired {
		t.Error("rejecting and asking for changes need a reason")
	}
	if CheckDecision(Reject, strings.Repeat("x", 301)) != ProblemReasonTooLong {
		t.Error("301 characters is too long")
	}
	if CheckDecision(Reject, "Photocopy") != ProblemNone {
		t.Error("a short reason is fine")
	}
}

func TestTheThirdWarningBans(t *testing.T) {
	if n, banned := AfterWarning(0); n != 1 || banned {
		t.Errorf("first: %d %v", n, banned)
	}
	if n, banned := AfterWarning(2); n != 3 || !banned {
		t.Errorf("third: %d %v", n, banned)
	}
}
