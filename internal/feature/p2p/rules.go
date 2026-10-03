// Package p2p is the used marketplace: listings readers sell, seller pages, and the status changes
// the inbox and handled sales make.
package p2p

import (
	"slices"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// Statuses of a listing (P2pListingStatus).
const (
	Draft            = "draft"
	InReview         = "inReview"
	ChangesRequested = "changesRequested"
	Rejected         = "rejected"
	Live             = "live"
	Reserved         = "reserved"
	Sold             = "sold"
)

// Port of ListingRules (lib/features/p2p/domain/entities/listing_rules.dart).
const (
	MaxTitle = 120
	MaxNote  = 500
	MaxPrice = 50000
)

// Flags a seller can tick, and the photo slots in order. Front and back are needed.
var (
	Flags      = []string{"highlighting", "notes", "damage"}
	PhotoSlots = []string{"front", "back", "spine", "inside", "damage"}
)

// Problem is what a listing lacks before it is saved or sent for review.
type Problem string

// The problems ListingRules finds, in the order it checks them.
const (
	ProblemNone            Problem = ""
	ProblemNoTitle         Problem = "noTitle"
	ProblemTitleTooLong    Problem = "titleTooLong"
	ProblemNoteTooLong     Problem = "noteTooLong"
	ProblemNoPrice         Problem = "noPrice"
	ProblemPriceTooHigh    Problem = "priceTooHigh"
	ProblemNeedFront       Problem = "needFront"
	ProblemNeedBack        Problem = "needBack"
	ProblemNeedDamagePhoto Problem = "needDamagePhoto"
)

// Draft is the part of a listing the rules look at.
type Fields struct {
	Title    string
	Note     string
	PriceBdt int
	Flags    []string
	Photos   []string // slots that have a photo
}

// CanEdit: the seller can still change a listing that is not sent yet, or was sent back.
func CanEdit(status string) bool {
	return status == Draft || status == ChangesRequested || status == Rejected
}

// IsOpen: shown in the marketplace, on sale or reserved.
func IsOpen(status string) bool { return status == Live || status == Reserved }

// needsPhoto: whether slot must have a photo before the listing goes for review.
func needsPhoto(f Fields, slot string) bool {
	return slot == "front" || slot == "back" || (slot == "damage" && slices.Contains(f.Flags, "damage"))
}

// Check is ListingRules.check: a draft needs only its title (and sane lengths and price).
func Check(f Fields, submit bool) Problem {
	title := strings.TrimSpace(f.Title)
	switch {
	case title == "":
		return ProblemNoTitle
	case textutil.Len(title) > MaxTitle:
		return ProblemTitleTooLong
	case textutil.Len(strings.TrimSpace(f.Note)) > MaxNote:
		return ProblemNoteTooLong
	case f.PriceBdt > MaxPrice:
		return ProblemPriceTooHigh
	}
	if !submit {
		return ProblemNone
	}
	if f.PriceBdt <= 0 {
		return ProblemNoPrice
	}
	for _, c := range []struct {
		slot    string
		problem Problem
	}{{"front", ProblemNeedFront}, {"back", ProblemNeedBack}, {"damage", ProblemNeedDamagePhoto}} {
		if needsPhoto(f, c.slot) && !slices.Contains(f.Photos, c.slot) {
			return c.problem
		}
	}
	return ProblemNone
}
