package orders_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func TestSeededOrdersAndLedgers(t *testing.T) {
	e := testenv.New(t)
	l := e.Call("reader", "GET", "/orders", nil).List(t)
	if len(l) != 2 || l[0]["number"] != "WQ-100215" || l[1]["status"] != "delivered" {
		t.Fatalf("orders: %s", e.Call("reader", "GET", "/orders", nil).Raw)
	}
	if b := e.Call("reader", "GET", "/wallet", nil).Obj(t)["balanceBdt"]; b != 180.0 {
		t.Errorf("wallet: %v", b)
	}
	if b := e.Call("reader", "GET", "/points", nil).Obj(t)["balance"]; b != 217.0 {
		t.Errorf("points: %v", b)
	}
}

func TestCancelRules(t *testing.T) {
	e := testenv.New(t)
	if r := e.Call("reader", "POST", "/orders/cancel", map[string]any{"number": "WQ-100215"}); r.Body != nil || r.Refusal != "order_not_cancellable" {
		t.Errorf("shipped order: %q %s", r.Refusal, r.Raw)
	}
	if r := e.Call("reader", "POST", "/orders/cancel", map[string]any{"number": "WQ-9"}); r.Refusal != "order_unknown" {
		t.Errorf("unknown: %q", r.Refusal)
	}
	e.Call("reader", "POST", "/cart/add", map[string]any{"kind": "edition", "id": "bk-atomic-pb-en"})
	n := e.Call("reader", "POST", "/orders/place", map[string]any{"addressId": "addr-home", "payment": "bkash", "usePoints": true}).Obj(t)["number"].(string)
	pts := e.Call("reader", "GET", "/points", nil).Obj(t)["balance"]
	r := e.Call("reader", "POST", "/orders/cancel", map[string]any{"number": n})
	if r.Status != 200 || r.Obj(t)["status"] != "cancelled" {
		t.Fatalf("cancel: %s", r.Raw)
	}
	if got := e.Call("reader", "GET", "/points", nil).Obj(t)["balance"]; got == pts {
		t.Error("cancelling must give back and take back points")
	}
	if w := e.Call("reader", "GET", "/wallet", nil).Obj(t)["balanceBdt"].(float64); w <= 180 {
		t.Errorf("a paid cancelled order is refunded to the wallet: %v", w)
	}
}

func TestReturnFlow(t *testing.T) {
	e := testenv.New(t)
	if r := e.Call("reader", "POST", "/orders/return", map[string]any{"number": "WQ-100201", "reason": "bad"}); r.Refusal != "return_reason_invalid" {
		t.Errorf("reason: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/orders/return", map[string]any{"number": "WQ-100215", "reason": "damaged"}); r.Refusal != "return_not_allowed" {
		t.Errorf("not delivered: %q", r.Refusal)
	}
	r := e.Call("reader", "POST", "/orders/return", map[string]any{"number": "WQ-100201", "reason": "damaged", "note": "torn"})
	if r.Status != 200 || r.Body == nil {
		t.Fatalf("return: %q %s", r.Refusal, r.Raw)
	}
	if again := e.Call("reader", "POST", "/orders/return", map[string]any{"number": "WQ-100201", "reason": "damaged"}); again.Refusal != "return_not_allowed" {
		t.Errorf("twice: %q", again.Refusal)
	}
	if r := e.Call("support", "POST", "/admin/orders/return", map[string]any{"number": "WQ-100201", "approve": true}); r.Status != 200 || r.Body == nil {
		t.Fatalf("approve: %q %s", r.Refusal, r.Raw)
	}
	if w := e.Call("reader", "GET", "/wallet", nil).Obj(t)["balanceBdt"].(float64); w <= 180 {
		t.Errorf("approved return refunds the wallet: %v", w)
	}
	if r := e.Call("support", "POST", "/admin/orders/return", map[string]any{"number": "WQ-100201", "approve": true}); r.Refusal != "return_not_waiting" {
		t.Errorf("decided twice: %q", r.Refusal)
	}
}

func TestAdvanceAndReorder(t *testing.T) {
	e := testenv.New(t)
	if r := e.Call("support", "POST", "/admin/orders/advance", map[string]any{"number": "WQ-100215", "status": "packed"}); r.Refusal != "order_step_unavailable" {
		t.Errorf("wrong step: %q", r.Refusal)
	}
	if r := e.Call("support", "POST", "/admin/orders/advance", map[string]any{"number": "WQ-100215", "status": "delivered"}); r.Status != 200 || r.Obj(t)["status"] != "delivered" {
		t.Fatalf("advance: %s", r.Raw)
	}
	if r := e.Call("reader", "POST", "/admin/orders/advance", map[string]any{"number": "WQ-100215", "status": "delivered"}); r.Status != 403 {
		t.Errorf("reader advancing: %d", r.Status)
	}
	res := e.Call("reader", "POST", "/orders/reorder", map[string]any{"number": "WQ-100201"}).Obj(t)
	if res["added"] != 2.0 {
		t.Errorf("reorder: %v", res)
	}
}
