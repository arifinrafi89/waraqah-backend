package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrBooklistNotYours = Refusal(httpx.ErrBooklistNotYours)
	ErrBooklistInvalid  = Refusal(httpx.ErrBooklistInvalid)
	ErrBookUnknown      = Refusal(httpx.ErrBookUnknown)
	ErrQuestionInvalid  = Refusal(httpx.ErrQuestionInvalid)
)

// SaveMineInput is the body of /booklists/mine/save: no id makes a new own list (a name is
// needed); with an id it renames the list and/or replaces its books.
type SaveMineInput struct {
	ID      *string   `json:"id"`
	Name    *string   `json:"name"`
	BookIDs *[]string `json:"bookIds"`
}

// SaveMine is BooklistFakeApi._saveMine.
func (s *Service) SaveMine(ctx context.Context, snap *Snapshot, viewer string, in SaveMineInput) (*Booklist, error) {
	name, hasName := trimmed(in.Name)
	var old *sqlc.Booklist
	if in.ID != nil {
		row, err := s.q().GetBooklist(ctx, *in.ID)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (!row.OwnerID.Valid || row.OwnerID.String != viewer)) {
			return nil, ErrBooklistNotYours
		}
		if err != nil {
			return nil, err
		}
		old = &row
	}
	if (hasName && name == "") || (in.ID == nil && !hasName) {
		return nil, ErrBooklistInvalid
	}
	if in.BookIDs != nil && !validBookIDs(snap, *in.BookIDs) {
		return nil, ErrBooklistInvalid
	}
	if old == nil {
		books := []string{}
		if in.BookIDs != nil {
			books = *in.BookIDs
		}
		row, err := s.q().InsertBooklist(ctx, sqlc.InsertBooklistParams{
			ID: "bl-mine-" + strings.TrimPrefix(ids.New("x"), "x-"), OwnerID: pgtype.Text{String: viewer, Valid: true},
			TitleEn: name, BookIds: books, UpdatedAt: s.Now()})
		if err != nil {
			return nil, err
		}
		b := toBooklist(snap, row, viewer)
		return &b, nil
	}
	titleEn, titleBn, books := old.TitleEn, old.TitleBn, old.BookIds
	if hasName {
		titleEn, titleBn = name, name
	}
	if in.BookIDs != nil {
		books = *in.BookIDs
	}
	row, err := s.q().UpdateBooklist(ctx, sqlc.UpdateBooklistParams{ID: old.ID, TitleEn: titleEn, TitleBn: titleBn, BookIds: books, UpdatedAt: s.Now()})
	if err != nil {
		return nil, err
	}
	b := toBooklist(snap, row, viewer)
	return &b, nil
}

func validBookIDs(snap *Snapshot, ids []string) bool {
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return false
		}
		seen[id] = true
		if _, ok := snap.Book(id); !ok {
			return false
		}
	}
	return true
}

// DeleteMine deletes one of the viewer own lists; refused for any other list.
func (s *Service) DeleteMine(ctx context.Context, viewer, id string) error {
	n, err := s.q().DeleteOwnBooklist(ctx, sqlc.DeleteOwnBooklistParams{ID: id, OwnerID: pgtype.Text{String: viewer, Valid: true}})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrBooklistNotYours
	}
	return nil
}
