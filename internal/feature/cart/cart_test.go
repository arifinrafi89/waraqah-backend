package cart_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func lines(t *testing.T, r testenv.Result) []map[string]any {
	t.Helper()
	l, ok := r.Obj(t)["lines"].([]any)
	if !ok {
		t.Fatalf("no lines: %s", r.Raw)
	}
	out := make([]map[string]any, 0, len(l))
	for _, v := range l {
		out = append(out, v.(map[string]any))
	}
	return out
}

func add(e *testenv.Env, as, kind, id string) testenv.Result {
	return e.Call(as, "POST", "/cart/add", map[string]any{"kind": kind, "id": id})
}

func TestGuestAndPerReaderCarts(t *testing.T) {
	e := testenv.New(t)
	if got := lines(t, e.Call("", "GET", "/cart", nil)); len(got) != 0 {
		t.Errorf("guest cart: %v", got)
	}
	if r := e.Call("", "POST", "/cart/add", map[string]any{"kind": "edition", "id": "bk-atomic-pb-en"}); r.Status != 401 {
		t.Errorf("guest add: %d", r.Status)
	}
	add(e, "reader", "edition", "bk-atomic-pb-en")
	if got := lines(t, e.Call("moderator", "GET", "/cart", nil)); len(got) != 0 {
		t.Error("a cart must not leak to another reader")
	}
}

func TestQuantitiesAndCaps(t *testing.T) {
	e := testenv.New(t)
	add(e, "reader", "edition", "bk-atomic-pb-en")
	got := lines(t, add(e, "reader", "edition", "bk-atomic-pb-en"))
	if len(got) != 1 || got[0]["quantity"] != 2.0 {
		t.Fatalf("adding twice keeps one line: %v", got)
	}
	max := got[0]["maxQuantity"].(float64)
	if max > 10 {
		t.Error("never more than 10 of a printed edition per order")
	}
	// an update stays between 1 and the maximum
	for q, want := range map[int]float64{0: 1, 5: 5, 999: max} {
		l := lines(t, e.Call("reader", "POST", "/cart/update", map[string]any{"lineId": "edition-bk-atomic-pb-en", "quantity": q}))
		if l[0]["quantity"] != want {
			t.Errorf("quantity %d: got %v want %v", q, l[0]["quantity"], want)
		}
	}
	// an eBook is one copy
	var eb []map[string]any
	for i := 0; i < 3; i++ {
		eb = lines(t, add(e, "reader", "edition", "bk-sapiens-eb-en"))
	}
	for _, l := range eb {
		if l["itemId"] == "bk-sapiens-eb-en" && (l["quantity"] != 1.0 || l["maxQuantity"] != 1.0) {
			t.Errorf("ebook line: %v", l)
		}
	}
	// stock falling below the quantity shows the lower maximum
	e.Call("catalog", "POST", "/admin/catalog/editions/stock", map[string]any{"editionId": "bk-atomic-pb-en", "stock": 3})
	for _, l := range lines(t, e.Call("reader", "GET", "/cart", nil)) {
		if l["itemId"] == "bk-atomic-pb-en" && (l["maxQuantity"] != 3.0 || l["quantity"].(float64) > 3) {
			t.Errorf("after the stock fell: %v", l)
		}
	}
	// removing
	if got := lines(t, e.Call("reader", "POST", "/cart/remove", map[string]any{"lineId": "edition-bk-atomic-pb-en"})); len(got) != 1 {
		t.Errorf("after remove: %v", got)
	}
	if got := lines(t, e.Call("reader", "POST", "/cart/remove", map[string]any{"lineId": "edition-nope"})); len(got) != 1 {
		t.Errorf("removing an unknown line changes nothing: %v", got)
	}
}

func TestFlashBundleAndRefusedItems(t *testing.T) {
	e := testenv.New(t)
	flash := lines(t, add(e, "reader", "edition", "bk-sherlock-pb-en"))[0]
	if flash["unitPriceBdt"] != 299.0 || flash["listPriceBdt"].(float64) <= 299 {
		t.Errorf("flash line: %v", flash)
	}
	bundle := lines(t, add(e, "reader", "bundle", "big-ideas"))[1]
	if bundle["kind"] != "bundle" || bundle["unitPriceBdt"] != 1450.0 || bundle["maxQuantity"] != 5.0 || bundle["listPriceBdt"].(float64) <= 1450 ||
		bundle["id"] != "bundle-big-ideas" {
		t.Errorf("bundle line: %v", bundle)
	}
	for name, body := range map[string]map[string]any{
		"unknown edition":  {"kind": "edition", "id": "bk-nope-pb-en"},
		"unknown bundle":   {"kind": "bundle", "id": "nope"},
		"a reader listing": {"kind": "listing", "id": "p2p-1"},
		"no used stock":    {"kind": "certifiedUsed", "id": "cu-1"},
		"unknown kind":     {"kind": "gift", "id": "x"},
	} {
		r := e.Call("reader", "POST", "/cart/add", body)
		if r.Status != 200 || r.Refusal != "cart_item_unknown" || len(lines(t, r)) != 2 {
			t.Errorf("%s: %d %q %s", name, r.Status, r.Refusal, r.Raw)
		}
	}
}

func TestDealsEndpoint(t *testing.T) {
	e := testenv.New(t)
	r := e.Call("", "GET", "/deals", nil)
	d := r.Obj(t)
	flash := d["flashSale"].(map[string]any)
	if len(flash["items"].([]any)) != 3 || len(d["bundles"].([]any)) != 2 || len(d["preorders"].([]any)) != 1 {
		t.Fatalf("deals: %s", r.Raw)
	}
	for _, i := range flash["items"].([]any) {
		m := i.(map[string]any)
		if m["priceBdt"].(float64) >= m["regularPriceBdt"].(float64) {
			t.Errorf("a flash item must be cheaper than usual: %v", m)
		}
	}
}
