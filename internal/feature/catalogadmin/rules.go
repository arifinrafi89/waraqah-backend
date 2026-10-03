// Package catalogadmin serves Admin → Catalog: Staff add and edit Books, Editions, stock,
// Categories, Authors, Publishers, Banners, Collections and Booklists, force the Home Season, and
// use the ISBN lookup, low-stock list and CSV import tools.
package catalogadmin

import (
	"slices"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/isbn"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// RuleError is one reason Staff change can not be saved (RuleError in the app).
type RuleError string

// The rule errors, named as the Dart enum values.
const (
	TitleBlank           RuleError = "titleBlank"
	AuthorMissing        RuleError = "authorMissing"
	PublisherMissing     RuleError = "publisherMissing"
	CategoryMissing      RuleError = "categoryMissing"
	CategoryWrongSection RuleError = "categoryWrongSection"
	ClassNotAllowed      RuleError = "classNotAllowed"
	ExamNotAllowed       RuleError = "examNotAllowed"
	NoEditions           RuleError = "noEditions"
	PriceNotPositive     RuleError = "priceNotPositive"
	ListPriceTooLow      RuleError = "listPriceTooLow"
	StockNegative        RuleError = "stockNegative"
	IsbnInvalid          RuleError = "isbnInvalid"
	IsbnTaken            RuleError = "isbnTaken"
	EditionTaken         RuleError = "editionTaken"
	NameBlank            RuleError = "nameBlank"
	NameBnBlank          RuleError = "nameBnBlank"
	BannerTitleBlank     RuleError = "bannerTitleBlank"
	BannerTargetBlank    RuleError = "bannerTargetBlank"
	ListTitleBlank       RuleError = "listTitleBlank"
	ListNoBooks          RuleError = "listNoBooks"
	ListDuplicateBook    RuleError = "listDuplicateBook"
	ListNoteTooLong      RuleError = "listNoteTooLong"
)

// Port of CatalogAdminRules and ListRules.
const (
	// EbookStock: eBooks never run out.
	EbookStock = 999
	// LowStock: a printed Edition this low needs more copies.
	LowStock = 5
	// NoteMax is the most characters in each language note of a list.
	NoteMax = 300
)

// Errors is a set of rule errors.
type Errors map[RuleError]bool

func (e Errors) add(cond bool, r RuleError) {
	if cond {
		e[r] = true
	}
}

func (e Errors) merge(o Errors) {
	for k := range o {
		e[k] = true
	}
}

// Section helpers (SectionAcademics in the app): which classes, exams and subjects a Section offers.
func classLevels(section string) []int {
	if section == "schoolCollege" {
		return []int{6, 7, 8, 9, 10, 11, 12}
	}
	return nil
}

func allowedExams(section string) []string {
	switch section {
	case "schoolCollege":
		return []string{"ssc", "hsc"}
	case "admissionJobPrep":
		return []string{"admission", "bcs"}
	}
	return nil
}

// EditionDraft is an Edition as Staff fill it in.
type EditionDraft struct {
	ID           string  `json:"id"`
	Format       string  `json:"format"`
	Language     string  `json:"language"`
	PriceBdt     int     `json:"priceBdt"`
	Stock        int     `json:"stock"`
	ListPriceBdt *int    `json:"listPriceBdt"`
	IsPreorder   bool    `json:"isPreorder"`
	ISBN         *string `json:"isbn"`
}

// BookDraft is a Book as Staff fill it in. No ID means a new Book.
type BookDraft struct {
	ID               *string        `json:"id"`
	Title            string         `json:"title"`
	TitleBn          string         `json:"titleBn"`
	AuthorID         string         `json:"authorId"`
	PublisherID      string         `json:"publisherId"`
	Section          string         `json:"section"`
	CategoryID       string         `json:"categoryId"`
	OriginalLanguage string         `json:"originalLanguage"`
	CoverSeed        int            `json:"coverSeed"`
	Editions         []EditionDraft `json:"editions"`
	Classes          []int          `json:"classes"`
	Exams            []string       `json:"exams"`
	SubjectID        string         `json:"subjectId"`
}

// CheckBook is CatalogAdminRules.book. categorySection is the Section of the picked Category
// ("" when unknown). Classes and Exams must be ones the Section offers.
func CheckBook(d BookDraft, categorySection string) Errors {
	e := Errors{}
	e.add(strings.TrimSpace(d.Title) == "", TitleBlank)
	e.add(d.AuthorID == "", AuthorMissing)
	e.add(d.PublisherID == "", PublisherMissing)
	if d.CategoryID == "" {
		e[CategoryMissing] = true
	} else if categorySection != d.Section {
		e[CategoryWrongSection] = true
	}
	e.add(len(d.Editions) == 0, NoEditions)
	for _, c := range d.Classes {
		if !slices.Contains(classLevels(d.Section), c) {
			e[ClassNotAllowed] = true
		}
	}
	for _, x := range d.Exams {
		if !slices.Contains(allowedExams(d.Section), x) {
			e[ExamNotAllowed] = true
		}
	}
	return e
}

// Tidy is CatalogAdminRules.tidy: an eBook has stock 999, no ISBN and no pre-order; a valid
// ISBN becomes its ISBN-13.
func Tidy(e EditionDraft) EditionDraft {
	if e.Format == "ebook" {
		e.Stock, e.ISBN, e.IsPreorder = EbookStock, nil, false
		return e
	}
	raw := ""
	if e.ISBN != nil {
		raw = strings.TrimSpace(*e.ISBN)
	}
	if raw == "" {
		e.ISBN = nil
		return e
	}
	if n, ok := isbn.Normalize(raw); ok {
		raw = n
	}
	e.ISBN = &raw
	return e
}

// CheckEdition is CatalogAdminRules.edition. siblings are the other Editions of the Book;
// takenIsbns are the ISBNs of every other Book Editions.
func CheckEdition(edition EditionDraft, siblings []EditionDraft, takenIsbns map[string]bool) Errors {
	e := Tidy(edition)
	out := Errors{}
	out.add(e.PriceBdt <= 0, PriceNotPositive)
	out.add(e.ListPriceBdt != nil && *e.ListPriceBdt <= e.PriceBdt, ListPriceTooLow)
	out.add(e.Stock < 0, StockNegative)
	if e.ISBN != nil {
		_, valid := isbn.Normalize(*e.ISBN)
		out.add(!valid, IsbnInvalid)
		taken := takenIsbns[*e.ISBN]
		for _, s := range siblings {
			if t := Tidy(s); t.ISBN != nil && *t.ISBN == *e.ISBN {
				taken = true
			}
		}
		out.add(taken, IsbnTaken)
	}
	for _, s := range siblings {
		if s.Format == e.Format && s.Language == e.Language {
			out[EditionTaken] = true
		}
	}
	return out
}

// CheckWholeBook is CatalogAdminRules.wholeBook: the details and each Edition against the others.
func CheckWholeBook(d BookDraft, categorySection string, takenIsbns map[string]bool) Errors {
	out := CheckBook(d, categorySection)
	for i, e := range d.Editions {
		siblings := append(append([]EditionDraft{}, d.Editions[:i]...), d.Editions[i+1:]...)
		out.merge(CheckEdition(e, siblings, takenIsbns))
	}
	return out
}

// CheckRecord is CatalogAdminRules.record: a Category needs both names; an Author or Publisher an
// English one.
func CheckRecord(kind, name, nameBn string) Errors {
	e := Errors{}
	e.add(strings.TrimSpace(name) == "", NameBlank)
	e.add(kind == "category" && strings.TrimSpace(nameBn) == "", NameBnBlank)
	return e
}

// CheckBanner is CatalogAdminRules.banner: both titles and a target.
func CheckBanner(titleEn, titleBn, targetValue string) Errors {
	e := Errors{}
	e.add(strings.TrimSpace(titleEn) == "" || strings.TrimSpace(titleBn) == "", BannerTitleBlank)
	e.add(strings.TrimSpace(targetValue) == "", BannerTargetBlank)
	return e
}

// ListDraft is a Collection, or with a Kind a Staff Booklist, as Staff fill it in.
type ListDraft struct {
	ID       *string  `json:"id"`
	TitleEn  string   `json:"titleEn"`
	TitleBn  string   `json:"titleBn"`
	NoteEn   string   `json:"noteEn"`
	NoteBn   string   `json:"noteBn"`
	Section  *string  `json:"section"`
	ExpertID *string  `json:"expertId"`
	Kind     *string  `json:"kind"`
	BookIDs  []string `json:"bookIds"`
}

// CheckList is ListRules.check.
func CheckList(d ListDraft) Errors {
	e := Errors{}
	e.add(strings.TrimSpace(d.TitleEn) == "" || strings.TrimSpace(d.TitleBn) == "", ListTitleBlank)
	e.add(len(d.BookIDs) == 0, ListNoBooks)
	seen := map[string]bool{}
	for _, id := range d.BookIDs {
		if seen[id] {
			e[ListDuplicateBook] = true
		}
		seen[id] = true
	}
	e.add(textutil.TrimLen(d.NoteEn) > NoteMax || textutil.TrimLen(d.NoteBn) > NoteMax, ListNoteTooLong)
	return e
}
