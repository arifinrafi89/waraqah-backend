package donate_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func need(t *testing.T, e *testenv.Env, place, book string) map[string]any {
	t.Helper()
	r := e.Call("", "GET", "/donate/recipient?id="+place, nil).Obj(t)
	for _, n := range r["needs"].([]any) {
		m := n.(map[string]any)
		if m["book"].(map[string]any)["id"] == book {
			return m
		}
	}
	t.Fatalf("no need for %s", book)
	return nil
}

func TestRecipientsArePublic(t *testing.T) {
	e := testenv.New(t)
	if got := e.Call("", "GET", "/donate/recipients", nil).List(t); len(got) != 3 {
		t.Errorf("places: %d", len(got))
	}
	if r := e.Call("", "GET", "/donate/recipient?id=rc-nope", nil); r.Status != 200 || r.Body != nil {
		t.Errorf("unknown: %d %s", r.Status, r.Raw)
	}
	if n := need(t, e, "rc-aloghar", "bk-hpstone"); n["wanted"] != 10.0 || n["received"] != 4.0 || n["editionId"] == "" {
		t.Errorf("need: %v", n)
	}
}

func TestGiveCreatesADonationOrder(t *testing.T) {
	e := testenv.New(t)
	give := func(q int, pay string) testenv.Result {
		return e.Call("reader", "POST", "/donate/give", map[string]any{"recipientId": "rc-aloghar", "bookId": "bk-hobbit", "quantity": q, "payment": pay, "note": "Enjoy"})
	}
	if r := give(1, "cashOnDelivery"); r.Body != nil || r.Refusal != "donate_cod_not_allowed" {
		t.Errorf("cod: %q", r.Refusal)
	}
	if r := give(6, "bkash"); r.Body != nil || r.Refusal != "donate_too_many" {
		t.Errorf("too many (5 still needed): %q", r.Refusal)
	}
	if r := give(0, "bkash"); r.Refusal != "donate_too_many" {
		t.Errorf("zero: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/donate/give", map[string]any{"recipientId": "rc-nope", "bookId": "bk-hobbit", "quantity": 1, "payment": "bkash"}); r.Refusal != "donate_recipient_unknown" {
		t.Errorf("unknown place: %q", r.Refusal)
	}
	if r := e.Call("", "POST", "/donate/give", map[string]any{}); r.Status != 401 {
		t.Errorf("guest: %d", r.Status)
	}
	d := give(2, "bkash").Obj(t)
	if d["orderNumber"] == nil || d["totalBdt"].(float64) <= 0 {
		t.Fatalf("donation: %v", d)
	}
	if n := need(t, e, "rc-aloghar", "bk-hobbit"); n["received"] != 3.0 {
		t.Errorf("received: %v", n["received"])
	}
	o := e.Call("reader", "GET", "/orders/details?number="+d["orderNumber"].(string), nil).Obj(t)
	if o["isDonation"] != true || o["deliveryFeeBdt"] != 0.0 {
		t.Errorf("order: %v", o)
	}
	if r := give(4, "bkash"); r.Refusal != "donate_too_many" {
		t.Errorf("only 3 still needed: %q", r.Refusal)
	}
}

func TestPlacesAdmin(t *testing.T) {
	e := testenv.New(t)
	draft := map[string]any{"name": "New Reading Room", "kind": "school", "district": "Barishal", "area": "Bakerganj",
		"story": "A small school library.", "needs": []any{map[string]any{"bookId": "bk-hobbit", "wanted": 4}}}
	if r := e.Call("reader", "POST", "/admin/donate/places/save", draft); r.Status != 403 {
		t.Errorf("reader: %d", r.Status)
	}
	list := e.Call("support", "POST", "/admin/donate/places/save", draft).List(t)
	if len(list) != 4 {
		t.Fatalf("saved: %d", len(list))
	}
	id := list[3]["id"].(string)
	draft["id"], draft["name"] = id, "Renamed Room"
	if got := e.Call("support", "POST", "/admin/donate/places/save", draft).List(t); got[3]["name"] != "Renamed Room" || len(got) != 4 {
		t.Errorf("replace: %v", got[3])
	}
	bad := map[string]any{"name": "x", "kind": "school", "district": "d", "area": "a", "story": "A small school library.", "needs": []any{}}
	if r := e.Call("support", "POST", "/admin/donate/places/save", bad); r.Body != nil || r.Refusal != "place_invalid" {
		t.Errorf("invalid: %q", r.Refusal)
	}
	draft["id"] = "rc-nope"
	if r := e.Call("support", "POST", "/admin/donate/places/save", draft); r.Refusal != "place_unknown" {
		t.Errorf("unknown: %q", r.Refusal)
	}
	if got := e.Call("support", "POST", "/admin/donate/places/remove", map[string]any{"id": id}).List(t); len(got) != 3 {
		t.Errorf("remove: %d", len(got))
	}
	if r := e.Call("support", "POST", "/admin/donate/places/remove", map[string]any{"id": id}); r.Body != nil || r.Refusal != "place_unknown" {
		t.Errorf("remove twice: %q", r.Refusal)
	}
}
