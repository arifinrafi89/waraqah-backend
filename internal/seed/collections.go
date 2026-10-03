package seed

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// loadCollections loads experts, collections and booklists. The one personal booklist of the
// fake reader belongs to the demo reader; the rest are staff lists.
func loadCollections(ctx context.Context, r *Run) error {
	var experts []struct {
		ID, Name, NameBn, CredentialEn, CredentialBn, Kind string
		Verified                                           bool
	}
	var collections []struct {
		ID, TitleEn, TitleBn, NoteEn, NoteBn string
		BookIds                              []string
		Section, ExpertID                    *string
	}
	var lists []struct {
		ID, Kind, TitleEn, TitleBn string
		NoteEn, NoteBn             *string
		BookIds                    []string
		IsMine                     bool
	}
	for name, v := range map[string]any{"experts.json": &experts, "collections.json": &collections, "booklists.json": &lists} {
		if err := r.read(name, v); err != nil {
			return err
		}
	}
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, e := range experts {
			err := q.UpsertSeedExpert(ctx, sqlc.UpsertSeedExpertParams{ID: e.ID, Name: e.Name, NameBn: e.NameBn,
				CredentialEn: e.CredentialEn, CredentialBn: e.CredentialBn, Kind: e.Kind, Verified: e.Verified})
			if err != nil {
				return err
			}
		}
		for _, c := range collections {
			err := q.UpsertSeedCollection(ctx, sqlc.UpsertSeedCollectionParams{ID: c.ID, TitleEn: c.TitleEn, TitleBn: c.TitleBn,
				NoteEn: c.NoteEn, NoteBn: c.NoteBn, Section: opt(c.Section), ExpertID: opt(c.ExpertID), BookIds: nonNil(c.BookIds)})
			if err != nil {
				return err
			}
		}
		for _, l := range lists {
			owner := pgtype.Text{}
			if l.IsMine {
				owner = pgtype.Text{String: r.Me, Valid: true}
			}
			err := q.UpsertSeedBooklist(ctx, sqlc.UpsertSeedBooklistParams{ID: l.ID, OwnerID: owner, Kind: l.Kind, TitleEn: l.TitleEn,
				TitleBn: l.TitleBn, NoteEn: opt(l.NoteEn), NoteBn: opt(l.NoteBn), BookIds: nonNil(l.BookIds)})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
