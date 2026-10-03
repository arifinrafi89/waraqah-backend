package catalogadmin

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

var (
	sections   = []string{"academic", "religious", "literature", "admissionJobPrep", "schoolCollege", "nonFiction", "skillsTech", "children"}
	languages  = []string{"bangla", "english", "arabic"}
	formats    = []string{"paperback", "hardcover", "ebook"}
	formatCode = map[string]string{"paperback": "pb", "hardcover": "hc", "ebook": "eb"}
	langCode   = map[string]string{"english": "en", "bangla": "bn", "arabic": "ar"}
)

func oneOf(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// validEnums reports whether the section, languages, formats and exams of a draft are names the app knows.
func validEnums(d BookDraft) bool {
	if !oneOf(sections, d.Section) || !oneOf(languages, d.OriginalLanguage) {
		return false
	}
	for _, e := range d.Editions {
		if !oneOf(formats, e.Format) || !oneOf(languages, e.Language) {
			return false
		}
	}
	for _, x := range d.Exams {
		if !oneOf([]string{"ssc", "hsc", "admission", "bcs"}, x) {
			return false
		}
	}
	return true
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

// saveBook adds or updates a Book from a draft inside tx. It checks the draft the way
// CatalogAdminFakeStore.saveBook does and answers the saved Book, or a Refusal.
func (s *Service) saveBook(ctx context.Context, q *sqlc.Queries, d BookDraft) (catalog.Book, error) {
	if !validEnums(d) {
		return catalog.Book{}, ErrBookInvalid
	}
	ids, err := q.BookIDs(ctx)
	if err != nil {
		return catalog.Book{}, err
	}
	var old *sqlc.Book
	id := ""
	if d.ID != nil {
		row, err := q.GetBookForUpdate(ctx, *d.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return catalog.Book{}, ErrBookUnknown
		}
		if err != nil {
			return catalog.Book{}, err
		}
		old, id = &row, row.ID
	} else {
		id = uniqueID("bk", d.Title, ids)
	}
	author, err := q.GetAuthor(ctx, d.AuthorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return catalog.Book{}, ErrBookInvalid
	}
	if err != nil {
		return catalog.Book{}, err
	}
	categorySection := ""
	if cat, err := q.GetCategory(ctx, d.CategoryID); err == nil {
		categorySection = cat.Section
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return catalog.Book{}, err
	}
	if ok, err := q.PublisherExists(ctx, d.PublisherID); err != nil {
		return catalog.Book{}, err
	} else if !ok {
		return catalog.Book{}, ErrBookInvalid
	}
	isbns, err := q.OtherBooksIsbns(ctx, id)
	if err != nil {
		return catalog.Book{}, err
	}
	taken := map[string]bool{}
	for _, i := range isbns {
		taken[i.String] = true
	}
	if len(CheckWholeBook(d, categorySection, taken)) > 0 {
		return catalog.Book{}, ErrBookInvalid
	}
	return s.writeBook(ctx, q, d, id, old, author)
}

func (s *Service) writeBook(ctx context.Context, q *sqlc.Queries, d BookDraft, id string, old *sqlc.Book, author sqlc.Author) (catalog.Book, error) {
	title := strings.TrimSpace(d.Title)
	short := pgtype.Text{}
	if old != nil && old.Title == title {
		short = old.ShortTitle
	}
	classes := make([]int32, 0, len(d.Classes))
	for _, c := range d.Classes {
		classes = append(classes, int32(c))
	}
	exams := d.Exams
	if exams == nil {
		exams = []string{}
	}
	titleBn := text(strings.TrimSpace(d.TitleBn))
	subject := text(d.SubjectID)
	if old == nil {
		err := q.InsertBook(ctx, sqlc.InsertBookParams{ID: id, Title: title, TitleBn: titleBn, ShortTitle: short,
			Author: author.Name, AuthorID: author.ID, PublisherID: d.PublisherID, CategoryID: d.CategoryID, Section: d.Section,
			OriginalLanguage: d.OriginalLanguage, AddedAt: s.Clock.Now(), Rating: 0, Tags: []string{}, CoverSeed: int32(d.CoverSeed),
			Hidden: false, Classes: classes, Exams: exams, SubjectID: subject})
		if err != nil {
			return catalog.Book{}, err
		}
	} else {
		err := q.UpdateBook(ctx, sqlc.UpdateBookParams{ID: id, Title: title, TitleBn: titleBn, ShortTitle: short,
			Author: author.Name, AuthorID: author.ID, PublisherID: d.PublisherID, CategoryID: d.CategoryID, Section: d.Section,
			OriginalLanguage: d.OriginalLanguage, CoverSeed: int32(d.CoverSeed), Classes: classes, Exams: exams, SubjectID: subject})
		if err != nil {
			return catalog.Book{}, err
		}
	}
	if err := s.writeEditions(ctx, q, id, d.Editions); err != nil {
		return catalog.Book{}, err
	}
	return s.loadBook(ctx, q, id)
}

// writeEditions stores the Editions with ids like bk-atomic-pb-en, drops the ones no longer in the
// draft, and remembers a cheaper earlier price for the 30-day low badge.
func (s *Service) writeEditions(ctx context.Context, q *sqlc.Queries, bookID string, drafts []EditionDraft) error {
	before, err := q.ListEditionsOfBook(ctx, bookID)
	if err != nil {
		return err
	}
	oldPrice := map[string]int{}
	for _, e := range before {
		oldPrice[e.ID] = int(e.PriceBdt)
	}
	keep := make([]string, 0, len(drafts))
	for _, raw := range drafts {
		e := Tidy(raw)
		id := bookID + "-" + formatCode[e.Format] + "-" + langCode[e.Language]
		keep = append(keep, id)
		list := pgtype.Int4{}
		if e.ListPriceBdt != nil {
			list = pgtype.Int4{Int32: int32(*e.ListPriceBdt), Valid: true}
		}
		isbnText := pgtype.Text{}
		if e.ISBN != nil {
			isbnText = pgtype.Text{String: *e.ISBN, Valid: true}
		}
		err := q.UpsertEdition(ctx, sqlc.UpsertEditionParams{ID: id, BookID: bookID, Format: e.Format, Language: e.Language,
			PriceBdt: int32(e.PriceBdt), ListPriceBdt: list, Stock: int32(e.Stock), IsPreorder: e.IsPreorder, Isbn: isbnText})
		if err != nil {
			return err
		}
		if was, ok := oldPrice[id]; ok && was != e.PriceBdt {
			if err := s.notePriceChange(ctx, q, id, was, e.PriceBdt); err != nil {
				return err
			}
		}
	}
	return q.DeleteEditionsExcept(ctx, sqlc.DeleteEditionsExceptParams{BookID: bookID, Column2: keep})
}

// notePriceChange keeps the lowest price of the last 30 days: the cheaper of the old and the new
// price, unless a lower one is already remembered.
func (s *Service) notePriceChange(ctx context.Context, q *sqlc.Queries, editionID string, was, now int) error {
	lowest := min(was, now)
	cur, err := q.GetPriceLow(ctx, editionID)
	if err == nil && cur.Since.After(s.Clock.Now().AddDate(0, 0, -30)) && int(cur.LowBdt) <= lowest {
		return nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return q.UpsertPriceLow(ctx, sqlc.UpsertPriceLowParams{EditionID: editionID, LowBdt: int32(lowest), Since: s.Clock.Now()})
}

func (s *Service) loadBook(ctx context.Context, q *sqlc.Queries, id string) (catalog.Book, error) {
	row, err := q.GetBookForUpdate(ctx, id)
	if err != nil {
		return catalog.Book{}, err
	}
	editions, err := q.ListEditionsOfBook(ctx, id)
	if err != nil {
		return catalog.Book{}, err
	}
	return catalog.BookFromRow(row, editions, s.Loc), nil
}

// SaveBook is CatalogAdminFakeApi.saveBook.
func (s *Service) SaveBook(ctx context.Context, d BookDraft) (catalog.Book, error) {
	var out catalog.Book
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		var err error
		out, err = s.saveBook(ctx, q, d)
		return err
	})
	if err == nil {
		s.changed(ctx, true)
	}
	return out, err
}

// HideBook is CatalogAdminFakeApi.hideBook.
func (s *Service) HideBook(ctx context.Context, id string, hidden bool) (catalog.Book, error) {
	var out catalog.Book
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		n, err := q.SetBookHidden(ctx, sqlc.SetBookHiddenParams{ID: id, Hidden: hidden})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrBookUnknown
		}
		out, err = s.loadBook(ctx, q, id)
		return err
	})
	if err == nil {
		s.changed(ctx, false)
	}
	return out, err
}
