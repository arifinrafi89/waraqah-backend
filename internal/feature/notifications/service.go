package notifications

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/sse"
)

// Sender is how every other feature tells a reader something happened (BACKEND_PLAN.md
// section 8). Call it after the transaction commits.
type Sender interface {
	Send(ctx context.Context, userID string, kind Kind, params map[string]string, target *Target) error
}

// Muted answers whether a reader muted a settings group (Profile owns the settings).
type Muted interface {
	Muted(ctx context.Context, userID, group string) (bool, error)
}

// Notification is AppNotificationModel.
type Notification struct {
	ID        string            `json:"id"`
	Kind      Kind              `json:"kind"`
	CreatedAt string            `json:"createdAt"`
	Read      bool              `json:"read"`
	Params    map[string]string `json:"params"`
	Target    *Target           `json:"target"`
}

// Service stores notifications and publishes the unread count.
type Service struct {
	DB    *db.DB
	SSE   *sse.Broker
	Prefs Muted
	Clock clock.Clock
	Loc   *time.Location
	Log   *slog.Logger
}

// Topic is the live stream of one reader.
func Topic(userID string) string { return "notifications:" + userID }

// Send stores a notification, unless the reader muted its group, and publishes the new unread
// count. It implements Sender.
func (s *Service) Send(ctx context.Context, userID string, kind Kind, params map[string]string, target *Target) error {
	if g := kind.Group(); g != "" {
		if muted, err := s.Prefs.Muted(ctx, userID, g); err != nil {
			return err
		} else if muted {
			return nil
		}
	}
	return s.insert(ctx, ids.Notification(), userID, kind, params, target, nil, s.Clock.Now())
}

// Insert stores a notification with an id, a read time and a creation time (the seed loader).
func (s *Service) insert(ctx context.Context, id, userID string, kind Kind, params map[string]string, target *Target, readAt *time.Time, at time.Time) error {
	if params == nil {
		params = map[string]string{}
	}
	pj, err := json.Marshal(params)
	if err != nil {
		return err
	}
	var tj []byte
	if target != nil {
		if tj, err = json.Marshal(target); err != nil {
			return err
		}
	}
	read := pgtype.Timestamptz{}
	if readAt != nil {
		read = pgtype.Timestamptz{Time: *readAt, Valid: true}
	}
	if err := s.DB.Q().InsertNotification(ctx, sqlc.InsertNotificationParams{
		ID: id, UserID: userID, Kind: string(kind), Params: pj, Target: tj, ReadAt: read, CreatedAt: at,
	}); err != nil {
		return err
	}
	s.publishUnread(ctx, userID)
	return nil
}

// List returns the notifications of a reader, newest first.
func (s *Service) List(ctx context.Context, userID string) ([]Notification, error) {
	rows, err := s.DB.Q().ListNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Notification, 0, len(rows))
	for _, r := range rows {
		n := Notification{ID: r.ID, Kind: Kind(r.Kind), Read: r.ReadAt.Valid,
			CreatedAt: clock.FormatRFC3339(r.CreatedAt, s.Loc), Params: map[string]string{}}
		_ = json.Unmarshal(r.Params, &n.Params)
		if len(r.Target) > 0 {
			var t Target
			if json.Unmarshal(r.Target, &t) == nil {
				n.Target = &t
			}
		}
		out = append(out, n)
	}
	return out, nil
}

// MarkRead marks one notification read. False for an id that is not the reader own.
func (s *Service) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	n, err := s.DB.Q().MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{UserID: userID, ID: id, ReadAt: pgtype.Timestamptz{Time: s.Clock.Now(), Valid: true}})
	if err != nil || n == 0 {
		return false, err
	}
	s.publishUnread(ctx, userID)
	return true, nil
}

// MarkAllRead marks every notification of the reader read.
func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	err := s.DB.Q().MarkAllNotificationsRead(ctx, sqlc.MarkAllNotificationsReadParams{UserID: userID, ReadAt: pgtype.Timestamptz{Time: s.Clock.Now(), Valid: true}})
	if err == nil {
		s.publishUnread(ctx, userID)
	}
	return err
}

// Unread counts the unread notifications of a reader.
func (s *Service) Unread(ctx context.Context, userID string) (int64, error) {
	return s.DB.Q().CountUnreadNotifications(ctx, userID)
}

func (s *Service) publishUnread(ctx context.Context, userID string) {
	n, err := s.Unread(ctx, userID)
	if err != nil {
		s.Log.Error("count unread notifications", "error", err)
		return
	}
	s.SSE.Publish(Topic(userID), map[string]int64{"unread": n})
}
