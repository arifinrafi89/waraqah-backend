package sellback_test

import (
	"context"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func walletBalance(e *testenv.Env) float64 {
	return e.Call("reader", "GET", "/wallet", nil).Obj(e.T)["balanceBdt"].(float64)
}

func TestBooksAndQuotes(t *testing.T) {
	e := testenv.New(t)
	if got := e.Call("reader", "GET", "/sell-back/books?q=zero", nil).List(t); len(got) != 1 || got[0]["newPriceBdt"] != 430.0 {
		t.Errorf("search: %v", got)
	}
	if got := e.Call("reader", "GET", "/sell-back/books?q=z", nil).List(t); len(got) != 0 {
		t.Error("one letter finds nothing")
	}
	if got := e.Call("reader", "GET", "/sell-back/book?id=bk-nope", nil); got.Body != nil {
		t.Error("unknown book is null")
	}
	if got := e.Call("reader", "GET", "/sell-back/mine", nil).IDs(t); len(got) != 2 || got[0] != "SB-203" {
		t.Errorf("mine, newest first: %v", got)
	}
	if got := e.Call("", "GET", "/sell-back/mine", nil).List(t); len(got) != 0 {
		t.Error("guest")
	}
	if got := e.Call("reader", "GET", "/sell-back/queue", nil); got.Status != 403 {
		t.Errorf("a reader on the queue: %d", got.Status)
	}
}

func TestCreateRefusals(t *testing.T) {
	e := testenv.New(t)
	for _, c := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"bookId": "bk-nope", "condition": "good", "pickupAddress": "Road 7, Dhanmondi"}, "sell_back_book_unknown"},
		{map[string]any{"bookId": "bk-zero", "condition": "good", "pickupAddress": "x"}, "sell_back_invalid"},
		{map[string]any{"bookId": "bk-zero", "condition": "mint", "pickupAddress": "Road 7, Dhanmondi"}, "sell_back_invalid"},
	} {
		if r := e.Call("reader", "POST", "/sell-back", c.body); r.Body != nil || r.Refusal != c.code {
			t.Errorf("%v: %q", c.body, r.Refusal)
		}
	}
}

func TestBookedPickedUpGradedPaidAndOnSale(t *testing.T) {
	e := testenv.New(t)
	sb := e.Call("reader", "POST", "/sell-back", map[string]any{"bookId": "bk-zero", "condition": "good", "flags": 1, "pickupAddress": "Road 7, Dhanmondi"}).Obj(t)
	if sb["status"] != "scheduled" || sb["quoteBdt"] != 90.0 {
		t.Fatalf("create: %v", sb)
	}
	// The courier ran (COURIER_PICKUP_DELAY is 0 in tests); running the job again changes nothing.
	for range 2 {
		if err := e.Deps.SellBack.PickUp(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	queue := e.Call("catalog", "GET", "/sell-back/queue", nil).List(t)
	if len(queue) != 3 || queue[0]["readerName"] != "Nabila" || queue[2]["id"] != sb["id"] {
		t.Fatalf("queue, oldest first: %v", queue)
	}

	before := walletBalance(e)
	left := e.Call("catalog", "POST", "/sell-back/grade", map[string]any{"id": "SB-203", "condition": "good", "accept": true, "by": "x"}).List(t)
	if len(left) != 2 {
		t.Errorf("graded leaves the queue: %d", len(left))
	}
	if after := walletBalance(e); after != before+130 {
		t.Errorf("paid the quote for the graded condition: %v -> %v", before, after)
	}
	if r := e.Call("catalog", "POST", "/sell-back/grade", map[string]any{"id": "SB-203", "condition": "good", "accept": true}); r.Refusal != "sell_back_not_waiting" {
		t.Errorf("graded twice: %q", r.Refusal)
	}
	// Sapiens now has two copies on sale; the book page offers the cheapest.
	used := e.Call("", "GET", "/books/used-options?id=bk-sapiens", nil).Obj(t)["certifiedUsed"].(map[string]any)
	if used["id"] != "cu-sapiens-2" || used["priceBdt"] != 230.0 {
		t.Errorf("cheapest copy: %v", used)
	}
	if left := e.Call("catalog", "POST", "/sell-back/grade", map[string]any{"id": "SB-202", "condition": "good", "accept": false}).List(t); len(left) != 1 {
		t.Errorf("returned: %v", left)
	}
}

func TestCertifiedCopyInCartAndOrder(t *testing.T) {
	e := testenv.New(t)
	cart := e.Call("reader", "POST", "/cart/add", map[string]any{"kind": "certifiedUsed", "id": "cu-atomic-1"}).Obj(t)
	lines := cart["lines"].([]any)
	if len(lines) != 1 || lines[0].(map[string]any)["unitPriceBdt"] != 380.0 {
		t.Fatalf("cart: %v", cart)
	}
	r := e.Call("reader", "POST", "/orders/place", map[string]any{"addressId": "addr-home", "payment": "bkash"})
	if r.Body == nil {
		t.Fatalf("place: %q %s", r.Refusal, r.Raw)
	}
	if got := e.Call("", "GET", "/books/used-options?id=bk-atomic", nil).Obj(t)["certifiedUsed"]; got != nil {
		t.Errorf("the copy was sold: %v", got)
	}
	if got := e.Call("reader", "POST", "/cart/add", map[string]any{"kind": "certifiedUsed", "id": "cu-atomic-1"}); got.Refusal != "cart_item_unknown" {
		t.Errorf("a sold copy cannot be added: %q", got.Refusal)
	}
}
