package handledsale_test

import (
	"context"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

const photo = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func listingStatus(e *testenv.Env, id string) any {
	return e.Call("", "GET", "/p2p/listing?id="+id, nil).Obj(e.T)["status"]
}

func balance(e *testenv.Env, as string) float64 {
	return e.Call(as, "GET", "/wallet", nil).Obj(e.T)["balanceBdt"].(float64)
}

func TestSeededSalesFromTheTokenSide(t *testing.T) {
	e := testenv.New(t)
	if got := e.Call("reader", "GET", "/sales/mine", nil).IDs(t); len(got) != 3 || got[0] != "HS-103" || got[2] != "HS-102" {
		t.Errorf("mine, newest first: %v", got)
	}
	s := e.Call("reader", "GET", "/sales/detail?id=HS-101", nil).Obj(t)
	if s["role"] != "buyer" || s["otherName"] != "Arif" || s["title"] != "Thinking, Fast and Slow" || s["feeBdt"] != 19.0 {
		t.Errorf("detail: %v", s)
	}
	if e.Call("reader", "GET", "/sales/detail?id=HS-104", nil).Body != nil {
		t.Error("a sale of other readers is not shown")
	}
	earn := e.Call("reader", "GET", "/sales/earnings", nil).Obj(t)
	if earn["heldBdt"] != 209.0 || earn["earnedBdt"] != 285.0 || earn["paidOutBdt"] != 285.0 {
		t.Errorf("earnings: %v", earn)
	}
	// guests get the empty values; changes need a token
	if got := e.Call("", "GET", "/sales/mine", nil).List(t); len(got) != 0 {
		t.Error("guest mine")
	}
	if got := e.Call("", "GET", "/sales/earnings", nil).Obj(t); got["heldBdt"] != 0.0 {
		t.Error("guest earnings")
	}
	if got := e.Call("", "POST", "/sales/buy", map[string]any{"listingId": "p2p-1", "method": "bkash"}); got.Status != 401 {
		t.Errorf("guest buy: %d", got.Status)
	}
	if got := e.Call("reader", "GET", "/sales/disputes", nil); got.Status != 403 {
		t.Errorf("reader on disputes: %d", got.Status)
	}
}

func TestBuyRules(t *testing.T) {
	e := testenv.New(t)
	for _, c := range []struct{ listing, method, code string }{
		{"p2p-1", "cashOnDelivery", "sale_prepaid_only"},
		{"p2p-7", "bkash", "listing_own"},
		{"p2p-2", "bkash", "listing_unavailable"},
		{"p2p-nope", "bkash", "listing_unavailable"},
	} {
		if r := e.Call("reader", "POST", "/sales/buy", map[string]any{"listingId": c.listing, "method": c.method}); r.Body != nil || r.Refusal != c.code {
			t.Errorf("%s by %s: %s %q", c.listing, c.method, r.Raw, r.Refusal)
		}
	}
}

func TestBuySendConfirmAndPayout(t *testing.T) {
	e := testenv.New(t)
	// A demo seller sends at once in tests (DEMO_BOT_DELAY is 0); the answer is the sale as paid.
	sale := e.Call("reader", "POST", "/sales/buy", map[string]any{"listingId": "p2p-1", "method": "bkash"}).Obj(t)
	id := sale["id"].(string)
	if sale["status"] != "paid" || sale["role"] != "buyer" || sale["deliveryBdt"] != 80.0 || listingStatus(e, "p2p-1") != "reserved" {
		t.Fatalf("buy: %v", sale)
	}
	if got := e.Call("reader", "GET", "/sales/detail?id="+id, nil).Obj(t); got["status"] != "sent" {
		t.Fatalf("the demo seller sent it: %v", got["status"])
	}
	if r := e.Call("reader", "POST", "/sales/step", map[string]any{"id": id, "step": "cancel"}); r.Refusal != "sale_step_refused" {
		t.Errorf("too late to cancel: %q", r.Refusal)
	}
	if got := e.Call("reader", "POST", "/sales/step", map[string]any{"id": id, "step": "confirm"}).Obj(t); got["status"] != "completed" {
		t.Errorf("confirm: %v", got)
	}
	if listingStatus(e, "p2p-1") != "sold" {
		t.Error("confirming sells the book")
	}

	// The reader as a seller: send HS-103, Mahi is not a demo bot buyer, so the reader confirms nothing.
	if r := e.Call("reader", "POST", "/sales/step", map[string]any{"id": "HS-103", "step": "confirm"}); r.Refusal != "sale_step_refused" {
		t.Errorf("a seller cannot confirm: %q", r.Refusal)
	}
	if got := e.Call("reader", "POST", "/sales/step", map[string]any{"id": "HS-103", "step": "send"}).Obj(t); got["status"] != "sent" {
		t.Errorf("send: %v", got)
	}
	if r := e.Call("reader", "POST", "/sales/payout", nil); r.Body != nil || r.Refusal != "payout_nothing" {
		t.Errorf("nothing earned since the last payout: %s", r.Raw)
	}
}

func TestCancelRefundsTheBuyer(t *testing.T) {
	e := testenv.New(t)
	before := balance(e, "moderator")
	sale := e.Call("moderator", "POST", "/sales/buy", map[string]any{"listingId": "p2p-7", "method": "nagad"}).Obj(t)
	id := sale["id"].(string)
	if got := e.Call("reader", "GET", "/sales/detail?id="+id, nil).Obj(t); got["role"] != "seller" || got["status"] != "paid" {
		t.Fatalf("a real seller sends it themselves: %v", got)
	}
	if got := e.Call("moderator", "POST", "/sales/step", map[string]any{"id": id, "step": "cancel"}).Obj(t); got["status"] != "cancelled" {
		t.Fatalf("cancel: %v", got)
	}
	if after := balance(e, "moderator"); after != before+430 {
		t.Errorf("refund of price and delivery: %v -> %v", before, after)
	}
	if listingStatus(e, "p2p-7") != "live" {
		t.Error("cancelling puts the book back on sale")
	}
	if r := e.Call("support", "POST", "/sales/step", map[string]any{"id": id, "step": "cancel"}); r.Refusal != "sale_unknown" {
		t.Errorf("not their sale: %q", r.Refusal)
	}
}

func TestDisputeAndSettle(t *testing.T) {
	e := testenv.New(t)
	for _, body := range []map[string]any{
		{"id": "HS-101", "reason": "lost", "photos": []string{}},
		{"id": "HS-101", "reason": "damaged", "photos": []string{photo, photo, photo, photo}},
	} {
		if r := e.Call("reader", "POST", "/sales/dispute", body); r.Refusal != "dispute_invalid" {
			t.Errorf("%v: %q", body, r.Refusal)
		}
	}
	if r := e.Call("reader", "POST", "/sales/dispute", map[string]any{"id": "HS-101", "reason": "damaged", "photos": []string{"bm90IGFuIGltYWdl"}}); r.Refusal != "photo_invalid" {
		t.Errorf("a photo must be an image: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/sales/dispute", map[string]any{"id": "HS-103", "reason": "damaged"}); r.Refusal != "sale_step_refused" {
		t.Errorf("a seller cannot dispute: %q", r.Refusal)
	}
	got := e.Call("reader", "POST", "/sales/dispute", map[string]any{"id": "HS-101", "reason": "damaged", "note": " Torn ", "photos": []string{photo}}).Obj(t)
	if got["status"] != "disputed" || got["disputeNote"] != "Torn" || len(got["disputePhotos"].([]any)) != 1 {
		t.Fatalf("dispute: %v", got)
	}
	if list := e.Call("moderator", "GET", "/sales/disputes", nil).List(t); len(list) != 2 || list[1]["buyerName"] == "" {
		t.Fatalf("open disputes: %v", list)
	}
	before := balance(e, "reader")
	left := e.Call("moderator", "POST", "/sales/disputes/settle", map[string]any{"id": "HS-101", "refund": true, "by": "someone else"}).List(t)
	if len(left) != 1 || balance(e, "reader") != before+460 || listingStatus(e, "p2p-hs-1") != "live" {
		t.Errorf("refund: %v, wallet %v -> %v", left, before, balance(e, "reader"))
	}
	log := e.Call("moderator", "GET", "/moderation/log", nil).List(t)
	if log[0]["action"] != "refunded" || log[0]["by"] == "someone else" {
		t.Errorf("audit log uses the token: %v", log[0])
	}
	if got := e.Call("moderator", "POST", "/sales/disputes/settle", map[string]any{"id": "HS-104", "refund": false}).List(t); len(got) != 0 {
		t.Errorf("pay the seller: %v", got)
	}
	if listingStatus(e, "p2p-hs-4") != "sold" {
		t.Error("paying the seller sells the book")
	}
	if r := e.Call("moderator", "POST", "/sales/disputes/settle", map[string]any{"id": "HS-104", "refund": true}); r.Refusal != "sale_not_disputed" {
		t.Errorf("settled twice: %q", r.Refusal)
	}
}

func TestDemoSendJobIsIdempotent(t *testing.T) {
	e := testenv.New(t)
	ctx := context.Background()
	for range 2 {
		if err := e.Deps.Sales.SendDemoSales(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// HS-103's seller is the demo reader's real account: the bot never acts for it.
	if got := e.Call("reader", "GET", "/sales/detail?id=HS-103", nil).Obj(t); got["status"] != "paid" {
		t.Errorf("a real seller's sale is left alone: %v", got["status"])
	}
}
