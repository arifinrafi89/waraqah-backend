package bookrequest_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func TestSellersWhoHaveItAreToldAndDemandAddsUp(t *testing.T) {
	e := testenv.New(t)
	create := func(body map[string]any) testenv.Result { return e.Call("reader", "POST", "/requests", body) }
	sent := create(map[string]any{"title": "Clean Code"}).Obj(t)
	// Tanvir sells a live copy of Clean Code
	if sent["matchCount"] != 1.0 || sent["notifiedSellers"] != 1.0 || sent["isOpen"] != true || sent["requesterId"] != nil {
		t.Fatalf("request: %v", sent)
	}
	if mine := e.Call("reader", "GET", "/requests/mine", nil).List(t); len(mine) != 1 || mine[0]["id"] != sent["id"] {
		t.Errorf("mine: %v", mine)
	}
	for name, body := range map[string]map[string]any{
		"blank":      {"title": " "},
		"zero price": {"title": "SICP", "maxPriceBdt": 0},
	} {
		if r := create(body); r.Body != nil || r.Refusal != "request_invalid" {
			t.Errorf("%s: %q", name, r.Refusal)
		}
	}
	closed := e.Call("reader", "POST", "/requests/close", map[string]any{"id": sent["id"]}).List(t)
	if len(closed) != 1 || closed[0]["isOpen"] != false {
		t.Errorf("close: %v", closed)
	}
	if r := e.Call("reader", "POST", "/requests/close", map[string]any{"id": "rq-s1"}); r.Body != nil || r.Refusal != "request_unknown" {
		t.Errorf("another reader's request: %q", r.Refusal)
	}
	// Rafi is looking for the reader's own Atomic Habits
	wanted := e.Call("reader", "GET", "/requests/wanted", nil).List(t)
	if len(wanted) != 1 || wanted[0]["readerName"] != "Rafi" || wanted[0]["listingId"] != "p2p-7" {
		t.Errorf("wanted: %v", wanted)
	}
	demand := e.Call("support", "GET", "/requests/demand", nil).List(t)
	if len(demand) == 0 || demand[0]["title"] != "Calculus: Early Transcendentals" || demand[0]["requests"] != 2.0 {
		t.Errorf("demand: %v", demand)
	}
	if r := e.Call("reader", "GET", "/requests/demand", nil); r.Status != 403 {
		t.Errorf("readers do not see demand: %d", r.Status)
	}
}

func TestSellersGetANotification(t *testing.T) {
	e := testenv.New(t)
	// the demo reader sells Atomic Habits, so a request from another reader tells them
	e.Call("moderator", "POST", "/requests", map[string]any{"title": "Atomic Habits", "bookId": "bk-atomic"})
	found := false
	for _, n := range e.Call("reader", "GET", "/notifications", nil).List(t) {
		if n["kind"] == "bookWanted" {
			found = true
		}
	}
	if !found {
		t.Error("the seller must hear about the request")
	}
}
