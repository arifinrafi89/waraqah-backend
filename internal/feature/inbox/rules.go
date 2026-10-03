// Package inbox is the chat of buyers and sellers of used books: threads, offers, the deal that
// follows (reserve, sold, release) and the ratings after a sale, with live updates.
package inbox

import (
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// Port of OfferRules and RatingRules (lib/features/inbox/domain/entities/).
const (
	// MaxMessageLength is the longest chat message, in characters.
	MaxMessageLength = 1000
	// MaxComment is the longest comment of a rating.
	MaxComment = 300
)

// Statuses of an offer (OfferStatus), handovers (OfferHandover) and deal events (ThreadEvent).
const (
	OfferPending  = "pending"
	OfferAccepted = "accepted"
	OfferDeclined = "declined"
	OfferClosed   = "closed"

	HandoverMeetup  = "meetup"
	HandoverCourier = "courier"

	EventOfferAccepted     = "offerAccepted"
	EventOfferDeclined     = "offerDeclined"
	EventReservedElsewhere = "reservedElsewhere"
	EventMadeAvailable     = "madeAvailable"
	EventSold              = "sold"
	EventSoldElsewhere     = "soldElsewhere"

	// System is the author of deal lines nobody in the thread wrote.
	System = "system"
)

// OfferProblem is why an offer cannot be sent.
type OfferProblem string

// The problems OfferRules finds.
const (
	OfferOK         OfferProblem = ""
	OfferMissing    OfferProblem = "missing"
	OfferAboveAsk   OfferProblem = "aboveAsking"
	OfferFixedPrice OfferProblem = "fixedPrice"
)

// CheckOffer is OfferRules.check: at least 1 taka and never more than the asking price, and
// exactly the asking price when the seller said it is not negotiable.
func CheckOffer(amountBdt, askingBdt int, negotiable bool) OfferProblem {
	switch {
	case amountBdt < 1:
		return OfferMissing
	case amountBdt > askingBdt:
		return OfferAboveAsk
	case !negotiable && amountBdt != askingBdt:
		return OfferFixedPrice
	}
	return OfferOK
}

// CheckRating is RatingRules.check: 1 to 5 stars, and a comment of up to 300 characters.
func CheckRating(stars int, comment string) bool {
	return stars >= 1 && stars <= 5 && textutil.Len(strings.TrimSpace(comment)) <= MaxComment
}

// ValidHandover reports whether h is meetup or courier.
func ValidHandover(h string) bool { return h == HandoverMeetup || h == HandoverCourier }
