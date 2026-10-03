package inbox

import (
	"context"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// The demo bot (BACKEND_PLAN.md section 12): so the live updates can be seen with one reader, a
// seeded demo reader answers once per thread, and rates the seller after a sale, a few seconds
// after the real reader acts. Ports of inbox_fake_replies.dart and inbox_fake_rating.dart. It
// only acts for seeded demo accounts, never for readers who signed up.

// botReply schedules the one answer of a demo reader to the writer of this thread.
func (sc *scope) botReply(t sqlc.Thread, text string) error {
	other := otherOf(t, sc.viewer)
	if !sc.s.DemoMode || !isDemo(other) {
		return nil
	}
	first, err := sc.q.MarkBotReplied(sc.ctx, t.ID)
	if err != nil || first == 0 {
		return err
	}
	sc.later = append(sc.later, func() {
		sc.s.afterDelay("answer", func(ctx context.Context, q *sqlc.Queries) ([]sqlc.Thread, error) {
			// A reader the demo one blocked, or who blocked it, cannot be written to.
			if blocked, err := sc.s.Blocks.IsBlocked(ctx, sc.viewer, other); err != nil || blocked {
				return nil, err
			}
			scoped := &scope{s: sc.s, q: q, ctx: ctx, viewer: other, changed: map[string]sqlc.Thread{}}
			if err := scoped.post(t, other, &text, nil, nil, nil); err != nil {
				return nil, err
			}
			return []sqlc.Thread{t}, nil
		})
	})
	return nil
}

// botRate schedules the rating a demo buyer gives the seller after a sale.
func (sc *scope) botRate(t sqlc.Thread) {
	buyer := t.BuyerID
	if !sc.s.DemoMode || !isDemo(buyer) {
		return
	}
	sc.later = append(sc.later, func() {
		sc.s.afterDelay("rating", func(ctx context.Context, q *sqlc.Queries) ([]sqlc.Thread, error) {
			err := sc.s.Market.AddRating(ctx, q, buyer, t.SellerID, t.ListingID, 5, "Smooth handover, thank you!", sc.s.Clock.Now())
			return []sqlc.Thread{t}, err
		})
	})
}

// afterDelay runs a bot action in its own transaction after BotDelay (at once when it is zero, which
// the tests use), then tells both readers.
func (s *Service) afterDelay(what string, fn func(ctx context.Context, q *sqlc.Queries) ([]sqlc.Thread, error)) {
	run := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var changed []sqlc.Thread
		err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
			var err error
			changed, err = fn(ctx, q)
			return err
		})
		if err != nil {
			s.Log.Error("demo bot could not do its part", "what", what, "error", err)
			return
		}
		for _, t := range changed {
			s.publish(t)
		}
	}
	if s.BotDelay <= 0 {
		run()
		return
	}
	time.AfterFunc(s.BotDelay, run)
}
