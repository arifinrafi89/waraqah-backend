package catalogadmin_test

import (
	"slices"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func TestRecordsAndTheirBooks(t *testing.T) {
	e := testenv.New(t)
	authors := e.Call("catalog", "GET", "/admin/catalog/authors", nil).List(t)
	var harari map[string]any
	for _, a := range authors {
		if a["id"] == "au-harari" {
			harari = a
		}
	}
	if harari["bookCount"].(float64) < 1 {
		t.Fatalf("harari: %v", harari)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/authors/delete", map[string]any{"id": "au-harari"}); r.Refusal != "record_in_use" {
		t.Errorf("delete used author: %q", r.Refusal)
	}
	// renaming an author renames them on all their books
	e.Call("catalog", "POST", "/admin/catalog/authors/save", map[string]any{"id": "au-harari", "name": "Y. N. Harari", "nameBn": "ইউভাল"})
	if got := e.Call("", "GET", "/books/detail?id=bk-sapiens", nil).Obj(t)["author"]; got != "Y. N. Harari" {
		t.Errorf("book author after rename: %v", got)
	}
	// a category needs both names and a section; a used category keeps its section
	if r := e.Call("catalog", "POST", "/admin/catalog/categories/save", map[string]any{"name": "Poetry", "nameBn": "", "section": "literature"}); r.Refusal != "record_invalid" {
		t.Errorf("category without Bangla name: %q", r.Refusal)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/categories/save", map[string]any{"id": "cat-academic", "name": "Academic", "nameBn": "শিক্ষা", "section": "religious"}); r.Refusal != "record_invalid" {
		t.Errorf("moving a used category: %q", r.Refusal)
	}
	made := e.Call("catalog", "POST", "/admin/catalog/categories/save", map[string]any{"name": "Poetry", "nameBn": "কবিতা", "section": "literature"}).Obj(t)
	if made["id"] != "cat-poetry" || made["bookCount"] != 0.0 {
		t.Errorf("new category: %v", made)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/categories/delete", map[string]any{"id": "cat-poetry"}); r.Obj(t)["id"] != "cat-poetry" {
		t.Errorf("delete unused: %s", r.Raw)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/categories/delete", map[string]any{"id": "cat-poetry"}); r.Refusal != "record_unknown" {
		t.Errorf("delete twice: %q", r.Refusal)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/publishers/save", map[string]any{"id": "pub-nobody", "name": "X"}); r.Refusal != "record_unknown" {
		t.Errorf("save unknown: %q", r.Refusal)
	}
}

func TestBannersAndTheForcedSeason(t *testing.T) {
	e := testenv.New(t)
	before := e.Call("catalog", "GET", "/admin/catalog/banners", nil).IDs(t)
	made := e.Call("catalog", "POST", "/admin/catalog/banners/save", map[string]any{"id": "", "titleEn": "Eid sale", "titleBn": "ঈদ",
		"subtitleEn": "", "subtitleBn": "", "seed": 1, "target": map[string]any{"kind": "search", "value": "eid"}, "season": "ramadan"}).IDs(t)
	if len(made) != len(before)+1 || made[len(made)-1] != "ban-eid-sale" {
		t.Fatalf("after add: %v", made)
	}
	moved := e.Call("catalog", "POST", "/admin/catalog/banners/move", map[string]any{"id": "ban-eid-sale", "by": -1}).IDs(t)
	if moved[len(moved)-2] != "ban-eid-sale" {
		t.Errorf("after move up: %v", moved)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/banners/move", map[string]any{"id": made[0], "by": -1}); r.Refusal != "banner_invalid" {
		t.Errorf("move the first up: %q", r.Refusal)
	}
	// forcing Ramadan shows its card and its banner; going back to automatic hides them
	if r := e.Call("catalog", "POST", "/admin/catalog/season/save", map[string]any{"season": "ramadan"}); r.Obj(t)["season"] != "ramadan" {
		t.Fatalf("force: %s", r.Raw)
	}
	if card := e.Call("", "GET", "/home/season?date=2026-06-01", nil).Obj(t); card["season"] != "ramadan" {
		t.Errorf("card: %v", card)
	}
	if !slices.Contains(e.Call("", "GET", "/home/banners?date=2026-06-01", nil).IDs(t), "ban-eid-sale") {
		t.Error("the Ramadan banner must show while Ramadan is forced")
	}
	e.Call("catalog", "POST", "/admin/catalog/season/save", map[string]any{"season": nil})
	if r := e.Call("", "GET", "/home/season?date=2026-06-01", nil); r.Body != nil {
		t.Errorf("automatic in June: %s", r.Raw)
	}
	if slices.Contains(e.Call("", "GET", "/home/banners?date=2026-06-01", nil).IDs(t), "ban-eid-sale") {
		t.Error("a Season banner must hide outside its Season")
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/season/save", map[string]any{"season": "winter"}); r.Refusal != "season_unknown" {
		t.Errorf("unknown season: %q", r.Refusal)
	}
	for name, body := range map[string]map[string]any{
		"blank bn title": {"titleEn": "A", "titleBn": " ", "target": map[string]any{"kind": "book", "value": "x"}},
		"no target":      {"titleEn": "A", "titleBn": "এ", "target": map[string]any{"kind": "book", "value": ""}},
		"bad kind":       {"titleEn": "A", "titleBn": "এ", "target": map[string]any{"kind": "page", "value": "x"}},
		"bad season":     {"titleEn": "A", "titleBn": "এ", "target": map[string]any{"kind": "book", "value": "x"}, "season": "winter"},
	} {
		if r := e.Call("catalog", "POST", "/admin/catalog/banners/save", body); r.Refusal != "banner_invalid" {
			t.Errorf("%s: %q", name, r.Refusal)
		}
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/banners/delete", map[string]any{"id": "ban-nope"}); r.Refusal != "banner_unknown" {
		t.Errorf("delete unknown: %q", r.Refusal)
	}
	if got := e.Call("catalog", "POST", "/admin/catalog/banners/delete", map[string]any{"id": "ban-eid-sale"}).IDs(t); len(got) != len(before) {
		t.Errorf("after delete: %v", got)
	}
}

func TestCollectionsAndStaffBooklists(t *testing.T) {
	e := testenv.New(t)
	col := map[string]any{"titleEn": "Winter reads", "titleBn": "শীত", "noteEn": "", "noteBn": "", "section": "literature", "bookIds": []string{"bk-hobbit"}}
	id := e.Call("catalog", "POST", "/admin/catalog/collections/save", col).Obj(t)["id"]
	if id != "col-winter-reads" {
		t.Fatalf("id %v", id)
	}
	if got := e.Call("", "GET", "/collections/detail?id=col-winter-reads", nil).Obj(t); len(got["books"].([]any)) != 1 {
		t.Errorf("collection: %v", got)
	}
	bad := map[string]map[string]any{
		"no books":     {"titleEn": "A", "titleBn": "এ", "bookIds": []string{}},
		"unknown book": {"titleEn": "A", "titleBn": "এ", "bookIds": []string{"bk-nope"}},
		"has a kind":   {"titleEn": "A", "titleBn": "এ", "bookIds": []string{"bk-hobbit"}, "kind": "classList"},
		"bad expert":   {"titleEn": "A", "titleBn": "এ", "bookIds": []string{"bk-hobbit"}, "expertId": "exp-nobody"},
	}
	for name, body := range bad {
		if r := e.Call("catalog", "POST", "/admin/catalog/collections/save", body); r.Refusal != "list_invalid" {
			t.Errorf("%s: %q", name, r.Refusal)
		}
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/collections/delete", map[string]any{"id": id}); r.Obj(t)["id"] != id {
		t.Error("delete collection")
	}
	// staff booklists need a staff kind; a reader own list cannot be touched here
	list := map[string]any{"titleEn": "Exam list", "titleBn": "পরীক্ষা", "kind": "examPrep", "bookIds": []string{"bk-hobbit"}}
	if r := e.Call("catalog", "POST", "/admin/catalog/booklists/save", list); r.Obj(t)["id"] != "bl-exam-list" {
		t.Errorf("save booklist: %s", r.Raw)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/booklists/save", with(list, "kind", "personal")); r.Refusal != "list_invalid" {
		t.Errorf("personal kind: %q", r.Refusal)
	}
	if r := e.Call("catalog", "POST", "/admin/catalog/booklists/delete", map[string]any{"id": "bl-summer-reads"}); r.Refusal != "list_unknown" {
		t.Errorf("a reader own list: %q", r.Refusal)
	}
}

func with(m map[string]any, k string, v any) map[string]any {
	out := map[string]any{}
	for kk, vv := range m {
		out[kk] = vv
	}
	out[k] = v
	return out
}

func TestIsbnLookupAndImport(t *testing.T) {
	e := testenv.New(t)
	if r := e.Call("catalog", "GET", "/admin/catalog/isbn-lookup?isbn=9789840001774", nil).Obj(t); r["inCatalog"] != true || r["bookId"] == nil {
		t.Errorf("in catalog: %v", r)
	}
	out := e.Call("catalog", "GET", "/admin/catalog/isbn-lookup?isbn=9781455586691", nil).Obj(t)
	if out["inCatalog"] != false || out["title"] != "Deep Work" || out["language"] != "english" || out["listPriceBdt"] != 950.0 {
		t.Errorf("outside book: %v", out)
	}
	if r := e.Call("catalog", "GET", "/admin/catalog/isbn-lookup?isbn=9780000000002", nil); r.Body != nil {
		t.Error("unknown isbn")
	}
	row := func(n int, title, author string, price int) map[string]any {
		return draft(map[string]any{"row": n, "author": author, "publisher": "Imported Press", "title": title, "authorId": "", "publisherId": "",
			"editions": []map[string]any{{"format": "paperback", "language": "english", "priceBdt": price, "stock": 3}}})
	}
	res := e.Call("catalog", "POST", "/admin/catalog/import", map[string]any{"books": []map[string]any{
		row(2, "Good One", "New Writer", 300),
		row(3, "Bad Price", "Other Writer", 0), // refused: its new author must not be left behind
		row(4, "Good Two", "new writer", 350),  // the author made by row 2 is reused (any case)
	}}).Obj(t)
	if res["imported"] != 2.0 || len(res["skipped"].([]any)) != 1 || res["skipped"].([]any)[0].(map[string]any)["row"] != 3.0 {
		t.Fatalf("import: %v", res)
	}
	n, other := 0, false
	for _, a := range e.Call("catalog", "GET", "/admin/catalog/authors", nil).List(t) {
		if a["name"] == "New Writer" {
			n++
		}
		if a["name"] == "Other Writer" {
			other = true
		}
	}
	if n != 1 || other {
		t.Errorf("authors: new=%d other=%v", n, other)
	}
}
