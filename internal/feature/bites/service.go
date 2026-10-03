package bites

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// Books finds a tagged Book (catalog.Books).
type Books interface {
	Find(ctx context.Context, id string) (catalog.Book, bool, error)
}

// Blocks is blocks.Checker: the readers hidden from a viewer, and whether two readers blocked each other.
type Blocks interface {
	BlockedEither(ctx context.Context, viewer string) ([]string, error)
	IsBlocked(ctx context.Context, viewer, other string) (bool, error)
}

// Bans is moderation.Bans.
type Bans interface {
	IsBanned(ctx context.Context, userID string) (bool, error)
}

// Follows says whom a reader follows (readers implements it).
type Follows interface {
	Following(ctx context.Context, userID string) ([]string, error)
}

// Service holds the Bite rules.
type Service struct {
	DB      *db.DB
	Books   Books
	Blocks  Blocks
	Bans    Bans
	Follows Follows
	Notify  notifications.Sender
	Clock   clock.Clock
	Loc     *time.Location
	Log     *slog.Logger
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func (s *Service) t(at time.Time) time.Time { return at.In(s.Loc).Truncate(time.Second) }

func optText(p pgtype.Text) *string {
	if !p.Valid {
		return nil
	}
	return &p.String
}

// toBite is BiteFakeJson.biteJson: the tagged Book's title comes from the catalog.
func (s *Service) toBite(ctx context.Context, viewer string, b sqlc.Bite, name, area string, likes, comments int32, liked bool) Bite {
	out := Bite{ID: b.ID, AuthorID: b.AuthorID, AuthorName: name, AuthorArea: area, Text: b.Text, CreatedAt: s.t(b.CreatedAt),
		BookID: optText(b.BookID), Spoiler: b.Spoiler, Likes: int(likes), Liked: liked, Comments: int(comments),
		IsMine: viewer != "" && b.AuthorID == viewer}
	if b.EditedAt.Valid {
		at := s.t(b.EditedAt.Time)
		out.EditedAt = &at
	}
	if b.BookID.Valid {
		if book, ok, err := s.Books.Find(ctx, b.BookID.String); err == nil && ok {
			out.BookTitle = &book.Title
		}
	}
	return out
}

// Feed is BiteFakeStore.feed: newest first, at most 30. Readers blocked either way, banned or
// gone are left out; Following keeps the readers the viewer follows (none for a guest).
func (s *Service) Feed(ctx context.Context, viewer string, q Query) ([]Bite, error) {
	hidden, err := s.Blocks.BlockedEither(ctx, viewer)
	if err != nil {
		return nil, err
	}
	following := []string{}
	if q.Following && viewer != "" {
		if following, err = s.Follows.Following(ctx, viewer); err != nil {
			return nil, err
		}
	}
	rows, err := s.DB.Q().ListBites(ctx, sqlc.ListBitesParams{Viewer: viewer, Hidden: hidden, OnlyFollowing: q.Following, Following: following,
		BookID: pgtype.Text{String: q.BookID, Valid: q.BookID != ""}, AuthorID: pgtype.Text{String: q.AuthorID, Valid: q.AuthorID != ""}})
	if err != nil {
		return nil, err
	}
	out := make([]Bite, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toBite(ctx, viewer, r.Bite, r.AuthorName, r.AuthorArea, r.Likes, r.Comments, r.Liked))
	}
	return out, nil
}

// One answers a Bite as the viewer sees it, or nil when it is unknown.
func (s *Service) One(ctx context.Context, viewer, id string) (*Bite, error) {
	return s.oneIn(ctx, s.DB.Q(), viewer, id)
}

func (s *Service) oneIn(ctx context.Context, q *sqlc.Queries, viewer, id string) (*Bite, error) {
	r, err := q.GetBite(ctx, sqlc.GetBiteParams{Viewer: viewer, ID: id})
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b := s.toBite(ctx, viewer, r.Bite, r.AuthorName, r.AuthorArea, r.Likes, r.Comments, r.Liked)
	return &b, nil
}

// Detail is BiteFakeJson.detailJson: the Bite with its top comments, oldest first, each carrying
// its replies. nil when the Bite is unknown.
func (s *Service) Detail(ctx context.Context, viewer, id string) (*Detail, error) {
	q := s.DB.Q()
	b, err := s.oneIn(ctx, q, viewer, id)
	if err != nil || b == nil {
		return nil, err
	}
	rows, err := q.ListBiteComments(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &Detail{Bite: *b, Comments: []Comment{}}
	index := map[string]int{}
	for _, c := range rows {
		cm := Comment{ID: c.ID, AuthorID: c.AuthorID, AuthorName: c.AuthorName, Text: c.Text, CreatedAt: s.t(c.CreatedAt),
			ParentID: optText(c.ParentID), IsMine: viewer != "" && c.AuthorID == viewer, Replies: []Comment{}}
		if !c.ParentID.Valid {
			index[c.ID] = len(out.Comments)
			out.Comments = append(out.Comments, cm)
			continue
		}
		if i, ok := index[c.ParentID.String]; ok {
			out.Comments[i].Replies = append(out.Comments[i].Replies, cm)
		}
	}
	return out, nil
}

// CountBy counts the Bites of a reader (the reader page).
func (s *Service) CountBy(ctx context.Context, authorID string) (int, error) {
	n, err := s.DB.Q().CountBitesBy(ctx, authorID)
	return int(n), err
}
