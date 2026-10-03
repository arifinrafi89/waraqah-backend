package catalog_test

import (
	"slices"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func TestHiddenBooksOnlyForCatalogStaff(t *testing.T) {
	e := testenv.New(t)
	if _, err := e.Deps.DB.Conn.Exec(t.Context(), "UPDATE books SET hidden = true WHERE id = 'bk-sapiens'"); err != nil {
		t.Fatal(err)
	}
	e.Deps.Catalog.Invalidate()
	cases := []struct {
		as   string
		path string
		want bool
	}{
		{"", "/books?q=sapiens&includeHidden=true", false},
		{"reader", "/books?q=sapiens&includeHidden=true", false},
		{"moderator", "/books?q=sapiens&includeHidden=true", false},
		{"catalog", "/books?q=sapiens", false},
		{"catalog", "/books?q=sapiens&includeHidden=true", true},
		{"admin", "/books?q=sapiens&includeHidden=true", true},
	}
	for _, c := range cases {
		got := slices.Contains(e.Call(c.as, "GET", c.path, nil).IDs(t), "bk-sapiens")
		if got != c.want {
			t.Errorf("%q %s: visible=%v want %v", c.as, c.path, got, c.want)
		}
	}
	// the page of a hidden book still opens from an old link
	if r := e.Call("", "GET", "/books/detail?id=bk-sapiens", nil); r.Obj(t)["hidden"] != true {
		t.Errorf("detail: %s", r.Raw)
	}
	// hidden books leave collections and subjects
	for _, c := range e.Call("", "GET", "/collections", nil).List(t) {
		for _, b := range c["books"].([]any) {
			if b.(map[string]any)["id"] == "bk-sapiens" {
				t.Errorf("hidden book still in collection %v", c["id"])
			}
		}
	}
}

func TestSearchAndFiltersOverHTTP(t *testing.T) {
	e := testenv.New(t)
	for _, q := range []string{"স্যাপিয়েন্স", "sapiyens", "sapiens"} {
		if ids := e.Call("", "GET", "/books?q="+q, nil).IDs(t); len(ids) == 0 || ids[0] != "bk-sapiens" {
			t.Errorf("%q: %v", q, ids)
		}
	}
	all := e.Call("", "GET", "/books", nil).IDs(t)
	if len(all) < 30 {
		t.Fatalf("storefront has %d books", len(all))
	}
	low := e.Call("", "GET", "/books?sort=priceLow", nil).List(t)
	prev := 0.0
	for _, b := range low {
		fromPrice := 1e9
		for _, ed := range b["editions"].([]any) {
			m := ed.(map[string]any)
			if m["stock"].(float64) > 0 || m["isPreorder"] == true {
				fromPrice = min(fromPrice, m["priceBdt"].(float64))
			}
		}
		if fromPrice == 1e9 {
			continue
		}
		if fromPrice < prev {
			t.Fatalf("price sort broken at %v", b["id"])
		}
		prev = fromPrice
	}
	if first := e.Call("", "GET", "/books?sort=bestselling", nil).IDs(t)[0]; first != "bk-atomic" {
		t.Errorf("bestselling first: %s", first)
	}
	if got := e.Call("", "GET", "/books?section=religious&inStock=true&minRating=4", nil).List(t); len(got) == 0 {
		t.Error("filters returned nothing")
	}
}

func TestQuestionsUseTheTokenNotTheBody(t *testing.T) {
	e := testenv.New(t)
	asked := e.Call("reader", "POST", "/books/questions/ask", map[string]any{"bookId": "bk-cleancode", "text": "Is it the second edition?", "name": "Mallory"}).List(t)
	if len(asked) != 1 || asked[0]["askerName"] != "Reader" {
		t.Fatalf("asked: %v", asked)
	}
	qid := asked[0]["id"].(string)
	// a reader claiming isStaff is still not staff; the name still comes from the account
	r := e.Call("reader", "POST", "/books/questions/answer", map[string]any{"bookId": "bk-cleancode", "questionId": qid, "text": "I think so.", "name": "Waraqah", "isStaff": true}).List(t)
	a := r[0]["answers"].([]any)[0].(map[string]any)
	if a["isStaff"] != false || a["authorName"] != "Reader" {
		t.Errorf("reader answer: %v", a)
	}
	s := e.Call("catalog", "POST", "/books/questions/answer", map[string]any{"bookId": "bk-cleancode", "questionId": qid, "text": "Yes, second edition."}).List(t)
	b := s[0]["answers"].([]any)[1].(map[string]any)
	if b["isStaff"] != true || b["authorName"] != "Waraqah" {
		t.Errorf("staff answer: %v", b)
	}
	if got := e.Call("", "GET", "/books/questions?id=bk-cleancode", nil).List(t); len(got) != 1 {
		t.Errorf("guest read: %v", got)
	}
	// refusals
	if r := e.Call("reader", "POST", "/books/questions/ask", map[string]any{"bookId": "bk-nope", "text": "x"}); r.Refusal != "book_unknown" || r.Body != nil {
		t.Errorf("unknown book: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/books/questions/ask", map[string]any{"bookId": "bk-cleancode", "text": "  "}); r.Refusal != "question_invalid" {
		t.Errorf("blank: %q", r.Refusal)
	}
	if r := e.Call("", "POST", "/books/questions/ask", map[string]any{"bookId": "bk-cleancode", "text": "x"}); r.Status != 401 {
		t.Errorf("guest ask: %d", r.Status)
	}
}

func TestOwnBooklists(t *testing.T) {
	e := testenv.New(t)
	guest := e.Call("", "GET", "/booklists", nil).List(t)
	mine := 0
	for _, l := range e.Call("reader", "GET", "/booklists", nil).List(t) {
		if l["isMine"] == true {
			mine++
		}
	}
	if mine != 1 || len(guest) != len(e.Call("reader", "GET", "/booklists", nil).List(t))-1 {
		t.Errorf("a reader sees staff lists plus their own (mine=%d guest=%d)", mine, len(guest))
	}
	made := e.Call("reader", "POST", "/booklists/mine/save", map[string]any{"name": "  Weekend  ", "bookIds": []string{"bk-cleancode", "bk-sapiens"}}).Obj(t)
	if made["titleEn"] != "Weekend" || made["isMine"] != true || made["kind"] != "personal" || len(made["books"].([]any)) != 2 {
		t.Fatalf("made: %v", made)
	}
	id := made["id"].(string)
	renamed := e.Call("reader", "POST", "/booklists/mine/save", map[string]any{"id": id, "name": "Weekdays"}).Obj(t)
	if renamed["titleEn"] != "Weekdays" || len(renamed["bookIds"].([]any)) != 2 {
		t.Errorf("rename keeps the books: %v", renamed)
	}
	// nobody else sees it or can change it
	if r := e.Call("", "GET", "/booklists/detail?id="+id, nil); r.Body != nil {
		t.Error("a guest saw a reader list")
	}
	if r := e.Call("moderator", "POST", "/booklists/mine/save", map[string]any{"id": id, "name": "Mine now"}); r.Refusal != "booklist_not_yours" {
		t.Errorf("other reader save: %q", r.Refusal)
	}
	if r := e.Call("moderator", "POST", "/booklists/mine/delete", map[string]any{"id": id}); r.Refusal != "booklist_not_yours" {
		t.Errorf("other reader delete: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/booklists/mine/delete", map[string]any{"id": "bl-class-8"}); r.Refusal != "booklist_not_yours" {
		t.Errorf("staff list delete: %q", r.Refusal)
	}
	for name, body := range map[string]map[string]any{
		"no name":        {},
		"blank name":     {"name": "  "},
		"duplicate book": {"name": "X", "bookIds": []string{"bk-sapiens", "bk-sapiens"}},
		"unknown book":   {"name": "X", "bookIds": []string{"bk-nope"}},
	} {
		if r := e.Call("reader", "POST", "/booklists/mine/save", body); r.Refusal != "booklist_invalid" || r.Body != nil {
			t.Errorf("%s: %q", name, r.Refusal)
		}
	}
	if r := e.Call("reader", "POST", "/booklists/mine/delete", map[string]any{"id": id}); r.Obj(t)["id"] != id {
		t.Error("delete own")
	}
}

func TestSmallReads(t *testing.T) {
	e := testenv.New(t)
	if d := e.Call("", "GET", "/books/series?id=bk-hpstone", nil).Obj(t); len(d["entries"].([]any)) != 7 {
		t.Errorf("series: %v", d)
	}
	if r := e.Call("", "GET", "/books/series?id=bk-nope", nil); r.Body != nil {
		t.Error("series of an unknown book")
	}
	lows := e.Call("", "GET", "/books/price-lows?id=bk-sapiens", nil).Obj(t)
	if lows["bk-sapiens-pb-en"] != 620.0 || lows["bk-sapiens-pb-bn"] != 520.0 {
		t.Errorf("lows: %v", lows)
	}
	if r := e.Call("", "GET", "/books/price-lows?id=bk-nope", nil); len(r.Obj(t)) != 0 {
		t.Error("unknown book lows must be an empty object")
	}
	used := e.Call("", "GET", "/books/used-options?id=bk-sapiens", nil).Obj(t)
	if used["certifiedUsed"] != nil || used["resaleValueBdt"] == nil {
		t.Errorf("used options: %v", used)
	}
	if r := e.Call("", "GET", "/books/used-options?id=bk-nope", nil); r.Body != nil {
		t.Error("used options of an unknown book")
	}
	if got := e.Call("", "GET", "/categories?section=academic", nil).List(t); len(got) == 0 {
		t.Error("categories")
	}
	if got := e.Call("", "GET", "/categories", nil).List(t); len(got) != 0 {
		t.Error("categories need a section")
	}
	if got := e.Call("", "GET", "/collections?hasExpert=true", nil).List(t); len(got) == 0 || got[0]["expert"] == nil {
		t.Error("expert picks")
	}
	if r := e.Call("", "GET", "/experts/detail?id=nope", nil); r.Body != nil {
		t.Error("unknown expert")
	}
}
