package sellback

import "github.com/arifinrafi89/waraqah-backend/internal/feature/p2p"

// FinishedItOffers is a port of FinishedItOffers (finished_it_offers.dart): the two ways to sell
// a book the reader just finished, for a copy read once (Like New).
type FinishedItOffers struct {
	// Readers is the fair range on the used marketplace.
	Readers p2p.FairPrice
	// SellBackBdt is what Waraqah pays straight away.
	SellBackBdt int
}

// FinishedItOffersOf is FinishedItOffers.of: nil without a new price.
func FinishedItOffersOf(newPriceBdt *int) *FinishedItOffers {
	if newPriceBdt == nil {
		return nil
	}
	fair := p2p.FairPriceOf(*newPriceBdt, "likeNew", 0)
	if fair == nil {
		return nil
	}
	return &FinishedItOffers{Readers: *fair, SellBackBdt: Quote(*newPriceBdt, "likeNew", 0)}
}
