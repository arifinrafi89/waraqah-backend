package alerts_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func set(e *testenv.Env, kind string, target any) testenv.Result {
	body := map[string]any{"kind": kind, "bookId": "bk-atomic", "editionId": "bk-atomic-pb-en"}
	if target != nil {
		body["targetPriceBdt"] = target
	}
	return e.Call("reader", "POST", "/alerts/set", body)
}

// setPrice saves Atomic Habits through Admin → Catalog with a new price for its paperback.
func setPrice(t *testing.T, e *testenv.Env, price float64) {
	t.Helper()
	book := e.Call("", "GET", "/books/detail?id=bk-atomic", nil).Obj(t)
	var eds []map[string]any
	for _, x := range book["editions"].([]any) {
		m := x.(map[string]any)
		if m["id"] == "bk-atomic-pb-en" {
			m["priceBdt"] = price
		}
		eds = append(eds, m)
	}
	subject := ""
	if s, ok := book["subjectId"].(string); ok {
		subject = s
	}
	r := e.Call("catalog", "POST", "/admin/catalog/books/save", map[string]any{"id": "bk-atomic", "title": book["title"], "titleBn": "",
		"authorId": book["authorId"], "publisherId": book["publisherId"], "section": book["section"], "categoryId": book["categoryId"],
		"originalLanguage": book["originalLanguage"], "coverSeed": book["coverSeed"], "editions": eds, "classes": []int{}, "exams": []string{}, "subjectId": subject})
	if r.Body == nil {
		t.Fatalf("save: %s", r.Raw)
	}
}

func alertNotes(t *testing.T, e *testenv.Env) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, n := range e.Call("reader", "GET", "/notifications", nil).List(t) {
		if n["kind"] == "alertTriggered" {
			out = append(out, n)
		}
	}
	return out
}

func TestPriceDropFiresOnceAndNotifies(t *testing.T) {
	e := testenv.New(t)
	got := set(e, "priceDrop", 500).List(t)
	if len(got) != 1 || got[0]["isTriggered"] != false || got[0]["currentPriceBdt"] != 590.0 || got[0]["bookTitle"] != "Atomic Habits" {
		t.Fatalf("set: %v", got)
	}
	before := len(alertNotes(t, e))
	setPrice(t, e, 480)
	if got := e.Call("reader", "GET", "/alerts", nil).List(t); got[0]["isTriggered"] != true {
		t.Errorf("after the drop: %v", got)
	}
	n := alertNotes(t, e)
	if len(n) != before+1 || n[0]["params"].(map[string]any)["reason"] != "priceDrop" || n[0]["params"].(map[string]any)["title"] != "Atomic Habits" {
		t.Fatalf("notification: %v", n)
	}
	setPrice(t, e, 470) // a second change must not notify again
	if len(alertNotes(t, e)) != before+1 {
		t.Error("a fired alert notifies once")
	}
}

func TestAlertAlreadyFiringWhenSetNeedsNoNotification(t *testing.T) {
	e := testenv.New(t)
	before := len(alertNotes(t, e))
	if got := set(e, "priceDrop", 700).List(t); got[0]["isTriggered"] != true {
		t.Fatalf("set: %v", got)
	}
	setPrice(t, e, 580)
	if len(alertNotes(t, e)) != before {
		t.Error("an alert that already fired when set is shown on its page, not notified")
	}
}

func TestReplaceRemoveAndRefusals(t *testing.T) {
	e := testenv.New(t)
	set(e, "priceDrop", 500)
	got := set(e, "priceDrop", 450).List(t)
	if len(got) != 1 || got[0]["targetPriceBdt"] != 450.0 {
		t.Errorf("same kind on the same edition is replaced: %v", got)
	}
	both := set(e, "backInStock", nil).List(t)
	if len(both) != 2 {
		t.Fatalf("a second kind is added: %v", both)
	}
	e.Call("moderator", "POST", "/alerts/remove", map[string]any{"id": both[0]["id"]})
	if len(e.Call("reader", "GET", "/alerts", nil).List(t)) != 2 {
		t.Error("another reader must not remove my alert")
	}
	if got := e.Call("reader", "POST", "/alerts/remove", map[string]any{"id": both[0]["id"]}).List(t); len(got) != 1 {
		t.Errorf("remove: %v", got)
	}
	for name, body := range map[string]map[string]any{
		"unknown edition": {"kind": "priceDrop", "bookId": "bk-atomic", "editionId": "nope"},
		"wrong book":      {"kind": "priceDrop", "bookId": "bk-sapiens", "editionId": "bk-atomic-pb-en"},
		"unknown kind":    {"kind": "gossip", "bookId": "bk-atomic", "editionId": "bk-atomic-pb-en"},
	} {
		r := e.Call("reader", "POST", "/alerts/set", body)
		if r.Refusal != "alert_invalid" || len(r.List(t)) != 1 {
			t.Errorf("%s: %q %s", name, r.Refusal, r.Raw)
		}
	}
	if got := e.Call("", "GET", "/alerts", nil).List(t); len(got) != 0 {
		t.Error("guest alerts")
	}
}

func TestBackInStockFires(t *testing.T) {
	e := testenv.New(t)
	e.Call("catalog", "POST", "/admin/catalog/editions/stock", map[string]any{"editionId": "bk-atomic-pb-en", "stock": 0})
	if got := set(e, "backInStock", nil).List(t); got[0]["isTriggered"] != false {
		t.Fatalf("out of stock: %v", got)
	}
	before := len(alertNotes(t, e))
	e.Call("catalog", "POST", "/admin/catalog/editions/stock", map[string]any{"editionId": "bk-atomic-pb-en", "stock": 5})
	n := alertNotes(t, e)
	if len(n) != before+1 || n[0]["params"].(map[string]any)["reason"] != "backInStock" {
		t.Errorf("back in stock: %v", n)
	}
}
