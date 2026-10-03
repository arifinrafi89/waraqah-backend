package catalogadmin_test

import (
	"context"
	"slices"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

type countingSweeper struct{ n int }

func (c *countingSweeper) Sweep(context.Context) error { c.n++; return nil }

func draft(over map[string]any) map[string]any {
	d := map[string]any{
		"title": "Staff Test Book", "titleBn": "", "authorId": "au-harari", "publisherId": "pub-harper",
		"section": "academic", "categoryId": "cat-academic", "originalLanguage": "english", "coverSeed": 2,
		"editions": []map[string]any{{"id": "", "format": "paperback", "language": "english", "priceBdt": 400, "stock": 10}},
		"classes":  []int{}, "exams": []string{}, "subjectId": "",
	}
	for k, v := range over {
		d[k] = v
	}
	return d
}

func TestOnlyCatalogStaffCanUseAdminCatalog(t *testing.T) {
	e := testenv.New(t)
	paths := []struct{ method, path string }{
		{"GET", "/admin/catalog/banners"}, {"GET", "/admin/catalog/categories"}, {"GET", "/admin/catalog/low-stock"},
		{"POST", "/admin/catalog/books/hide"}, {"POST", "/admin/catalog/import"}, {"GET", "/admin/catalog/season"},
	}
	for _, p := range paths {
		var body any
		if p.method == "POST" {
			body = map[string]any{}
		}
		want := map[string]int{"": 401, "reader": 403, "moderator": 403, "support": 403, "catalog": 200, "admin": 200}
		for as, code := range want {
			got := e.Call(as, p.method, p.path, body).Status
			if got != code {
				t.Errorf("%s %s as %q: %d want %d", p.method, p.path, as, got, code)
			}
		}
	}
}

func TestSaveBookRulesAndEdits(t *testing.T) {
	e := testenv.New(t)
	sw := &countingSweeper{}
	e.Deps.Sweeper = sw
	e = rebuild(t, e, sw)

	made := e.Call("catalog", "POST", "/admin/catalog/books/save", draft(nil)).Obj(t)
	id := made["id"].(string)
	if id != "bk-staff-test-book" || made["author"] != "Yuval Noah Harari" || made["rating"] != 0.0 || made["hidden"] != false {
		t.Fatalf("made: %v", made)
	}
	eds := made["editions"].([]any)
	if len(eds) != 1 || eds[0].(map[string]any)["id"] != id+"-pb-en" {
		t.Errorf("edition ids: %v", eds)
	}
	if sw.n != 1 {
		t.Errorf("alerts swept %d times, want 1", sw.n)
	}
	// the new book is on the storefront
	if !slices.Contains(e.Call("", "GET", "/books?q=staff+test", nil).IDs(t), id) {
		t.Error("new book not searchable (cache not invalidated?)")
	}
	// the same title gets another id
	if again := e.Call("catalog", "POST", "/admin/catalog/books/save", draft(nil)).Obj(t); again["id"] != id+"-2" {
		t.Errorf("second id: %v", again["id"])
	}
	// editing keeps rating, tags, added date and hidden
	edit := draft(map[string]any{"id": id, "title": "Staff Test Book (2nd ed)"})
	edited := e.Call("catalog", "POST", "/admin/catalog/books/save", edit).Obj(t)
	if edited["id"] != id || edited["title"] != "Staff Test Book (2nd ed)" || edited["addedAt"] != made["addedAt"] {
		t.Errorf("edited: %v", edited)
	}

	refusals := map[string]map[string]any{
		"blank title":      draft(map[string]any{"title": " "}),
		"unknown author":   draft(map[string]any{"authorId": "au-nobody"}),
		"unknown category": draft(map[string]any{"categoryId": "cat-nobody"}),
		"wrong section":    draft(map[string]any{"section": "religious"}),
		"no editions":      draft(map[string]any{"editions": []map[string]any{}}),
		"unknown id":       draft(map[string]any{"id": "bk-nobody"}),
		"bad enum":         draft(map[string]any{"originalLanguage": "klingon"}),
		"bad price":        draft(map[string]any{"editions": []map[string]any{{"format": "paperback", "language": "english", "priceBdt": 0, "stock": 1}}}),
		"isbn taken": draft(map[string]any{"editions": []map[string]any{{"format": "paperback", "language": "english", "priceBdt": 5, "stock": 1,
			"isbn": "9789840001774"}}}),
	}
	for name, body := range refusals {
		r := e.Call("catalog", "POST", "/admin/catalog/books/save", body)
		want := "book_invalid"
		if name == "unknown id" {
			want = "book_unknown"
		}
		if r.Refusal != want || r.Body != nil {
			t.Errorf("%s: %q %s", name, r.Refusal, r.Raw)
		}
	}
}

func rebuild(t *testing.T, e *testenv.Env, sw *countingSweeper) *testenv.Env {
	t.Helper()
	e.Deps.Sweeper = sw
	e.Deps.CatalogAdmin.Sweeper = sw // the service is built once in deps.go and shared with the dashboard
	e.H = e.Rebuild()
	return e
}

func TestHideAndStock(t *testing.T) {
	e := testenv.New(t)
	if r := e.Call("catalog", "POST", "/admin/catalog/books/hide", map[string]any{"id": "bk-sapiens", "hidden": true}); r.Obj(t)["hidden"] != true {
		t.Fatalf("hide: %s", r.Raw)
	}
	if slices.Contains(e.Call("", "GET", "/books?q=sapiens", nil).IDs(t), "bk-sapiens") {
		t.Error("a hidden book is still on the storefront")
	}
	e.Call("catalog", "POST", "/admin/catalog/books/hide", map[string]any{"id": "bk-sapiens", "hidden": false})
	if !slices.Contains(e.Call("", "GET", "/books?q=sapiens", nil).IDs(t), "bk-sapiens") {
		t.Error("unhide")
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/books/hide", map[string]any{"id": "bk-nope", "hidden": true}); r.Refusal != "book_unknown" {
		t.Errorf("unknown: %q", r.Refusal)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/editions/stock", map[string]any{"editionId": "bk-atomic-pb-en", "stock": 2}); r.Obj(t)["stock"] != 2.0 {
		t.Errorf("stock: %s", r.Raw)
	}
	low := e.Call("catalog", "GET", "/admin/catalog/low-stock", nil).List(t)
	if !slices.ContainsFunc(low, func(m map[string]any) bool { return m["editionId"] == "bk-atomic-pb-en" }) {
		t.Error("a printed edition at 2 copies must be on the low stock list")
	}
	for i, m := range low {
		if i > 0 && m["stock"].(float64) < low[i-1]["stock"].(float64) {
			t.Fatal("low stock must be lowest first")
		}
	}
	for name, body := range map[string]map[string]any{
		"negative": {"editionId": "bk-atomic-pb-en", "stock": -1},
		"ebook":    {"editionId": "bk-atomic-eb-en", "stock": 5},
		"unknown":  {"editionId": "nope", "stock": 5},
	} {
		if r := e.Call("catalog", "POST", "/admin/catalog/editions/stock", body); r.Refusal != "stock_invalid" {
			t.Errorf("%s: %q", name, r.Refusal)
		}
	}
}

func TestPriceDropIsRememberedAsALow(t *testing.T) {
	e := testenv.New(t)
	book := e.Call("", "GET", "/books/detail?id=bk-atomic", nil).Obj(t)
	eds := book["editions"].([]any)
	var edits []map[string]any
	for _, x := range eds {
		m := x.(map[string]any)
		if m["id"] == "bk-atomic-pb-en" {
			m["priceBdt"] = 480.0
		}
		edits = append(edits, m)
	}
	d := draft(map[string]any{"id": "bk-atomic", "title": book["title"], "authorId": book["authorId"], "publisherId": book["publisherId"],
		"section": book["section"], "categoryId": book["categoryId"], "originalLanguage": book["originalLanguage"],
		"coverSeed": book["coverSeed"], "editions": edits, "subjectId": ""})
	if book["subjectId"] != nil {
		d["subjectId"] = book["subjectId"]
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/books/save", d); r.Body == nil {
		t.Fatalf("save: %s", r.Raw)
	}
	lows := e.Call("", "GET", "/books/price-lows?id=bk-atomic", nil).Obj(t)
	if lows["bk-atomic-pb-en"] != 480.0 {
		t.Errorf("lows: %v", lows)
	}
}
