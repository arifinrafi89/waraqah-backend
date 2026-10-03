package seed

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// naiveTime is how the Dart exporter writes a DateTime: no offset, local to the machine that
// exported it (Dhaka for the team).
const naiveTime = "2006-01-02T15:04:05.999999"

func (r *Run) parseNaive(s string) (time.Time, error) {
	return time.ParseInLocation(naiveTime, s, r.Loc)
}

func opt(p *string) pgtype.Text {
	if p == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *p, Valid: true}
}

func optInt(p *int) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*p), Valid: true}
}

type seedEdition struct {
	ID           string  `json:"id"`
	Format       string  `json:"format"`
	Language     string  `json:"language"`
	PriceBdt     int     `json:"priceBdt"`
	Stock        int     `json:"stock"`
	ListPriceBdt *int    `json:"listPriceBdt"`
	IsPreorder   bool    `json:"isPreorder"`
	ISBN         *string `json:"isbn"`
}

type seedBook struct {
	ID               string        `json:"id"`
	Title            string        `json:"title"`
	Author           string        `json:"author"`
	CategoryID       string        `json:"categoryId"`
	AuthorID         string        `json:"authorId"`
	PublisherID      string        `json:"publisherId"`
	Section          string        `json:"section"`
	OriginalLanguage string        `json:"originalLanguage"`
	Editions         []seedEdition `json:"editions"`
	AddedAt          string        `json:"addedAt"`
	Rating           float64       `json:"rating"`
	Tags             []string      `json:"tags"`
	CoverSeed        int           `json:"coverSeed"`
	ShortTitle       *string       `json:"shortTitle"`
	TitleBn          *string       `json:"titleBn"`
	Hidden           bool          `json:"hidden"`
	Classes          []int         `json:"classes"`
	Exams            []string      `json:"exams"`
	SubjectID        *string       `json:"subjectId"`
}

// loadCatalogRecords loads categories, authors, publishers and subjects.
func loadCatalogRecords(ctx context.Context, r *Run) error {
	var cats []struct{ ID, Section, NameEn, NameBn string }
	var auths []struct {
		ID, Name    string
		NameBn, Bio *string
	}
	var pubs []struct {
		ID, Name string
		NameBn   *string
	}
	var subs []struct{ ID, NameEn, NameBn string }
	for name, v := range map[string]any{"categories.json": &cats, "authors.json": &auths, "publishers.json": &pubs, "subjects.json": &subs} {
		if err := r.read(name, v); err != nil {
			return err
		}
	}
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, c := range cats {
			if err := q.UpsertSeedCategory(ctx, sqlc.UpsertSeedCategoryParams{ID: c.ID, Section: c.Section, NameEn: c.NameEn, NameBn: c.NameBn}); err != nil {
				return err
			}
		}
		for _, a := range auths {
			if err := q.UpsertSeedAuthor(ctx, sqlc.UpsertSeedAuthorParams{ID: a.ID, Name: a.Name, NameBn: opt(a.NameBn), Bio: opt(a.Bio)}); err != nil {
				return err
			}
		}
		for _, p := range pubs {
			if err := q.UpsertSeedPublisher(ctx, sqlc.UpsertSeedPublisherParams{ID: p.ID, Name: p.Name, NameBn: opt(p.NameBn)}); err != nil {
				return err
			}
		}
		for _, s := range subs {
			if err := q.UpsertSeedSubject(ctx, sqlc.UpsertSeedSubjectParams{ID: s.ID, NameEn: s.NameEn, NameBn: s.NameBn}); err != nil {
				return err
			}
		}
		return nil
	})
}

// loadBooks loads every book with its editions, in storefront order, and the sales and price
// history that go with them.
func loadBooks(ctx context.Context, r *Run) error {
	var books []seedBook
	if err := r.read("books.json", &books); err != nil {
		return err
	}
	var sales map[string]int
	if err := r.read("sales_30d.json", &sales); err != nil {
		return err
	}
	var lows []struct {
		EditionID string `json:"editionId"`
		LowBdt    int    `json:"lowBdt"`
	}
	if err := r.read("price_lows.json", &lows); err != nil {
		return err
	}
	now := time.Now().In(r.Loc)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, r.Loc)
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, b := range books {
			added, err := r.parseNaive(b.AddedAt)
			if err != nil {
				return err
			}
			classes := make([]int32, 0, len(b.Classes))
			for _, c := range b.Classes {
				classes = append(classes, int32(c))
			}
			err = q.UpsertSeedBook(ctx, sqlc.UpsertSeedBookParams{ID: b.ID, Title: b.Title, TitleBn: opt(b.TitleBn),
				ShortTitle: opt(b.ShortTitle), Author: b.Author, AuthorID: b.AuthorID, PublisherID: b.PublisherID,
				CategoryID: b.CategoryID, Section: b.Section, OriginalLanguage: b.OriginalLanguage, AddedAt: added,
				Rating: b.Rating, Tags: nonNil(b.Tags), CoverSeed: int32(b.CoverSeed), Hidden: b.Hidden, Classes: classes,
				Exams: nonNil(b.Exams), SubjectID: opt(b.SubjectID)})
			if err != nil {
				return err
			}
			for _, e := range b.Editions {
				err := q.UpsertSeedEdition(ctx, sqlc.UpsertSeedEditionParams{ID: e.ID, BookID: b.ID, Format: e.Format,
					Language: e.Language, PriceBdt: int32(e.PriceBdt), ListPriceBdt: optInt(e.ListPriceBdt), Stock: int32(e.Stock),
					IsPreorder: e.IsPreorder, Isbn: opt(e.ISBN)})
				if err != nil {
					return err
				}
			}
		}
		for edition, copies := range sales {
			if err := q.UpsertSeedSales(ctx, sqlc.UpsertSeedSalesParams{EditionID: edition, Month: month, Copies: int32(copies)}); err != nil {
				return err
			}
		}
		for _, l := range lows {
			if err := q.UpsertSeedPriceLow(ctx, sqlc.UpsertSeedPriceLowParams{EditionID: l.EditionID, LowBdt: int32(l.LowBdt), Since: now}); err != nil {
				return err
			}
		}
		return nil
	})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// loadBookExtras loads details, look inside, series and the questions readers asked.
func loadBookExtras(ctx context.Context, r *Run) error {
	var details []struct {
		BookID, Description string
		Pages               int
	}
	var inside []struct {
		BookID      string          `json:"bookId"`
		Contents    json.RawMessage `json:"contents"`
		SamplePages json.RawMessage `json:"samplePages"`
	}
	var series []struct {
		ID, Name string
		Entries  json.RawMessage
	}
	type seedAnswer struct {
		ID, Text, AuthorName string
		AgeMinutes           int
		IsStaff              bool
	}
	var questions []struct {
		BookID, ID, Text, AskerName string
		AgeMinutes                  int
		Answers                     []seedAnswer
	}
	for name, v := range map[string]any{"book_details.json": &details, "look_inside.json": &inside, "series.json": &series, "book_questions.json": &questions} {
		if err := r.read(name, v); err != nil {
			return err
		}
	}
	now := time.Now()
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, d := range details {
			if err := q.UpsertSeedBookDetails(ctx, sqlc.UpsertSeedBookDetailsParams{BookID: d.BookID, Description: d.Description, Pages: int32(d.Pages)}); err != nil {
				return err
			}
		}
		for _, l := range inside {
			if err := q.UpsertSeedLookInside(ctx, sqlc.UpsertSeedLookInsideParams{BookID: l.BookID, Contents: l.Contents, SamplePages: l.SamplePages}); err != nil {
				return err
			}
		}
		for _, s := range series {
			if err := q.UpsertSeedSeries(ctx, sqlc.UpsertSeedSeriesParams{ID: s.ID, Name: s.Name, Entries: s.Entries}); err != nil {
				return err
			}
		}
		for _, qu := range questions {
			at := now.Add(-time.Duration(qu.AgeMinutes) * time.Minute)
			if err := q.UpsertSeedQuestion(ctx, sqlc.UpsertSeedQuestionParams{ID: qu.ID, BookID: qu.BookID, AskerName: qu.AskerName, Text: qu.Text, AskedAt: at}); err != nil {
				return err
			}
			for _, a := range qu.Answers {
				err := q.UpsertSeedAnswer(ctx, sqlc.UpsertSeedAnswerParams{ID: a.ID, QuestionID: qu.ID, AuthorName: a.AuthorName, Text: a.Text,
					AnsweredAt: now.Add(-time.Duration(a.AgeMinutes) * time.Minute), IsStaff: a.IsStaff})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
