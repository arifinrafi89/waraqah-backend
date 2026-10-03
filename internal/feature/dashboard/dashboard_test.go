package dashboard_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func searches(t *testing.T, e *testenv.Env) map[string]float64 {
	t.Helper()
	out := map[string]float64{}
	for _, s := range e.Call("admin", "GET", "/admin/dashboard", nil).Obj(t)["topSearches"].([]any) {
		m := s.(map[string]any)
		out[m["term"].(string)] = m["count"].(float64)
	}
	return out
}

func TestDashboardMatchesTheOtherScreens(t *testing.T) {
	e := testenv.New(t)
	d := e.Call("admin", "GET", "/admin/dashboard", nil).Obj(t)
	if d["listingsWaiting"] != 3.0 || d["openReports"] != 3.0 || d["openDisputes"] != 1.0 || d["sellBackWaiting"] != 2.0 {
		t.Errorf("waiting: %v", d)
	}
	queue := e.Call("moderator", "GET", "/moderation/listings", nil).List(t)
	if float64(len(queue)) != d["listingsWaiting"] {
		t.Errorf("listings waiting %v, queue %d", d["listingsWaiting"], len(queue))
	}
	top := d["topRequested"].([]any)
	if len(top) == 0 || top[0].(map[string]any)["title"] != "Calculus: Early Transcendentals" {
		t.Errorf("most requested: %v", top)
	}
	// An order placed today counts in today's numbers.
	e.Call("reader", "POST", "/cart/add", map[string]any{"kind": "edition", "id": "bk-atomic-pb-en"})
	rec := e.Call("reader", "POST", "/orders/place", map[string]any{"addressId": "addr-home", "payment": "bkash"}).Obj(t)
	after := e.Call("support", "GET", "/admin/dashboard", nil).Obj(t)
	if after["ordersToday"] != d["ordersToday"].(float64)+1 || after["salesTodayBdt"] != d["salesTodayBdt"].(float64)+rec["totalBdt"].(float64) ||
		after["ordersToShip"] != d["ordersToShip"].(float64)+1 {
		t.Errorf("after an order: %v", after)
	}
	if r := e.Call("reader", "GET", "/admin/dashboard", nil); r.Status != 403 {
		t.Errorf("a reader: %d", r.Status)
	}
	if r := e.Call("", "GET", "/admin/dashboard", nil); r.Status != 401 {
		t.Errorf("a guest: %d", r.Status)
	}
}

func TestSearchLogCountsGrowingSearchesOnce(t *testing.T) {
	e := testenv.New(t)
	if got := searches(t, e); got["sapiens"] != 14 {
		t.Fatalf("seeded: %v", got)
	}
	for _, q := range []string{"sa", "sap", "sapi", "sapiens"} {
		e.Call("reader", "GET", "/books?q="+q, nil)
	}
	got := searches(t, e)
	if got["sapiens"] != 15 || got["sap"] != 0 || got["sapi"] != 0 {
		t.Errorf("a growing live search counts once: %v", got)
	}
	// Staff's own list with hidden books is not a reader's search.
	for range 5 {
		e.Call("catalog", "GET", "/books?q=tafsir&includeHidden=true", nil)
	}
	if got := searches(t, e); got["tafsir"] != 5 {
		t.Errorf("staff searches: %v", got["tafsir"])
	}
	// Another reader's search of the same term counts again.
	e.Call("moderator", "GET", "/books?q=sapiens", nil)
	if got := searches(t, e); got["sapiens"] != 16 {
		t.Errorf("another reader: %v", got["sapiens"])
	}
}
