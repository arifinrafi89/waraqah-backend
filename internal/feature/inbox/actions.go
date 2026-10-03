package inbox

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// scope is one change of the inbox: it runs in a transaction, remembers which threads changed
// and what the demo bot answers, and both are sent after the commit.
type scope struct {
	s       *Service
	q       *sqlc.Queries
	ctx     context.Context
	viewer  string
	changed map[string]sqlc.Thread
	later   []func()
}

func (sc *scope) touch(t sqlc.Thread) { sc.changed[t.ID] = t }

// tx runs fn in a transaction. fn answers the id of the thread to return. After the commit every
// changed thread is published on inbox:<buyer> and inbox:<seller>, then the bot is scheduled.
func (s *Service) tx(ctx context.Context, viewer string, fn func(sc *scope) (string, error)) (*Thread, error) {
	sc := &scope{s: s, ctx: ctx, viewer: viewer, changed: map[string]sqlc.Thread{}}
	var id string
	err := s.DB.WithTx(ctx, func(q *sqlc.Queries) error {
		sc.q = q
		var err error
		id, err = fn(sc)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, t := range sc.changed {
		s.publish(t)
	}
	for _, f := range sc.later {
		f()
	}
	return s.Get(ctx, viewer, id)
}

// publish tells both readers of a thread that it changed: `{seq, threadId, listingId}`.
func (s *Service) publish(t sqlc.Thread) {
	for _, user := range []string{t.BuyerID, t.SellerID} {
		s.SSE.PublishSeq("inbox:"+user, map[string]any{"threadId": t.ID, "listingId": t.ListingID})
	}
}

// lock finds a thread the viewer is in, locked for the change.
func (sc *scope) lock(id string) (sqlc.Thread, error) {
	t, err := sc.q.LockThread(sc.ctx, id)
	if isNoRows(err) || (err == nil && sc.viewer != t.BuyerID && sc.viewer != t.SellerID) {
		return sqlc.Thread{}, ErrThreadUnknown
	}
	return t, err
}

func (sc *scope) post(t sqlc.Thread, author string, text, offerID, event *string, amount *int) error {
	opt := func(p *string) pgtype.Text {
		if p == nil {
			return pgtype.Text{}
		}
		return pgtype.Text{String: *p, Valid: true}
	}
	amt := pgtype.Int4{}
	if amount != nil {
		amt = pgtype.Int4{Int32: int32(*amount), Valid: true}
	}
	_, err := sc.q.InsertMessage(sc.ctx, sqlc.InsertMessageParams{ID: ids.Message(), ThreadID: t.ID, AuthorID: author, At: sc.s.Clock.Now(),
		Text: opt(text), OfferID: opt(offerID), Event: opt(event), AmountBdt: amt})
	sc.touch(t)
	return err
}

func (sc *scope) event(t sqlc.Thread, author, event string, amount *int) error {
	return sc.post(t, author, nil, nil, &event, amount)
}

// readUp marks everything in the thread as seen by the viewer.
func (sc *scope) readUp(t sqlc.Thread) error {
	sc.touch(t)
	if sc.viewer == t.BuyerID {
		return sc.q.ReadUpBuyer(sc.ctx, t.ID)
	}
	return sc.q.ReadUpSeller(sc.ctx, t.ID)
}

// blocked: one of the two readers blocked the other.
func (sc *scope) blocked(other string) (bool, error) {
	return sc.s.Blocks.IsBlocked(sc.ctx, sc.viewer, other)
}

// openFor is the thread of the buyer about a listing, started if needed. Sellers do not message
// themselves, nobody messages a reader they blocked (or who blocked them), and a sold book cannot
// start a new conversation.
func (sc *scope) openFor(listingID string) (sqlc.Thread, error) {
	listing, err := sc.s.Market.Find(sc.ctx, sc.viewer, listingID)
	if err != nil {
		return sqlc.Thread{}, err
	}
	if listing == nil {
		return sqlc.Thread{}, ErrListingUnknown
	}
	if listing.SellerID == sc.viewer {
		return sqlc.Thread{}, ErrListingOwn
	}
	if b, err := sc.blocked(listing.SellerID); err != nil {
		return sqlc.Thread{}, err
	} else if b {
		return sqlc.Thread{}, ErrBlocked
	}
	if t, err := sc.q.FindThreadOf(sc.ctx, sqlc.FindThreadOfParams{ListingID: listingID, BuyerID: sc.viewer}); err == nil {
		return t, nil
	} else if !isNoRows(err) {
		return sqlc.Thread{}, err
	}
	if !p2p.IsOpen(listing.Status) {
		return sqlc.Thread{}, ErrListingClosed
	}
	err = sc.q.InsertThread(sc.ctx, sqlc.InsertThreadParams{ID: ids.Thread(), ListingID: listingID, BuyerID: sc.viewer, SellerID: listing.SellerID})
	if err != nil {
		return sqlc.Thread{}, err
	}
	return sc.q.FindThreadOf(sc.ctx, sqlc.FindThreadOfParams{ListingID: listingID, BuyerID: sc.viewer})
}

// Open answers the thread of the buyer about a listing, started if needed.
func (s *Service) Open(ctx context.Context, viewer, listingID string) (*Thread, error) {
	return s.tx(ctx, viewer, func(sc *scope) (string, error) {
		t, err := sc.openFor(listingID)
		return t.ID, err
	})
}

// Send writes a message to the other reader of a thread.
func (s *Service) Send(ctx context.Context, viewer, threadID, text string) (*Thread, error) {
	return s.tx(ctx, viewer, func(sc *scope) (string, error) {
		t, err := sc.lock(threadID)
		if err != nil {
			return "", err
		}
		text = strings.TrimSpace(text)
		if text == "" || textutil.Len(text) > MaxMessageLength {
			return "", ErrMessageInvalid
		}
		if b, err := sc.blocked(otherOf(t, viewer)); err != nil {
			return "", err
		} else if b {
			return "", ErrBlocked
		}
		if err := sc.post(t, viewer, &text, nil, nil, nil); err != nil {
			return "", err
		}
		reply := "Thanks for getting back to me!"
		if t.BuyerID == viewer {
			reply = "Hi! Yes, it's still available. Ask me anything."
		}
		return t.ID, sc.afterWriting(t, reply)
	})
}

// Offer makes an offer on a live listing (opening the thread if needed). One offer waits at a time.
func (s *Service) Offer(ctx context.Context, viewer, listingID string, amountBdt int, handover string) (*Thread, error) {
	return s.tx(ctx, viewer, func(sc *scope) (string, error) {
		listing, err := s.Market.Find(ctx, viewer, listingID)
		if err != nil {
			return "", err
		}
		if listing == nil {
			return "", ErrListingUnknown
		}
		if listing.Status != p2p.Live {
			return "", ErrListingUnavailable
		}
		if !ValidHandover(handover) || CheckOffer(amountBdt, listing.PriceBdt, listing.IsNegotiable) != OfferOK {
			return "", ErrOfferInvalid
		}
		t, err := sc.openFor(listingID)
		if err != nil {
			return "", err
		}
		if _, err := sc.q.PendingOffer(ctx, t.ID); err == nil {
			return "", ErrOfferPending
		} else if !isNoRows(err) {
			return "", err
		}
		offerID := ids.Offer()
		if err := sc.q.InsertOffer(ctx, sqlc.InsertOfferParams{ID: offerID, ThreadID: t.ID, AmountBdt: int32(amountBdt), Handover: handover}); err != nil {
			return "", err
		}
		if err := sc.post(t, viewer, nil, &offerID, nil, nil); err != nil {
			return "", err
		}
		return t.ID, sc.afterWriting(t, "Thanks for the offer! Let me think about it, I'll reply soon.")
	})
}

// afterWriting marks the thread as read by the writer and lets a demo reader answer, once.
func (sc *scope) afterWriting(t sqlc.Thread, reply string) error {
	if err := sc.readUp(t); err != nil {
		return err
	}
	return sc.botReply(t, reply)
}

// Read marks everything in the thread as seen by the viewer.
func (s *Service) Read(ctx context.Context, viewer, threadID string) (*Thread, error) {
	return s.tx(ctx, viewer, func(sc *scope) (string, error) {
		t, err := sc.lock(threadID)
		if err != nil {
			return "", err
		}
		return t.ID, sc.readUp(t)
	})
}
