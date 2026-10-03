package moderation_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/cloudinary"
	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func reportID(t *testing.T, e *testenv.Env, targetID string) string {
	t.Helper()
	for _, r := range e.Call("moderator", "GET", "/moderation/reports", nil).List(t) {
		if r["targetId"] == targetID {
			return r["id"].(string)
		}
	}
	t.Fatalf("no open report about %s", targetID)
	return ""
}

func TestOnlyModeratorsReachTheCenter(t *testing.T) {
	e := testenv.New(t)
	for _, path := range []string{"/moderation/listings", "/moderation/reports", "/moderation/log"} {
		if r := e.Call("reader", "GET", path, nil); r.Status != 403 {
			t.Errorf("reader %s: %d", path, r.Status)
		}
		if r := e.Call("catalog", "GET", path, nil); r.Status != 403 {
			t.Errorf("catalog manager %s: %d", path, r.Status)
		}
		if r := e.Call("moderator", "GET", path, nil); r.Status != 200 {
			t.Errorf("moderator %s: %d", path, r.Status)
		}
	}
}

func TestDecisionsMoveListingsOutOfTheQueueAndIntoTheLog(t *testing.T) {
	e := testenv.New(t)
	decide := func(id, decision, reason string) testenv.Result {
		return e.Call("moderator", "POST", "/moderation/listings/decide", map[string]any{"listingId": id, "decision": decision, "reason": reason, "by": "Someone Else"})
	}
	if q := e.Call("moderator", "GET", "/moderation/listings", nil).List(t); len(q) != 3 {
		t.Fatalf("queue: %d", len(q))
	}
	if r := decide("p2p-review-2", "reject", ""); r.Body != nil || r.Refusal != "decision_reason_invalid" {
		t.Errorf("a rejection needs a reason: %q", r.Refusal)
	}
	if r := decide("p2p-review-2", "maybe", ""); r.Refusal != "decision_invalid" {
		t.Errorf("unknown decision: %q", r.Refusal)
	}
	if r := decide("p2p-1", "approve", ""); r.Refusal != "listing_not_waiting" {
		t.Errorf("a live listing is not waiting: %q", r.Refusal)
	}
	if q := decide("p2p-review-2", "approve", "").List(t); len(q) != 2 {
		t.Errorf("approved: %d left", len(q))
	}
	decide("p2p-review-1", "requestChanges", "Pics")
	mine := e.Call("reader", "GET", "/p2p/listing?id=p2p-review-1", nil).Obj(t)
	if mine["status"] != "changesRequested" || mine["rejectionReason"] != "Pics" {
		t.Errorf("seller sees the answer: %v", mine)
	}
	if live := e.Call("", "GET", "/p2p/listing?id=p2p-review-2", nil).Obj(t); live["status"] != "live" {
		t.Errorf("approved: %v", live["status"])
	}
	log := e.Call("moderator", "GET", "/moderation/log", nil).List(t)
	if len(log) != 2 || log[0]["action"] != "changesRequested" || log[1]["action"] != "approved" {
		t.Fatalf("log: %v", log)
	}
	if log[0]["by"] == "Someone Else" || log[0]["by"] == "" {
		t.Errorf("the log names the staff member of the token, not the body: %v", log[0]["by"])
	}
	// the seller was told
	found := false
	for _, n := range e.Call("reader", "GET", "/notifications", nil).List(t) {
		if n["kind"] == "listingDecided" {
			found = true
		}
	}
	if !found {
		t.Error("the seller must be notified")
	}
}

func TestActingOnReports(t *testing.T) {
	e := testenv.New(t)
	open := e.Call("moderator", "GET", "/moderation/reports", nil).List(t)
	if len(open) != 3 {
		t.Fatalf("one card per thing: %d", len(open))
	}
	var photocopy map[string]any
	for _, r := range open {
		if r["targetId"] == "p2p-6" {
			photocopy = r
		}
	}
	if photocopy["reportCount"] != 2.0 || photocopy["ownerName"] != "Mahi" {
		t.Errorf("grouped report: %v", photocopy)
	}
	act := func(id, action string) testenv.Result {
		return e.Call("moderator", "POST", "/moderation/reports/act", map[string]any{"reportId": id, "action": action, "by": "x"})
	}
	if r := act("rp-nope", "dismiss"); r.Refusal != "report_not_open" {
		t.Errorf("unknown: %q", r.Refusal)
	}
	if r := act(reportID(t, e, "p2p-6"), "explode"); r.Refusal != "action_invalid" {
		t.Errorf("action: %q", r.Refusal)
	}
	// removing the listing closes both reports and takes it down
	left := act(photocopy["id"].(string), "remove").List(t)
	if len(left) != 2 {
		t.Errorf("left: %d", len(left))
	}
	gone := e.Call("", "GET", "/p2p/listing?id=p2p-6", nil).Obj(t)
	if gone["status"] != "rejected" {
		t.Errorf("removed listing: %v", gone["status"])
	}
	// warning the reader of the user report closes it; the Bite report is the one left
	rafi := reportID(t, e, "p-rafi")
	if r := act(rafi, "warn").List(t); len(r) != 1 {
		t.Fatalf("after the warning: %d", len(r))
	}
	if r := e.Call("moderator", "GET", "/moderation/log", nil).List(t); r[0]["action"] != "warned" {
		t.Errorf("log: %v", r[0])
	}
	// Bites register their moderation subject, so the author of the reported Bite (Rafi) is warned.
	if r := act(reportID(t, e, "bt-3"), "warn").List(t); len(r) != 0 {
		t.Errorf("after warning the Bite's author: %d", len(r))
	}
	if r := e.Call("moderator", "GET", "/moderation/log", nil).List(t); r[0]["action"] != "warned" {
		t.Errorf("log: %v", r[0])
	}
}

func TestThirdStrikeAndBanSignTheReaderOut(t *testing.T) {
	e := testenv.New(t)
	act := func(id, action string) testenv.Result {
		return e.Call("moderator", "POST", "/moderation/reports/act", map[string]any{"reportId": id, "action": action})
	}
	for i := 0; i < 3; i++ {
		e.Call("reader", "POST", "/reports", map[string]any{"kind": "user", "targetId": "p-sadia", "reason": "spam", "note": "n"})
		if r := act(reportID(t, e, "p-sadia"), "warn"); r.Status != 200 {
			t.Fatalf("warn %d: %s", i, r.Raw)
		}
	}
	banned, err := e.Deps.Moderation.IsBanned(t.Context(), "p-sadia")
	if err != nil || !banned {
		t.Fatalf("banned after three strikes: %v %v", banned, err)
	}
	log := e.Call("moderator", "GET", "/moderation/log", nil).List(t)
	if log[0]["action"] != "banned" || log[0]["reason"] != "strikes" {
		t.Errorf("log: %v", log[0])
	}
	// an outright ban
	e.Call("reader", "POST", "/reports", map[string]any{"kind": "listing", "targetId": "p2p-3", "reason": "fake"})
	if r := act(reportID(t, e, "p2p-3"), "ban"); r.Status != 200 {
		t.Fatalf("ban: %s", r.Raw)
	}
	if b, _ := e.Deps.Moderation.IsBanned(t.Context(), "p-rakib"); !b {
		t.Error("ban")
	}
}

func TestRemovingAListingDeletesItsPhotos(t *testing.T) {
	e := testenv.New(t)
	saved := e.Call("reader", "POST", "/p2p/listings/save", map[string]any{"title": "Photos", "priceBdt": 100,
		"photoData": map[string]any{"front": tiny, "back": tiny}}).Obj(t)
	id := saved["id"].(string)
	// "u_reader" cannot report their own listing, so a second reader does
	e.Call("moderator", "POST", "/reports", map[string]any{"kind": "listing", "targetId": id, "reason": "spam"})
	if r := e.Call("moderator", "POST", "/moderation/reports/act", map[string]any{"reportId": reportID(t, e, id), "action": "remove"}); r.Status != 200 {
		t.Fatalf("remove: %s", r.Raw)
	}
	fake := e.Deps.Images.(*cloudinary.Fake)
	if len(fake.Destroyed()) != 2 {
		t.Errorf("both photos must be deleted: %v", fake.Destroyed())
	}
}

// A 1x1 PNG.
const tiny = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
