package inbox

import (
	"context"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
)

// sellerThread finds a thread of the viewer as its seller.
func (sc *scope) sellerThread(id string) (sqlc.Thread, error) {
	t, err := sc.lock(id)
	if err != nil {
		return t, err
	}
	if t.SellerID != sc.viewer {
		return sqlc.Thread{}, ErrThreadUnknown
	}
	return t, nil
}

// dealThread is the seller thread with the buyer the listing is reserved for.
func (sc *scope) dealThread(id string) (sqlc.Thread, *p2p.Listing, error) {
	t, err := sc.sellerThread(id)
	if err != nil {
		return t, nil, err
	}
	l, err := sc.s.Market.Find(sc.ctx, sc.viewer, t.ListingID)
	if err != nil {
		return t, nil, err
	}
	if l == nil || l.Status != p2p.Reserved || l.BuyerID != t.BuyerID {
		return t, nil, ErrDealUnknown
	}
	return t, l, nil
}

// tellOthers puts a line in the other threads about the listing; with closeOffers their waiting
// offers are closed.
func (sc *scope) tellOthers(deal sqlc.Thread, event string, closeOffers bool) error {
	threads, err := sc.q.ListThreadsAbout(sc.ctx, deal.ListingID)
	if err != nil {
		return err
	}
	for _, other := range threads {
		if other.ID == deal.ID {
			continue
		}
		if n, err := sc.messageCount(other.ID); err != nil || n == 0 {
			if err != nil {
				return err
			}
			continue
		}
		if closeOffers {
			if err := sc.q.ClosePendingOffers(sc.ctx, other.ID); err != nil {
				return err
			}
		}
		if err := sc.event(other, System, event, nil); err != nil {
			return err
		}
	}
	return nil
}

func (sc *scope) messageCount(threadID string) (int, error) {
	pos, err := sc.q.ListMessagePositions(sc.ctx, threadID)
	return len(pos), err
}

// Decide accepts or declines the waiting offer of a thread (the seller). Accepting reserves the
// book for this buyer; the seller other threads about it are told it is reserved elsewhere.
func (s *Service) Decide(ctx context.Context, viewer, threadID, offerID string, accept bool) (*Thread, error) {
	return s.tx(ctx, viewer, func(sc *scope) (string, error) {
		t, err := sc.sellerThread(threadID)
		if err != nil {
			return "", err
		}
		offer, err := sc.q.PendingOffer(ctx, t.ID)
		if isNoRows(err) || (err == nil && offer.ID != offerID) {
			return "", ErrOfferUnknown
		}
		if err != nil {
			return "", err
		}
		listing, err := s.Market.Find(ctx, viewer, t.ListingID)
		if err != nil || listing == nil {
			return "", err
		}
		if accept {
			// A blocked buyer's offer can only be declined.
			if listing.Status != p2p.Live {
				return "", ErrListingUnavailable
			}
			if b, err := sc.blocked(t.BuyerID); err != nil {
				return "", err
			} else if b {
				return "", ErrBlocked
			}
		}
		status, event := OfferDeclined, EventOfferDeclined
		if accept {
			status, event = OfferAccepted, EventOfferAccepted
		}
		if err := sc.q.SetOfferStatus(ctx, sqlc.SetOfferStatusParams{ID: offer.ID, Status: status}); err != nil {
			return "", err
		}
		amount := int(offer.AmountBdt)
		if err := sc.event(t, viewer, event, &amount); err != nil {
			return "", err
		}
		if accept {
			if err := s.Market.SetStatus(ctx, sc.q, listing.ID, p2p.Reserved, t.BuyerID); err != nil {
				return "", err
			}
			if err := sc.tellOthers(t, EventReservedElsewhere, false); err != nil {
				return "", err
			}
			if err := sc.botReply(t, "Great, thank you! When and where suits you?"); err != nil {
				return "", err
			}
		}
		return t.ID, sc.readUp(t)
	})
}

// Release puts a reserved book on sale again, for everyone talking about it (the deal fell through).
func (s *Service) Release(ctx context.Context, viewer, threadID string) (*Thread, error) {
	return s.tx(ctx, viewer, func(sc *scope) (string, error) {
		t, l, err := sc.dealThread(threadID)
		if err != nil {
			return "", err
		}
		if err := s.Market.SetStatus(ctx, sc.q, l.ID, p2p.Live, ""); err != nil {
			return "", err
		}
		threads, err := sc.q.ListThreadsAbout(ctx, l.ID)
		if err != nil {
			return "", err
		}
		for _, other := range threads {
			if n, err := sc.messageCount(other.ID); err != nil || n == 0 {
				if err != nil {
					return "", err
				}
				continue
			}
			if err := sc.event(other, viewer, EventMadeAvailable, nil); err != nil {
				return "", err
			}
		}
		return t.ID, sc.readUp(t)
	})
}

// Sold marks the reserved book as sold to the buyer of the thread, after the handover. Other
// buyers' waiting offers are closed, and a demo buyer rates the seller a moment later.
func (s *Service) Sold(ctx context.Context, viewer, threadID string) (*Thread, error) {
	return s.tx(ctx, viewer, func(sc *scope) (string, error) {
		t, l, err := sc.dealThread(threadID)
		if err != nil {
			return "", err
		}
		if err := s.Market.SetStatus(ctx, sc.q, l.ID, p2p.Sold, t.BuyerID); err != nil {
			return "", err
		}
		if err := sc.event(t, viewer, EventSold, nil); err != nil {
			return "", err
		}
		sc.botRate(t)
		if err := sc.tellOthers(t, EventSoldElsewhere, true); err != nil {
			return "", err
		}
		return t.ID, sc.readUp(t)
	})
}

// Rate rates the other reader after the sale: once each, 1 to 5 stars and a short comment.
func (s *Service) Rate(ctx context.Context, viewer, threadID string, stars int, comment string) (*Thread, error) {
	return s.tx(ctx, viewer, func(sc *scope) (string, error) {
		t, err := sc.lock(threadID)
		if err != nil {
			return "", err
		}
		l, err := s.Market.Find(ctx, viewer, t.ListingID)
		if err != nil || l == nil {
			return "", err
		}
		ratings, err := s.Market.RatingsFor(ctx, sc.q, []string{l.ID})
		if err != nil {
			return "", err
		}
		if _, rated := ratings[l.ID][viewer]; rated || l.Status != p2p.Sold || l.BuyerID != t.BuyerID {
			return "", ErrRatingNotAllowed
		}
		comment = strings.TrimSpace(comment)
		if !CheckRating(stars, comment) {
			return "", ErrRatingInvalid
		}
		if err := s.Market.AddRating(ctx, sc.q, viewer, otherOf(t, viewer), l.ID, stars, comment, s.Clock.Now()); err != nil {
			return "", err
		}
		return t.ID, sc.readUp(t)
	})
}
