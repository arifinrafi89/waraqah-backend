package catalog

import (
	"regexp"
	"sort"
	"strings"
)

var (
	isbnDigits = regexp.MustCompile(`^\d{13}$`)
	spaceDash  = regexp.MustCompile(`[\s-]`)
)

func anyContains(names []string, q string) bool {
	for _, n := range names {
		if strings.Contains(n, q) {
			return true
		}
	}
	return false
}

func anyHasPrefix(names []string, q string) bool {
	for _, n := range names {
		if strings.HasPrefix(n, q) {
			return true
		}
	}
	return false
}

// startsWord: a word of one of the spellings starts with the sound key.
func startsWord(keys []string, sound string) bool {
	for _, k := range keys {
		if strings.Contains(" "+k, " "+sound) {
			return true
		}
	}
	return false
}

// rank is BookSearchMatch._rank: 0 title starts with the query, 1 title contains it, 2 Author,
// 3 Publisher, 4 a full ISBN-13; then the same by sound (title 1, Author 2, Publisher 3).
// ok is false when the book does not match.
func (s *Snapshot) rank(i int, query, sound string) (int, bool) {
	ix := s.index[i]
	if anyHasPrefix(ix.titles, query) {
		return 0, true
	}
	if anyContains(ix.titles, query) {
		return 1, true
	}
	if anyContains(ix.authors, query) {
		return 2, true
	}
	if anyContains(ix.publishers, query) {
		return 3, true
	}
	isbn := spaceDash.ReplaceAllString(query, "")
	if isbnDigits.MatchString(isbn) {
		for _, e := range s.Books[i].Editions {
			if e.ISBN != nil && *e.ISBN == isbn {
				return 4, true
			}
		}
	}
	// Keys under 3 letters ("the" is "t") would match half the catalog.
	if len(sound) < 3 {
		return 0, false
	}
	switch {
	case startsWord(ix.soundTitle, sound):
		return 1, true
	case startsWord(ix.soundAuth, sound):
		return 2, true
	case startsWord(ix.soundPub, sound):
		return 3, true
	}
	return 0, false
}

// Rank returns the books matching query (trimmed, lower case), best match first. Ties go to the
// higher rating, then the newer addedAt.
func (s *Snapshot) Rank(books []Book, query string) []Book {
	sound := PhoneticKey(query)
	type hit struct {
		rank int
		book Book
	}
	var hits []hit
	for _, b := range books {
		if r, ok := s.rank(s.byID[b.ID], query, sound); ok {
			hits = append(hits, hit{r, b})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].rank != hits[b].rank {
			return hits[a].rank < hits[b].rank
		}
		if hits[a].book.Rating != hits[b].book.Rating {
			return hits[a].book.Rating > hits[b].book.Rating
		}
		return hits[a].book.AddedAt.After(hits[b].book.AddedAt)
	})
	out := make([]Book, len(hits))
	for i, h := range hits {
		out[i] = h.book
	}
	return out
}

// Filters are the /books query parameters (BookFakeApi._books, BookEditionFilter).
type Filters struct {
	Query         string
	Category      string
	Section       string
	Author        string
	Publisher     string
	Class         *int
	Exam          string
	Subject       string
	Sort          string
	MinPrice      *int
	MaxPrice      *int
	Formats       map[string]bool
	Languages     map[string]bool
	MinRating     *float64
	InStock       bool
	IncludeHidden bool
}

func (f Filters) inScope(b Book) bool {
	return (f.IncludeHidden || !b.Hidden) &&
		(f.Category == "" || b.CategoryID == f.Category) &&
		(f.Section == "" || b.Section == f.Section) &&
		(f.Author == "" || b.AuthorID == f.Author) &&
		(f.Publisher == "" || b.PublisherID == f.Publisher)
}

// editionFits: format, language, price and inStock must all hold for ONE edition. The minimum
// price is included, the maximum excluded; an in-stock edition has stock above zero.
func (f Filters) editionFits(e Edition) bool {
	return (len(f.Formats) == 0 || f.Formats[e.Format]) &&
		(len(f.Languages) == 0 || f.Languages[e.Language]) &&
		(f.MinPrice == nil || e.PriceBdt >= *f.MinPrice) &&
		(f.MaxPrice == nil || e.PriceBdt < *f.MaxPrice) &&
		(!f.InStock || e.Stock > 0)
}

func (f Filters) passes(b Book) bool {
	if f.MinRating != nil && b.Rating < *f.MinRating {
		return false
	}
	if f.Class != nil && !containsInt(b.Classes, *f.Class) {
		return false
	}
	if f.Exam != "" && !containsStr(b.Exams, f.Exam) {
		return false
	}
	if f.Subject != "" && (b.SubjectID == nil || *b.SubjectID != f.Subject) {
		return false
	}
	for _, e := range b.Editions {
		if f.editionFits(e) {
			return true
		}
	}
	return false
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// Search answers /books: scope and filters, then the ranked matches of the query (or the
// storefront order without one), then the sort. sold is the copies sold per Edition in the last
// 30 days, needed only for the bestselling sort.
func (s *Snapshot) Search(f Filters, sold map[string]int) []Book {
	var matches []Book
	for _, b := range s.Books {
		if f.passes(b) && f.inScope(b) {
			matches = append(matches, b)
		}
	}
	query := strings.ToLower(strings.TrimSpace(f.Query))
	if query != "" {
		matches = s.Rank(matches, query)
	}
	return SortBooks(matches, f.Sort, sold)
}

// SortBooks orders books by a SearchSort name; relevance, unknown or empty keeps the order given.
func SortBooks(books []Book, sortName string, sold map[string]int) []Book {
	out := append([]Book{}, books...)
	less := map[string]func(a, b Book) bool{
		"priceLow":  func(a, b Book) bool { return a.FromPriceBdt() < b.FromPriceBdt() },
		"priceHigh": func(a, b Book) bool { return a.FromPriceBdt() > b.FromPriceBdt() },
		"newest":    func(a, b Book) bool { return a.AddedAt.After(b.AddedAt) },
		"bestselling": func(a, b Book) bool {
			return soldCopies(a, sold) > soldCopies(b, sold)
		},
	}[sortName]
	if less != nil {
		sort.SliceStable(out, func(i, j int) bool { return less(out[i], out[j]) })
	}
	return out
}

func soldCopies(b Book, sold map[string]int) int {
	total := 0
	for _, e := range b.Editions {
		total += sold[e.ID]
	}
	return total
}
