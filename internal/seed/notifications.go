package seed

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

type seedNotification struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Read       bool            `json:"read"`
	Params     json.RawMessage `json:"params"`
	Target     json.RawMessage `json:"target"`
	AgeMinutes int             `json:"ageMinutes"`
}

// loadNotifications gives the demo reader the notifications the fake API starts with. Times
// count back from now, so the demo reads as recent on the day it is seeded.
func loadNotifications(ctx context.Context, r *Run) error {
	var list []seedNotification
	if err := r.read("notifications.json", &list); err != nil {
		return err
	}
	now := time.Now()
	return r.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		for _, n := range list {
			at := now.Add(-time.Duration(n.AgeMinutes) * time.Minute)
			read := pgtype.Timestamptz{}
			if n.Read {
				read = pgtype.Timestamptz{Time: at, Valid: true}
			}
			var target []byte
			if string(n.Target) != "null" {
				target = n.Target
			}
			err := q.UpsertSeedNotification(ctx, sqlc.UpsertSeedNotificationParams{ID: n.ID, UserID: r.Me, Kind: n.Kind,
				Params: n.Params, Target: target, ReadAt: read, CreatedAt: at})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
