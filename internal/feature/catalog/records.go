package catalog

import (
	"context"
	"strings"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

func optional(s string, valid bool) *string {
	if !valid {
		return nil
	}
	return &s
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func toExpert(e sqlc.Expert) Expert {
	return Expert{ID: e.ID, Name: e.Name, NameBn: e.NameBn, CredentialEn: e.CredentialEn, CredentialBn: e.CredentialBn,
		Kind: e.Kind, Verified: e.Verified}
}

// visibleBooks resolves ids to the books on the storefront, in the order of ids (hidden ones are left out).
func visibleBooks(snap *Snapshot, ids []string) []Book {
	out := []Book{}
	for _, id := range ids {
		if b, ok := snap.Book(id); ok && !b.Hidden {
			out = append(out, b)
		}
	}
	return out
}

// CollectionFilter narrows /collections.
type CollectionFilter struct {
	Section   string
	Expert    string
	HasExpert *bool
}

// Collections lists the collections, each with its books and its expert (CollectionFakeApi).
func (s *Service) Collections(ctx context.Context, snap *Snapshot, f CollectionFilter) ([]Collection, error) {
	experts, err := s.experts(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.q().ListCollections(ctx)
	if err != nil {
		return nil, err
	}
	out := []Collection{}
	for _, c := range rows {
		hasExpert := c.ExpertID.Valid
		if (f.Section != "" && (!c.Section.Valid || c.Section.String != f.Section)) ||
			(f.Expert != "" && (!c.ExpertID.Valid || c.ExpertID.String != f.Expert)) ||
			(f.HasExpert != nil && hasExpert != *f.HasExpert) {
			continue
		}
		out = append(out, buildCollection(snap, experts, c))
	}
	return out, nil
}

// CollectionByID returns one collection or nil.
func (s *Service) CollectionByID(ctx context.Context, snap *Snapshot, id string) (*Collection, error) {
	all, err := s.Collections(ctx, snap, CollectionFilter{})
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, nil
}

func (s *Service) experts(ctx context.Context) (map[string]Expert, error) {
	rows, err := s.q().ListExperts(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]Expert{}
	for _, e := range rows {
		out[e.ID] = toExpert(e)
	}
	return out, nil
}

func buildCollection(snap *Snapshot, experts map[string]Expert, c sqlc.Collection) Collection {
	out := Collection{ID: c.ID, TitleEn: c.TitleEn, TitleBn: c.TitleBn, NoteEn: c.NoteEn, NoteBn: c.NoteBn,
		BookIDs: nonNil(c.BookIds), Section: optional(c.Section.String, c.Section.Valid),
		ExpertID: optional(c.ExpertID.String, c.ExpertID.Valid), Books: visibleBooks(snap, c.BookIds)}
	if c.ExpertID.Valid {
		if e, ok := experts[c.ExpertID.String]; ok {
			out.Expert = &e
		}
	}
	return out
}

// Experts lists every expert.
func (s *Service) Experts(ctx context.Context) ([]Expert, error) {
	rows, err := s.q().ListExperts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Expert, 0, len(rows))
	for _, e := range rows {
		out = append(out, toExpert(e))
	}
	return out, nil
}

// ExpertDetail returns an expert with their collections, or nil.
func (s *Service) ExpertDetail(ctx context.Context, snap *Snapshot, id string) (*ExpertPage, error) {
	experts, err := s.experts(ctx)
	if err != nil {
		return nil, err
	}
	e, ok := experts[id]
	if !ok {
		return nil, nil
	}
	cols, err := s.Collections(ctx, snap, CollectionFilter{Expert: id})
	if err != nil {
		return nil, err
	}
	return &ExpertPage{Expert: e, Collections: cols}, nil
}

func toBooklist(snap *Snapshot, b sqlc.Booklist, viewer string) Booklist {
	return Booklist{ID: b.ID, TitleEn: b.TitleEn, TitleBn: b.TitleBn, Kind: b.Kind, BookIDs: nonNil(b.BookIds),
		NoteEn: optional(b.NoteEn.String, b.NoteEn.Valid), NoteBn: optional(b.NoteBn.String, b.NoteBn.Valid),
		IsMine: viewer != "" && b.OwnerID.Valid && b.OwnerID.String == viewer, Books: visibleBooks(snap, b.BookIds)}
}

// Booklists lists every staff booklist and the viewer own lists.
func (s *Service) Booklists(ctx context.Context, snap *Snapshot, viewer string) ([]Booklist, error) {
	rows, err := s.q().ListBooklists(ctx, viewer)
	if err != nil {
		return nil, err
	}
	out := make([]Booklist, 0, len(rows))
	for _, b := range rows {
		out = append(out, toBooklist(snap, b, viewer))
	}
	return out, nil
}

// trimmed reports the trimmed text of an optional string.
func trimmed(s *string) (string, bool) {
	if s == nil {
		return "", false
	}
	return strings.TrimSpace(*s), true
}

// Now is the current time (kept here so the booklist code reads in one place).
func (s *Service) Now() time.Time { return s.Clock.Now() }
