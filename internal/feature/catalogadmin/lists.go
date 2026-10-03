package catalogadmin

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

var staffKinds = []string{"classList", "examPrep", "bookClub"}

// ListID is the answer of a Collection or Booklist save or delete.
type ListID struct {
	ID string `json:"id"`
}

// booksExist checks every id is a Book of the catalog.
func (s *Service) booksExist(ctx context.Context, q *sqlc.Queries, ids []string) (bool, error) {
	all, err := q.BookIDs(ctx)
	if err != nil {
		return false, err
	}
	have := map[string]bool{}
	for _, id := range all {
		have[id] = true
	}
	for _, id := range ids {
		if !have[id] {
			return false, nil
		}
	}
	return true, nil
}

// SaveCollection is CatalogAdminFakeLists.saveCollection: a ListDraft without a kind.
func (s *Service) SaveCollection(ctx context.Context, d ListDraft) (ListID, error) {
	var out ListID
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		known, err := q.ListCollectionIDs(ctx)
		if err != nil {
			return err
		}
		if len(CheckList(d)) > 0 || d.Kind != nil || (d.Section != nil && !oneOf(sections, *d.Section)) {
			return ErrListInvalid
		}
		if d.ID != nil && !oneOf(known, *d.ID) {
			return ErrListUnknown
		}
		if ok, err := s.booksExist(ctx, q, d.BookIDs); err != nil {
			return err
		} else if !ok {
			return ErrListInvalid
		}
		if d.ExpertID != nil {
			if ok, err := q.ExpertExists(ctx, *d.ExpertID); err != nil {
				return err
			} else if !ok {
				return ErrListInvalid
			}
		}
		id := ""
		if d.ID != nil {
			id = *d.ID
		} else {
			id = uniqueID("col", d.TitleEn, known)
		}
		out = ListID{ID: id}
		return q.UpsertCollection(ctx, sqlc.UpsertCollectionParams{ID: id, TitleEn: strings.TrimSpace(d.TitleEn), TitleBn: strings.TrimSpace(d.TitleBn),
			NoteEn: strings.TrimSpace(d.NoteEn), NoteBn: strings.TrimSpace(d.NoteBn), Section: ptrText(d.Section), ExpertID: ptrText(d.ExpertID), BookIds: d.BookIDs})
	})
	if err == nil {
		s.changed(ctx, false)
	}
	return out, err
}

func ptrText(p *string) pgtype.Text {
	if p == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *p, Valid: true}
}

// DeleteCollection is CatalogAdminFakeLists.deleteCollection.
func (s *Service) DeleteCollection(ctx context.Context, id string) (ListID, error) {
	n, err := s.DB.Q().DeleteCollection(ctx, id)
	if err != nil {
		return ListID{}, err
	}
	if n == 0 {
		return ListID{}, ErrListUnknown
	}
	s.changed(ctx, false)
	return ListID{ID: id}, nil
}

// SaveBooklist is CatalogAdminFakeLists.saveBooklist: Staff only, with a Staff kind. A Reader own
// lists are not theirs to change.
func (s *Service) SaveBooklist(ctx context.Context, d ListDraft) (ListID, error) {
	var out ListID
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		known, err := q.ListStaffBooklistIDs(ctx)
		if err != nil {
			return err
		}
		if len(CheckList(d)) > 0 || d.Kind == nil || !oneOf(staffKinds, *d.Kind) {
			return ErrListInvalid
		}
		if d.ID != nil && !oneOf(known, *d.ID) {
			return ErrListUnknown
		}
		if ok, err := s.booksExist(ctx, q, d.BookIDs); err != nil {
			return err
		} else if !ok {
			return ErrListInvalid
		}
		id := ""
		if d.ID != nil {
			id = *d.ID
		} else {
			id = uniqueID("bl", d.TitleEn, known)
		}
		out = ListID{ID: id}
		noteEn, noteBn := strings.TrimSpace(d.NoteEn), strings.TrimSpace(d.NoteBn)
		return q.UpsertStaffBooklist(ctx, sqlc.UpsertStaffBooklistParams{ID: id, Kind: *d.Kind, TitleEn: strings.TrimSpace(d.TitleEn),
			TitleBn: strings.TrimSpace(d.TitleBn), NoteEn: text(noteEn), NoteBn: text(noteBn), BookIds: d.BookIDs, UpdatedAt: s.Clock.Now()})
	})
	if err == nil {
		s.changed(ctx, false)
	}
	return out, err
}

// DeleteBooklist is CatalogAdminFakeLists.deleteBooklist; a Reader own list is refused.
func (s *Service) DeleteBooklist(ctx context.Context, id string) (ListID, error) {
	n, err := s.DB.Q().DeleteStaffBooklist(ctx, id)
	if err != nil {
		return ListID{}, err
	}
	if n == 0 {
		return ListID{}, ErrListUnknown
	}
	s.changed(ctx, false)
	return ListID{ID: id}, nil
}
