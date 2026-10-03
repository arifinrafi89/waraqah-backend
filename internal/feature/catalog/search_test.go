package catalog

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"
)

func ed(id, format, language string, price, stock int, preorder bool) Edition {
	return Edition{ID: id, Format: format, Language: language, PriceBdt: price, Stock: stock, IsPreorder: preorder}
}

func bk(id string, rating float64, editions ...Edition) Book {
	return Book{ID: id, Title: id, Author: "A", CategoryID: "c", AuthorID: "a", PublisherID: "p", Section: "literature",
		OriginalLanguage: "bangla", Editions: editions, AddedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Rating: rating, Tags: []string{}, Classes: []int{}, Exams: []string{}}
}

func idList(books []Book) []string {
	out := []string{}
	for _, b := range books {
		out = append(out, b.ID)
	}
	return out
}

func set(v ...string) map[string]bool {
	m := map[string]bool{}
	for _, s := range v {
		m[s] = true
	}
	return m
}

func ptr[T any](v T) *T { return &v }

// Ported from test/search_filters_test.dart.
func TestEditionFilters(t *testing.T) {
	split := bk("split", 4, ed("s1", "paperback", "bangla", 400, 5, false), ed("s2", "ebook", "english", 300, 5, false))
	match := bk("match", 3.5, ed("m1", "ebook", "bangla", 450, 5, false))
	pre := bk("pre", 4.8, ed("p1", "hardcover", "arabic", 900, 0, true))
	all := NewSnapshot([]Book{split, match, pre}, nil, nil)

	cases := []struct {
		name string
		f    Filters
		want []string
	}{
		{"no params keeps every book", Filters{}, []string{"split", "match", "pre"}},
		{"format", Filters{Formats: set("ebook", "hardcover")}, []string{"split", "match", "pre"}},
		{"language", Filters{Languages: set("arabic")}, []string{"pre"}},
		{"rating", Filters{MinRating: ptr(4.0)}, []string{"split", "pre"}},
		{"price min included max excluded", Filters{MinPrice: ptr(450), MaxPrice: ptr(900)}, []string{"match"}},
		{"max price only", Filters{MaxPrice: ptr(300)}, []string{}},
		{"min price only", Filters{MinPrice: ptr(900)}, []string{"pre"}},
		{"in stock excludes pre-order only", Filters{InStock: true}, []string{"split", "match"}},
		{"one single edition", Filters{Languages: set("bangla"), Formats: set("ebook"), MaxPrice: ptr(500)}, []string{"match"}},
	}
	for _, c := range cases {
		if got := idList(all.Search(c.f, nil)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	// The split book has each part, but only across two editions.
	only := NewSnapshot([]Book{split}, nil, nil)
	if got := only.Search(Filters{Languages: set("bangla"), Formats: set("ebook"), MaxPrice: ptr(500)}, nil); len(got) != 0 {
		t.Errorf("split book matched across editions: %v", idList(got))
	}
}

func TestHiddenBooksOnlyWithIncludeHidden(t *testing.T) {
	hidden := bk("h", 4, ed("h1", "paperback", "bangla", 100, 1, false))
	hidden.Hidden = true
	s := NewSnapshot([]Book{bk("v", 4, ed("v1", "paperback", "bangla", 100, 1, false)), hidden}, nil, nil)
	if got := idList(s.Search(Filters{}, nil)); !reflect.DeepEqual(got, []string{"v"}) {
		t.Errorf("storefront: %v", got)
	}
	if got := idList(s.Search(Filters{IncludeHidden: true}, nil)); !reflect.DeepEqual(got, []string{"v", "h"}) {
		t.Errorf("staff: %v", got)
	}
}

func TestSortOrders(t *testing.T) {
	a := bk("a", 4, ed("a1", "paperback", "english", 500, 1, false))
	b := bk("b", 4, ed("b1", "paperback", "english", 300, 1, false), ed("b2", "ebook", "english", 200, 999, false))
	c := bk("c", 4, ed("c1", "paperback", "english", 900, 1, false))
	b.AddedAt = b.AddedAt.Add(time.Hour)
	c.AddedAt = c.AddedAt.Add(2 * time.Hour)
	all := []Book{a, b, c}
	sold := map[string]int{"a1": 10, "b1": 4, "b2": 9, "c1": 1}
	cases := map[string][]string{
		"priceLow":    {"b", "a", "c"},
		"priceHigh":   {"c", "a", "b"},
		"newest":      {"c", "b", "a"},
		"bestselling": {"b", "a", "c"}, // all editions of a book together: b = 13
		"relevance":   {"a", "b", "c"},
		"":            {"a", "b", "c"},
	}
	for sortName, want := range cases {
		if got := idList(SortBooks(all, sortName, sold)); !reflect.DeepEqual(got, want) {
			t.Errorf("sort %q: got %v want %v", sortName, got, want)
		}
	}
}

// seedSnapshot loads the exported fixtures, so the tests run on the data the app shows.
func seedSnapshot(t *testing.T) *Snapshot {
	t.Helper()
	type seedBook struct {
		Book
		AddedAt string `json:"addedAt"`
	}
	var raw []seedBook
	read := func(name string, v any) {
		b, err := os.ReadFile("../../../seed/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	read("books.json", &raw)
	var authors []Author
	var pubs []Publisher
	read("authors.json", &authors)
	read("publishers.json", &pubs)
	books := make([]Book, 0, len(raw))
	for _, r := range raw {
		b := r.Book
		at, err := time.Parse("2006-01-02T15:04:05.999999", r.AddedAt)
		if err != nil {
			t.Fatal(err)
		}
		b.AddedAt = at
		books = append(books, b)
	}
	return NewSnapshot(books, authors, pubs)
}

// Ported from test/search_bangla_test.dart.
func TestBanglaBanglishAndEnglishFindSapiensFirst(t *testing.T) {
	s := seedSnapshot(t)
	for _, q := range []string{"স্যাপিয়েন্স", "sapiyens", "sapiens"} {
		got := idList(s.Search(Filters{Query: q}, nil))
		if len(got) == 0 || got[0] != "bk-sapiens" {
			t.Errorf("%q: %v", q, got)
		}
	}
	if got := idList(s.Search(Filters{Query: "হারারি"}, nil)); !containsStr(got, "bk-sapiens") {
		t.Errorf("Bangla author name: %v", got)
	}
	if got := idList(s.Search(Filters{Query: "অ্যাটমিক"}, nil)); len(got) == 0 || got[0] != "bk-atomic" {
		t.Errorf("Bangla title as written: %v", got)
	}
}

func TestSuggestionsComeInTheTypedScript(t *testing.T) {
	s := seedSnapshot(t)
	en := s.Suggest("sap")
	if !containsStr(en, "Sapiens: A Brief History of Humankind") || len(en) > 5 {
		t.Errorf("english: %v", en)
	}
	if bn := s.Suggest("স্যাপি"); !containsStr(bn, "স্যাপিয়েন্স") {
		t.Errorf("bangla: %v", bn)
	}
}

func TestDidYouMean(t *testing.T) {
	s := seedSnapshot(t)
	if got := s.DidYouMean("sapeinz"); got == nil || *got != "Sapiens: A Brief History of Humankind" {
		t.Errorf("sapeinz: %v", got)
	}
	for _, q := range []string{"qxqxqx", "zzzz"} {
		if got := s.DidYouMean(q); got != nil {
			t.Errorf("%q should guess nothing, got %q", q, *got)
		}
	}
}

// Ported from test/search_isbn_test.dart.
func TestFullIsbnFindsOnlyItsBook(t *testing.T) {
	s := seedSnapshot(t)
	var book Book
	var isbn string
	for _, b := range s.Books {
		for _, e := range b.Editions {
			if e.ISBN != nil {
				book, isbn = b, *e.ISBN
				break
			}
		}
		if isbn != "" {
			break
		}
	}
	hyphenated := isbn[0:3] + "-" + isbn[3:6] + "-" + isbn[6:12] + "-" + isbn[12:]
	spaced := " " + isbn[0:3] + " " + isbn[3:6] + " " + isbn[6:12] + " " + isbn[12:] + " "
	for _, q := range []string{isbn, hyphenated, spaced} {
		if got := idList(s.Search(Filters{Query: q}, nil)); !reflect.DeepEqual(got, []string{book.ID}) {
			t.Errorf("%q: %v want [%s]", q, got, book.ID)
		}
	}
	for _, q := range []string{isbn[:12], isbn[3:]} {
		if got := s.Search(Filters{Query: q}, nil); len(got) != 0 {
			t.Errorf("partial ISBN %q matched %v", q, idList(got))
		}
	}
}

func TestBestsellingUsesThirtyDaySales(t *testing.T) {
	s := seedSnapshot(t)
	var sold map[string]int
	b, err := os.ReadFile("../../../seed/sales_30d.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &sold); err != nil {
		t.Fatal(err)
	}
	got := s.Search(Filters{Sort: "bestselling"}, sold)
	if got[0].ID != "bk-atomic" || soldCopies(got[0], sold) != 175 {
		t.Errorf("first: %s %d", got[0].ID, soldCopies(got[0], sold))
	}
}
