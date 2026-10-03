package checkout_test

import (
	"context"
	"errors"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/checkout"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func place(e *testenv.Env, body map[string]any) testenv.Result {
	if body["addressId"] == nil {
		body["addressId"] = "addr-home"
	}
	if body["payment"] == nil {
		body["payment"] = "bkash"
	}
	return e.Call("reader", "POST", "/orders/place", body)
}

func balance(t *testing.T, e *testenv.Env, path, key string) float64 {
	t.Helper()
	return e.Call("reader", "GET", path, nil).Obj(t)[key].(float64)
}

func addBook(e *testenv.Env) {
	e.Call("reader", "POST", "/cart/add", map[string]any{"kind": "edition", "id": "bk-atomic-pb-en"})
}

func TestPlaceOrderEndToEnd(t *testing.T) {
	e := testenv.New(t)
	addBook(e)
	pointsBefore := balance(t, e, "/points", "balance")
	r := place(e, map[string]any{"couponCode": "welcome10"})
	if r.Status != 200 || r.Refusal != "" {
		t.Fatalf("place: %d %q %s", r.Status, r.Refusal, r.Raw)
	}
	rec := r.Obj(t)
	number := rec["number"].(string)
	if rec["itemCount"] != 1.0 || rec["insideDhaka"] != true || rec["totalBdt"] == nil {
		t.Errorf("receipt: %v", rec)
	}
	if cart := e.Call("reader", "GET", "/cart", nil).Obj(t)["lines"].([]any); len(cart) != 0 {
		t.Error("the cart must be empty after an order")
	}
	if got := e.Call("reader", "GET", "/orders/details?number="+number, nil); got.Status != 200 || got.Body == nil {
		t.Errorf("details: %s", got.Raw)
	}
	if after := balance(t, e, "/points", "balance"); after != pointsBefore+rec["pointsEarned"].(float64) {
		t.Errorf("points %v -> %v, earned %v", pointsBefore, after, rec["pointsEarned"])
	}
	// the admin sees it, and another reader does not
	if got := e.Call("support", "GET", "/admin/orders", nil).List(t); len(got) < 3 {
		t.Errorf("admin orders: %d", len(got))
	}
	if r := e.Call("moderator", "GET", "/orders/details?number="+number, nil); r.Body != nil {
		t.Error("another reader must not see the order")
	}
}

func TestPlaceRefusals(t *testing.T) {
	e := testenv.New(t)
	if r := place(e, map[string]any{}); r.Body != nil || r.Refusal != "cart_empty" {
		t.Errorf("empty cart: %q %s", r.Refusal, r.Raw)
	}
	addBook(e)
	for name, c := range map[string]struct {
		body map[string]any
		code string
	}{
		"address":  {map[string]any{"addressId": "nope"}, "address_unknown"},
		"payment":  {map[string]any{"payment": "cheque"}, "payment_invalid"},
		"giftname": {map[string]any{"gift": map[string]any{"recipientName": " ", "message": "hi", "wrapped": false}}, "gift_name_missing"},
	} {
		if r := place(e, c.body); r.Body != nil || r.Refusal != c.code {
			t.Errorf("%s: %q %s", name, r.Refusal, r.Raw)
		}
	}
	// nothing was written by the refusals
	if l := e.Call("reader", "GET", "/orders", nil).List(t); len(l) != 2 {
		t.Errorf("orders after refusals: %d", len(l))
	}
	if cart := e.Call("reader", "GET", "/cart", nil).Obj(t)["lines"].([]any); len(cart) != 1 {
		t.Error("a refused order keeps the cart")
	}
}

func TestCouponCheckAndAdmin(t *testing.T) {
	e := testenv.New(t)
	if r := e.Call("reader", "GET", "/coupons/check?code=eid100", nil); r.Obj(t)["code"] != "EID100" {
		t.Errorf("coupon: %s", r.Raw)
	}
	if r := e.Call("reader", "GET", "/coupons/check?code=NOPE", nil); r.Status != 200 || r.Body != nil {
		t.Errorf("unknown coupon: %d %s", r.Status, r.Raw)
	}
	if r := e.Call("reader", "GET", "/admin/coupons", nil); r.Status != 403 {
		t.Errorf("reader on admin coupons: %d", r.Status)
	}
	create := func(c map[string]any) testenv.Result { return e.Call("support", "POST", "/admin/coupons/create", c) }
	if got := create(map[string]any{"code": "spring20", "kind": "percentOff", "value": 20, "minOrderBdt": 0}).List(t); got[0]["code"] != "SPRING20" {
		t.Errorf("newest first: %v", got)
	}
	if r := create(map[string]any{"code": "SPRING20", "kind": "percentOff", "value": 20, "minOrderBdt": 0}); r.Refusal != "coupon_code_taken" {
		t.Errorf("taken: %q", r.Refusal)
	}
	for _, bad := range []map[string]any{
		{"code": "ab", "kind": "amountOff", "value": 10, "minOrderBdt": 0},
		{"code": "HIGH", "kind": "percentOff", "value": 95, "minOrderBdt": 0},
		{"code": "NEG", "kind": "amountOff", "value": 10, "minOrderBdt": -1},
		{"code": "PAST", "kind": "amountOff", "value": 10, "minOrderBdt": 0, "expiresAt": "2001-01-01T00:00:00Z"},
	} {
		if r := create(bad); r.Refusal != "coupon_invalid" {
			t.Errorf("%v: %q", bad["code"], r.Refusal)
		}
	}
}

// failingStock breaks the last step of an order, after the ledgers and the order were written.
type failingStock struct{ checkout.Inventory }

func (failingStock) Take(context.Context, *sqlc.Queries, string, int, time.Time) error {
	return errors.New("forced failure")
}

func TestPlaceIsAllOrNothing(t *testing.T) {
	e := testenv.New(t)
	e.Deps.Checkout.Stock = failingStock{e.Deps.Checkout.Stock}
	addBook(e)
	points, wallet := balance(t, e, "/points", "balance"), balance(t, e, "/wallet", "balanceBdt")
	if r := place(e, map[string]any{"usePoints": true, "useWallet": true}); r.Status != 500 {
		t.Fatalf("forced failure: %d %s", r.Status, r.Raw)
	}
	if got := balance(t, e, "/points", "balance"); got != points {
		t.Errorf("points changed: %v -> %v", points, got)
	}
	if got := balance(t, e, "/wallet", "balanceBdt"); got != wallet {
		t.Errorf("wallet changed: %v -> %v", wallet, got)
	}
	if l := e.Call("reader", "GET", "/orders", nil).List(t); len(l) != 2 {
		t.Errorf("an order was left behind: %d", len(l))
	}
	if cart := e.Call("reader", "GET", "/cart", nil).Obj(t)["lines"].([]any); len(cart) != 1 {
		t.Error("the cart must survive a failed order")
	}
}
