package catalogadmin

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Record is a Category, Author or Publisher as the admin endpoints send it, with how many Books
// use it. Empty nameBn and section are left out, as in the app.
type Record struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NameBn    string `json:"nameBn,omitempty"`
	Section   string `json:"section,omitempty"`
	BookCount int    `json:"bookCount"`
}

// RecordInput is the body of a record save: no id adds one.
type RecordInput struct {
	ID      *string `json:"id"`
	Name    string  `json:"name"`
	NameBn  string  `json:"nameBn"`
	Section *string `json:"section"`
}

func recordPrefix(kind string) string {
	return map[string]string{"category": "cat", "author": "au", "publisher": "pub"}[kind]
}

func counts(ctx context.Context, q *sqlc.Queries) (map[string]int, error) {
	rows, err := q.ListRecordCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.Kind+":"+r.ID] = int(r.N)
	}
	return out, nil
}

func (s *Service) listRecords(ctx context.Context, q *sqlc.Queries, kind string) ([]Record, error) {
	n, err := counts(ctx, q)
	if err != nil {
		return nil, err
	}
	out := []Record{}
	switch kind {
	case "category":
		rows, err := q.ListCategories(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range rows {
			out = append(out, Record{ID: c.ID, Name: c.NameEn, NameBn: c.NameBn, Section: c.Section, BookCount: n["category:"+c.ID]})
		}
	case "author":
		rows, err := q.ListAuthors(ctx)
		if err != nil {
			return nil, err
		}
		for _, a := range rows {
			out = append(out, Record{ID: a.ID, Name: a.Name, NameBn: a.NameBn.String, BookCount: n["author:"+a.ID]})
		}
	case "publisher":
		rows, err := q.ListPublishers(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range rows {
			out = append(out, Record{ID: p.ID, Name: p.Name, NameBn: p.NameBn.String, BookCount: n["publisher:"+p.ID]})
		}
	}
	return out, nil
}

// Records lists every record of a kind with its book count.
func (s *Service) Records(ctx context.Context, kind string) ([]Record, error) {
	return s.listRecords(ctx, s.DB.Q(), kind)
}

func ids(list []Record) []string {
	out := make([]string, len(list))
	for i, r := range list {
		out[i] = r.ID
	}
	return out
}

// SaveRecord is CatalogAdminFakeRecords.save: adds (no id) or updates one. A used Category keeps
// its Section, so its Books stay in theirs; renaming an Author renames it on all their Books.
func (s *Service) SaveRecord(ctx context.Context, kind string, in RecordInput) (Record, error) {
	var out Record
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		var err error
		out, err = s.saveRecord(ctx, q, kind, in)
		return err
	})
	if err == nil {
		s.changed(ctx, false)
	}
	return out, err
}

func (s *Service) saveRecord(ctx context.Context, q *sqlc.Queries, kind string, in RecordInput) (Record, error) {
	name, nameBn := strings.TrimSpace(in.Name), strings.TrimSpace(in.NameBn)
	section := ""
	if in.Section != nil {
		if !oneOf(sections, *in.Section) {
			return Record{}, ErrRecordInvalid
		}
		section = *in.Section
	}
	existing, err := s.listRecords(ctx, q, kind)
	if err != nil {
		return Record{}, err
	}
	known := ids(existing)
	id := ""
	if in.ID != nil {
		id = *in.ID
	} else {
		id = uniqueID(recordPrefix(kind), name, known)
	}
	if len(CheckRecord(kind, name, nameBn)) > 0 {
		return Record{}, ErrRecordInvalid
	}
	if in.ID != nil && !oneOf(known, id) {
		return Record{}, ErrRecordUnknown
	}
	switch kind {
	case "category":
		old, err := q.GetCategory(ctx, id)
		isOld := !errors.Is(err, pgx.ErrNoRows)
		if err != nil && isOld {
			return Record{}, err
		}
		if section == "" {
			return Record{}, ErrRecordInvalid
		}
		if isOld && old.Section != section {
			if n, err := q.CountBooksUsing(ctx, sqlc.CountBooksUsingParams{Kind: kind, ID: id}); err != nil {
				return Record{}, err
			} else if n > 0 {
				return Record{}, ErrRecordInvalid
			}
		}
		err = q.UpsertCategory(ctx, sqlc.UpsertCategoryParams{ID: id, Section: section, NameEn: name, NameBn: nameBn})
		if err != nil {
			return Record{}, err
		}
	case "author":
		err := q.UpsertAuthor(ctx, sqlc.UpsertAuthorParams{ID: id, Name: name, NameBn: text(nameBn)})
		if err != nil {
			return Record{}, err
		}
		if err := q.RenameBooksAuthor(ctx, sqlc.RenameBooksAuthorParams{AuthorID: id, Author: name}); err != nil {
			return Record{}, err
		}
	case "publisher":
		if err := q.UpsertPublisher(ctx, sqlc.UpsertPublisherParams{ID: id, Name: name, NameBn: text(nameBn)}); err != nil {
			return Record{}, err
		}
	}
	list, err := s.listRecords(ctx, q, kind)
	if err != nil {
		return Record{}, err
	}
	for _, r := range list {
		if r.ID == id {
			return r, nil
		}
	}
	return Record{}, ErrRecordUnknown
}

// DeleteRecord is CatalogAdminFakeRecords.delete: refused while a Book uses the record.
func (s *Service) DeleteRecord(ctx context.Context, kind, id string) error {
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		n, err := q.CountBooksUsing(ctx, sqlc.CountBooksUsingParams{Kind: kind, ID: id})
		if err != nil {
			return err
		}
		if n > 0 {
			return ErrRecordInUse
		}
		var rows int64
		switch kind {
		case "category":
			rows, err = q.DeleteCategory(ctx, id)
		case "author":
			rows, err = q.DeleteAuthor(ctx, id)
		case "publisher":
			rows, err = q.DeletePublisher(ctx, id)
		}
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrRecordUnknown
		}
		return nil
	})
	if err == nil {
		s.changed(ctx, false)
	}
	return err
}
