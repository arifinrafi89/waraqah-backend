// Package catalog serves the storefront: books and their editions, search, records (categories,
// authors, publishers, subjects), series, collections, experts and booklists, and the questions
// readers ask about a book.
package catalog

import "time"

// Edition is EditionModel: one buyable version of a Book (a format in a language).
type Edition struct {
	ID           string  `json:"id"`
	Format       string  `json:"format"`
	Language     string  `json:"language"`
	PriceBdt     int     `json:"priceBdt"`
	Stock        int     `json:"stock"`
	ListPriceBdt *int    `json:"listPriceBdt"`
	IsPreorder   bool    `json:"isPreorder"`
	ISBN         *string `json:"isbn"`
}

// IsOrderable: can be ordered now, in stock or a pre-order.
func (e Edition) IsOrderable() bool { return e.Stock > 0 || e.IsPreorder }

// Book is the Book model every feature speaks (core/models/book.dart).
type Book struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Author           string    `json:"author"`
	CategoryID       string    `json:"categoryId"`
	AuthorID         string    `json:"authorId"`
	PublisherID      string    `json:"publisherId"`
	Section          string    `json:"section"`
	OriginalLanguage string    `json:"originalLanguage"`
	Editions         []Edition `json:"editions"`
	AddedAt          time.Time `json:"addedAt"`
	Rating           float64   `json:"rating"`
	Tags             []string  `json:"tags"`
	CoverSeed        int       `json:"coverSeed"`
	ShortTitle       *string   `json:"shortTitle"`
	TitleBn          *string   `json:"titleBn"`
	Hidden           bool      `json:"hidden"`
	Classes          []int     `json:"classes"`
	Exams            []string  `json:"exams"`
	SubjectID        *string   `json:"subjectId"`
}

// FromEdition is the Edition a card leads with: the cheapest one that can be ordered now, or
// the cheapest overall when none can.
func (b Book) FromEdition() Edition {
	var orderable []Edition
	for _, e := range b.Editions {
		if e.IsOrderable() {
			orderable = append(orderable, e)
		}
	}
	if len(orderable) == 0 {
		orderable = b.Editions
	}
	best := orderable[0]
	for _, e := range orderable[1:] {
		if e.PriceBdt < best.PriceBdt {
			best = e
		}
	}
	return best
}

// ChosenEdition is the edition with that id; nothing chosen or an unknown id falls back to FromEdition.
func (b Book) ChosenEdition(id string) Edition {
	for _, e := range b.Editions {
		if e.ID == id {
			return e
		}
	}
	return b.FromEdition()
}

// FromPriceBdt is the price of FromEdition.
func (b Book) FromPriceBdt() int { return b.FromEdition().PriceBdt }

// Category is CategoryModel.
type Category struct {
	ID      string `json:"id"`
	Section string `json:"section"`
	NameEn  string `json:"nameEn"`
	NameBn  string `json:"nameBn"`
}

// Author is AuthorModel.
type Author struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	NameBn *string `json:"nameBn"`
	Bio    *string `json:"bio"`
}

// Publisher is PublisherModel.
type Publisher struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	NameBn *string `json:"nameBn"`
}

// Subject is SubjectModel.
type Subject struct {
	ID     string `json:"id"`
	NameEn string `json:"nameEn"`
	NameBn string `json:"nameBn"`
}

// Details is BookDetails: the summary and page count of a book.
type Details struct {
	BookID      string `json:"bookId"`
	Description string `json:"description"`
	Pages       int    `json:"pages"`
}

// ContentsEntry is one line of a table of contents.
type ContentsEntry struct {
	Title  string `json:"title"`
	IsPart bool   `json:"isPart"`
}

// LookInside is LookInsideModel.
type LookInside struct {
	Contents    []ContentsEntry `json:"contents"`
	SamplePages []string        `json:"samplePages"`
}

// SeriesEntry is one title of a series; BookID is set when Waraqah sells it.
type SeriesEntry struct {
	Position  int     `json:"position"`
	Title     string  `json:"title"`
	BookID    *string `json:"bookId"`
	CoverSeed int     `json:"coverSeed"`
}

// Series is BookSeriesModel.
type Series struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Entries []SeriesEntry `json:"entries"`
}

// Expert is ExpertModel.
type Expert struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	NameBn       string `json:"nameBn"`
	CredentialEn string `json:"credentialEn"`
	CredentialBn string `json:"credentialBn"`
	Kind         string `json:"kind"`
	Verified     bool   `json:"verified"`
}

// Collection is a staff-picked collection with its books (hidden ones left out) and its expert.
type Collection struct {
	ID       string   `json:"id"`
	TitleEn  string   `json:"titleEn"`
	TitleBn  string   `json:"titleBn"`
	NoteEn   string   `json:"noteEn"`
	NoteBn   string   `json:"noteBn"`
	BookIDs  []string `json:"bookIds"`
	Section  *string  `json:"section"`
	ExpertID *string  `json:"expertId"`
	Expert   *Expert  `json:"expert"`
	Books    []Book   `json:"books"`
}

// ExpertPage is an expert with their collections.
type ExpertPage struct {
	Expert
	Collections []Collection `json:"collections"`
}

// Booklist is a staff list or a reader own list, with its books (hidden ones left out).
type Booklist struct {
	ID      string   `json:"id"`
	TitleEn string   `json:"titleEn"`
	TitleBn string   `json:"titleBn"`
	Kind    string   `json:"kind"`
	BookIDs []string `json:"bookIds"`
	NoteEn  *string  `json:"noteEn"`
	NoteBn  *string  `json:"noteBn"`
	IsMine  bool     `json:"isMine"`
	Books   []Book   `json:"books"`
}

// UsedCopy is a Certified Used copy offered for a book.
type UsedCopy struct {
	ID       string `json:"id"`
	PriceBdt int    `json:"priceBdt"`
	// Condition is a BookCondition name (likeNew, veryGood, good, acceptable).
	Condition string `json:"condition"`
}

// UsedOptions is the answer of /books/used-options.
type UsedOptions struct {
	CertifiedUsed  *UsedCopy `json:"certifiedUsed"`
	ResaleValueBdt *int      `json:"resaleValueBdt"`
}

// Answer is a reply to a question about a book.
type Answer struct {
	ID         string    `json:"id"`
	Text       string    `json:"text"`
	AuthorName string    `json:"authorName"`
	AnsweredAt time.Time `json:"answeredAt"`
	IsStaff    bool      `json:"isStaff"`
}

// Question is a question a reader asked about a book.
type Question struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	AskerName string    `json:"askerName"`
	AskedAt   time.Time `json:"askedAt"`
	Answers   []Answer  `json:"answers"`
}
