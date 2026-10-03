package catalogadmin

import (
	"reflect"
	"strings"
	"testing"
)

func set(v ...RuleError) Errors {
	e := Errors{}
	for _, r := range v {
		e[r] = true
	}
	return e
}

func str(s string) *string { return &s }
func num(n int) *int       { return &n }

var pb = EditionDraft{Format: "paperback", Language: "bangla", PriceBdt: 300, Stock: 5}

func draft() BookDraft {
	return BookDraft{Title: "Shonar Tori", AuthorID: "au-x", PublisherID: "pub-x", Section: "literature",
		CategoryID: "cat-fiction", Editions: []EditionDraft{pb}}
}

func book(d BookDraft) Errors { return CheckBook(d, "literature") }

func edition(e EditionDraft, siblings ...EditionDraft) Errors {
	return CheckEdition(e, siblings, map[string]bool{"9789840001774": true})
}

func eq(t *testing.T, name string, got, want Errors) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: got %v want %v", name, got, want)
	}
}

// Ported from test/catalog_admin_rules_test.dart.
func TestBookRules(t *testing.T) {
	eq(t, "full book", book(draft()), nil)
	d := draft()
	d.Title = "  "
	eq(t, "blank title", book(d), set(TitleBlank))
	d = draft()
	d.AuthorID, d.PublisherID = "", ""
	eq(t, "author and publisher", book(d), set(AuthorMissing, PublisherMissing))
	d = draft()
	d.CategoryID = ""
	eq(t, "category", book(d), set(CategoryMissing))
	eq(t, "wrong section", CheckBook(draft(), "religious"), set(CategoryWrongSection))
	eq(t, "unknown category", CheckBook(draft(), ""), set(CategoryWrongSection))
	d = draft()
	d.Editions = nil
	eq(t, "no editions", book(d), set(NoEditions))
}

func TestClassesAndExamsMustBeOnesTheSectionOffers(t *testing.T) {
	d := draft()
	d.Classes = []int{8}
	eq(t, "class outside school", book(d), set(ClassNotAllowed))
	d.Section = "schoolCollege"
	eq(t, "class in school", CheckBook(d, "schoolCollege"), nil)
	d.Classes = []int{5}
	eq(t, "class 5", CheckBook(d, "schoolCollege"), set(ClassNotAllowed))
	d.Classes, d.Exams = nil, []string{"bcs"}
	eq(t, "bcs in school", CheckBook(d, "schoolCollege"), set(ExamNotAllowed))
	d.Section, d.Exams = "admissionJobPrep", []string{"bcs", "admission"}
	eq(t, "admission exams", CheckBook(d, "admissionJobPrep"), nil)
}

func TestEditionRules(t *testing.T) {
	e := pb
	e.PriceBdt = 0
	eq(t, "price", edition(e), set(PriceNotPositive))
	e = pb
	e.ListPriceBdt = num(300)
	eq(t, "list price equal", edition(e), set(ListPriceTooLow))
	e.ListPriceBdt = num(350)
	eq(t, "list price above", edition(e), nil)
	e = pb
	e.Stock = -1
	eq(t, "stock", edition(e), set(StockNegative))
	e = pb
	e.ISBN = str("9789840001775")
	eq(t, "bad check digit", edition(e), set(IsbnInvalid))
}

func TestIsbn10IsStoredAsIsbn13(t *testing.T) {
	e := pb
	e.ISBN = str("0-306-40615-2")
	if got := Tidy(e).ISBN; got == nil || *got != "9780306406157" {
		t.Errorf("tidy: %v", got)
	}
	e.ISBN = str("0306406152")
	eq(t, "isbn-10", edition(e), nil)
}

func TestTakenIsbns(t *testing.T) {
	e := pb
	e.ISBN = str("978-984-0001-774")
	eq(t, "another book has it", edition(e), set(IsbnTaken))
	sibling := pb
	sibling.Format, sibling.ISBN = "hardcover", str("9780306406157")
	e.ISBN = str("0306406152")
	eq(t, "a sibling has it", edition(e, sibling), set(IsbnTaken))
}

func TestOneEditionPerFormatAndLanguage(t *testing.T) {
	eq(t, "duplicate", edition(pb, pb), set(EditionTaken))
	english := pb
	english.Language = "english"
	eq(t, "other language", edition(english, pb), nil)
}

func TestEbookGetsStock999NoIsbnNoPreorder(t *testing.T) {
	e := pb
	e.Format, e.ISBN, e.IsPreorder = "ebook", str("123"), true
	got := Tidy(e)
	if got.Stock != 999 || got.ISBN != nil || got.IsPreorder {
		t.Errorf("ebook: %+v", got)
	}
	eq(t, "ebook edition", edition(got), nil)
}

func TestWholeBookChecksEveryEditionAgainstTheOthers(t *testing.T) {
	d := draft()
	d.Editions = []EditionDraft{pb, pb}
	eq(t, "two the same", CheckWholeBook(d, "literature", nil), set(EditionTaken))
}

// Ported from test/catalog_admin_record_rules_test.dart.
func TestRecordRules(t *testing.T) {
	eq(t, "author needs a name", CheckRecord("author", "", "লেখক"), set(NameBlank))
	eq(t, "author english only", CheckRecord("author", "Poetry", ""), nil)
	eq(t, "category needs both", CheckRecord("category", "Poetry", ""), set(NameBnBlank))
}

func TestBannerRules(t *testing.T) {
	eq(t, "ok", CheckBanner("Eid", "ঈদ", "eid"), nil)
	eq(t, "blank", CheckBanner("Eid", " ", ""), set(BannerTitleBlank, BannerTargetBlank))
}

func TestListRules(t *testing.T) {
	ok := ListDraft{TitleEn: "A", TitleBn: "এ", BookIDs: []string{"b1"}}
	eq(t, "ok", CheckList(ok), nil)
	eq(t, "no books", CheckList(ListDraft{TitleEn: "A", TitleBn: "এ"}), set(ListNoBooks))
	eq(t, "titles", CheckList(ListDraft{TitleEn: "A", BookIDs: []string{"b1"}}), set(ListTitleBlank))
	eq(t, "duplicate", CheckList(ListDraft{TitleEn: "A", TitleBn: "এ", BookIDs: []string{"b1", "b1"}}), set(ListDuplicateBook))
	long := ok
	long.NoteEn = strings.Repeat("x", 301)
	eq(t, "note", CheckList(long), set(ListNoteTooLong))
	long.NoteEn = strings.Repeat("x", 300)
	eq(t, "note at the limit", CheckList(long), nil)
}
