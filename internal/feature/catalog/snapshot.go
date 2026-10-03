package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Snapshot is the whole catalog in memory. The search, suggestion and did-you-mean rules are ports
// of Dart code that works on every book, so they run here; the catalog is a few hundred books.
type Snapshot struct {
	Books      []Book // in storefront order
	Categories []Category
	Authors    []Author
	Publishers []Publisher
	Subjects   []Subject
	Series     []Series

	byID       map[string]int
	authors    map[string]Author
	publishers map[string]Publisher
	index      []bookIndex // parallel to Books
}

// bookIndex holds what search needs per book, computed once: the spellings and their sound keys.
type bookIndex struct {
	titles     []string // title, Bangla title (lower case)
	soundTitle []string // phonetic keys of title, Bangla title and short title
	authors    []string
	soundAuth  []string
	publishers []string
	soundPub   []string
}

// Book returns a book by id (hidden ones too).
func (s *Snapshot) Book(id string) (Book, bool) {
	i, ok := s.byID[id]
	if !ok {
		return Book{}, false
	}
	return s.Books[i], true
}

// Visible lists the books on the storefront.
func (s *Snapshot) Visible() []Book {
	out := make([]Book, 0, len(s.Books))
	for _, b := range s.Books {
		if !b.Hidden {
			out = append(out, b)
		}
	}
	return out
}

// Store loads the snapshot from the database and keeps it until something changes the catalog.
type Store struct {
	DB  *db.DB
	Loc *time.Location

	mu   sync.Mutex
	snap *Snapshot
}

// Invalidate drops the cached snapshot; call it after a change to the catalog tables.
func (st *Store) Invalidate() {
	st.mu.Lock()
	st.snap = nil
	st.mu.Unlock()
}

// Snapshot returns the cached catalog, loading it when needed.
func (st *Store) Snapshot(ctx context.Context) (*Snapshot, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.snap != nil {
		return st.snap, nil
	}
	s, err := st.load(ctx)
	if err != nil {
		return nil, err
	}
	st.snap = s
	return s, nil
}

func (st *Store) load(ctx context.Context) (*Snapshot, error) {
	q := st.DB.Q()
	books, err := q.ListBooks(ctx)
	if err != nil {
		return nil, err
	}
	editions, err := q.ListEditions(ctx)
	if err != nil {
		return nil, err
	}
	s := &Snapshot{byID: map[string]int{}, authors: map[string]Author{}, publishers: map[string]Publisher{}}
	byBook := map[string][]Edition{}
	for _, e := range editions {
		byBook[e.BookID] = append(byBook[e.BookID], toEdition(e))
	}
	for _, b := range books {
		s.byID[b.ID] = len(s.Books)
		s.Books = append(s.Books, toBook(b, byBook[b.ID], st.Loc))
	}
	if err := st.loadRecords(ctx, s); err != nil {
		return nil, err
	}
	s.buildIndex()
	return s, nil
}

func (st *Store) loadRecords(ctx context.Context, s *Snapshot) error {
	q := st.DB.Q()
	cats, err := q.ListCategories(ctx)
	if err != nil {
		return err
	}
	for _, c := range cats {
		s.Categories = append(s.Categories, Category{ID: c.ID, Section: c.Section, NameEn: c.NameEn, NameBn: c.NameBn})
	}
	auths, err := q.ListAuthors(ctx)
	if err != nil {
		return err
	}
	for _, a := range auths {
		au := Author{ID: a.ID, Name: a.Name, NameBn: textPtr(a.NameBn.String, a.NameBn.Valid), Bio: textPtr(a.Bio.String, a.Bio.Valid)}
		s.Authors = append(s.Authors, au)
		s.authors[a.ID] = au
	}
	pubs, err := q.ListPublishers(ctx)
	if err != nil {
		return err
	}
	for _, p := range pubs {
		pu := Publisher{ID: p.ID, Name: p.Name, NameBn: textPtr(p.NameBn.String, p.NameBn.Valid)}
		s.Publishers = append(s.Publishers, pu)
		s.publishers[p.ID] = pu
	}
	subs, err := q.ListSubjects(ctx)
	if err != nil {
		return err
	}
	for _, x := range subs {
		s.Subjects = append(s.Subjects, Subject{ID: x.ID, NameEn: x.NameEn, NameBn: x.NameBn})
	}
	series, err := q.ListSeries(ctx)
	if err != nil {
		return err
	}
	for _, x := range series {
		var entries []SeriesEntry
		if err := json.Unmarshal(x.Entries, &entries); err != nil {
			return err
		}
		s.Series = append(s.Series, Series{ID: x.ID, Name: x.Name, Entries: entries})
	}
	return nil
}

// NewSnapshot builds a snapshot from loaded records (tests and the store use it).
func NewSnapshot(books []Book, authors []Author, publishers []Publisher) *Snapshot {
	s := &Snapshot{Books: books, Authors: authors, Publishers: publishers,
		byID: map[string]int{}, authors: map[string]Author{}, publishers: map[string]Publisher{}}
	for i, b := range books {
		s.byID[b.ID] = i
	}
	for _, a := range authors {
		s.authors[a.ID] = a
	}
	for _, p := range publishers {
		s.publishers[p.ID] = p
	}
	s.buildIndex()
	return s
}

func (s *Snapshot) buildIndex() {
	s.index = make([]bookIndex, len(s.Books))
	lower := func(parts ...*string) []string {
		var out []string
		for _, p := range parts {
			if p != nil {
				out = append(out, strings.ToLower(*p))
			}
		}
		return out
	}
	keys := func(parts ...*string) []string {
		var out []string
		for _, p := range parts {
			if p != nil {
				out = append(out, PhoneticKey(*p))
			}
		}
		return out
	}
	for i, b := range s.Books {
		a := s.authors[b.AuthorID]
		p := s.publishers[b.PublisherID]
		author := b.Author
		s.index[i] = bookIndex{
			titles:     lower(&b.Title, b.TitleBn),
			soundTitle: keys(&b.Title, b.TitleBn, b.ShortTitle),
			authors:    lower(&author, strPtrIf(a.ID != "", a.Name), a.NameBn),
			soundAuth:  keys(&author, strPtrIf(a.ID != "", a.Name), a.NameBn),
			publishers: lower(strPtrIf(p.ID != "", p.Name), p.NameBn),
			soundPub:   keys(strPtrIf(p.ID != "", p.Name), p.NameBn),
		}
	}
}

func strPtrIf(ok bool, s string) *string {
	if !ok {
		return nil
	}
	return &s
}

func textPtr(s string, valid bool) *string {
	if !valid {
		return nil
	}
	return &s
}

func toEdition(e sqlc.Edition) Edition {
	out := Edition{ID: e.ID, Format: e.Format, Language: e.Language, PriceBdt: int(e.PriceBdt), Stock: int(e.Stock), IsPreorder: e.IsPreorder}
	if e.ListPriceBdt.Valid {
		v := int(e.ListPriceBdt.Int32)
		out.ListPriceBdt = &v
	}
	if e.Isbn.Valid {
		out.ISBN = &e.Isbn.String
	}
	return out
}

func toBook(b sqlc.Book, editions []Edition, loc *time.Location) Book {
	classes := make([]int, 0, len(b.Classes))
	for _, c := range b.Classes {
		classes = append(classes, int(c))
	}
	return Book{
		ID: b.ID, Title: b.Title, Author: b.Author, CategoryID: b.CategoryID, AuthorID: b.AuthorID,
		PublisherID: b.PublisherID, Section: b.Section, OriginalLanguage: b.OriginalLanguage,
		Editions: editions, AddedAt: b.AddedAt.In(loc).Truncate(time.Second), Rating: b.Rating,
		Tags: nonNilStrings(b.Tags), CoverSeed: int(b.CoverSeed), ShortTitle: textPtr(b.ShortTitle.String, b.ShortTitle.Valid),
		TitleBn: textPtr(b.TitleBn.String, b.TitleBn.Valid), Hidden: b.Hidden, Classes: classes,
		Exams: nonNilStrings(b.Exams), SubjectID: textPtr(b.SubjectID.String, b.SubjectID.Valid),
	}
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
