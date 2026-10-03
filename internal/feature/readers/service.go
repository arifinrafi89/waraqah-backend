// Package readers is a reader's public page and who follows whom. Port of reader_fake_api.dart
// and follow_fake_store.dart.
package readers

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/profile"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Refusal is a rule the request broke; the handler answers 200 null with its code.
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Refusal codes (strings live in httpx/errors.go, documented in docs/error-codes.md).
const (
	ErrUnknown       = Refusal(httpx.ErrReaderUnknown)
	ErrFollowRefused = Refusal(httpx.ErrFollowRefused)
)

// Reader is ReaderModel. For a private profile only the name and the follow state are sent.
type Reader struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Area             string     `json:"area"`
	District         string     `json:"district"`
	MemberSince      *time.Time `json:"memberSince"`
	Followers        int        `json:"followers"`
	Following        int        `json:"following"`
	IsFollowing      bool       `json:"isFollowing"`
	IsMe             bool       `json:"isMe"`
	ProfileVisible   bool       `json:"profileVisible"`
	BiteCount        int        `json:"biteCount"`
	LiveListingCount int        `json:"liveListingCount"`
}

// Prefs reads a reader's privacy settings (profile).
type Prefs interface {
	Prefs(ctx context.Context, userID string) (profile.Prefs, error)
}

// Blocks is blocks.Checker.
type Blocks interface {
	IsBlocked(ctx context.Context, viewer, other string) (bool, error)
}

// Counts are the numbers of a page that other features own.
type Counts interface {
	CountBy(ctx context.Context, authorID string) (int, error)
}

// Listings counts a reader's listings on sale (p2p).
type Listings interface {
	LiveListingCount(ctx context.Context, sellerID string) (int, error)
}

// Service holds the reader page and the follows.
type Service struct {
	DB       *db.DB
	Prefs    Prefs
	Blocks   Blocks
	Bites    Counts
	Listings Listings
	Notify   notifications.Sender
	Clock    clock.Clock
	Loc      *time.Location
	Log      *slog.Logger
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// Me is the id the app uses for the signed-in reader's own page (the fake API's "me", see
// profile_header_card.dart: readerProvider('me')).
const Me = "me"

// resolve turns "me" into the signed-in reader's id.
func (s *Service) resolve(viewer, id string) string {
	if id == Me && viewer != "" {
		return viewer
	}
	return id
}

// Following is the bites Follows: the readers a reader follows.
func (s *Service) Following(ctx context.Context, userID string) ([]string, error) {
	return s.DB.Q().ListFollowing(ctx, userID)
}

// Page is a reader's page as the viewer sees it, or nil for someone unknown. A private profile
// (profileVisible off), or a reader blocked either way, shows only the name and the follow state;
// with activityVisible off the Bites are not counted. The reader always sees their own page.
func (s *Service) Page(ctx context.Context, viewer, id string) (*Reader, error) {
	id = s.resolve(viewer, id)
	q := s.DB.Q()
	p, err := q.GetReaderPage(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	isMe := viewer != "" && viewer == id
	prefs, err := s.Prefs.Prefs(ctx, id)
	if err != nil {
		return nil, err
	}
	blocked, err := s.Blocks.IsBlocked(ctx, viewer, id)
	if err != nil {
		return nil, err
	}
	following := false
	if viewer != "" && !isMe {
		if following, err = q.IsFollowing(ctx, sqlc.IsFollowingParams{FollowerID: viewer, FolloweeID: id}); err != nil {
			return nil, err
		}
	}
	out := &Reader{ID: p.ID, Name: p.Name, IsMe: isMe, IsFollowing: following, ProfileVisible: isMe || (prefs.ProfileVisible && !blocked)}
	if !out.ProfileVisible {
		return out, nil
	}
	since := p.MemberSince.In(s.Loc).Truncate(time.Second)
	out.Area, out.District, out.MemberSince = p.Area, p.District, &since
	counts, err := q.FollowCounts(ctx, id)
	if err != nil {
		return nil, err
	}
	out.Followers, out.Following = int(counts.Followers), int(counts.Following)
	if isMe || prefs.ActivityVisible {
		if out.BiteCount, err = s.Bites.CountBy(ctx, id); err != nil {
			return nil, err
		}
	}
	if out.LiveListingCount, err = s.Listings.LiveListingCount(ctx, id); err != nil {
		return nil, err
	}
	return out, nil
}

// Follow is FollowFakeStore.follow: the viewer follows or unfollows a reader and the page is
// answered. Following yourself, someone unknown or someone blocked either way is refused. The
// followed reader is told the first time.
func (s *Service) Follow(ctx context.Context, viewer, id string, follow bool) (*Reader, error) {
	if id = s.resolve(viewer, id); id == viewer {
		return nil, ErrFollowRefused
	}
	q := s.DB.Q()
	if _, err := q.GetReaderPage(ctx, id); err != nil {
		if isNoRows(err) {
			return nil, ErrUnknown
		}
		return nil, err
	}
	if blocked, err := s.Blocks.IsBlocked(ctx, viewer, id); err != nil || blocked {
		if err == nil {
			err = ErrFollowRefused
		}
		return nil, err
	}
	if !follow {
		if err := q.Unfollow(ctx, sqlc.UnfollowParams{FollowerID: viewer, FolloweeID: id}); err != nil {
			return nil, err
		}
		return s.Page(ctx, viewer, id)
	}
	n, err := q.Follow(ctx, sqlc.FollowParams{FollowerID: viewer, FolloweeID: id, At: s.Clock.Now()})
	if err != nil {
		return nil, err
	}
	if n > 0 {
		name := "?"
		if people, err := q.ListUserNames(ctx, []string{viewer}); err == nil && len(people) > 0 {
			name = people[0].Name
		}
		if err := notifications.Followed(ctx, s.Notify, id, viewer, name); err != nil {
			s.Log.Error("follow notification failed", "error", err)
		}
	}
	return s.Page(ctx, viewer, id)
}
