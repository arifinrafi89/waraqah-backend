package p2p

import "math"

// Port of FairPrice (lib/features/p2p/domain/entities/fair_price.dart): what a used copy usually
// sells for, from the new price and the condition.

// Verdicts of an asking price against the fair range.
const (
	VerdictLow         = "low"
	VerdictFair        = "fair"
	VerdictHigh        = "high"
	VerdictAboveNew    = "aboveNew"
	perFlag            = 0.05
	minShare, maxShare = 0.1, 1.0
)

// shares are the lowest and highest share of the new price, by condition.
var shares = map[string][2]float64{
	"likeNew":    {0.60, 0.75},
	"veryGood":   {0.50, 0.65},
	"good":       {0.40, 0.55},
	"acceptable": {0.25, 0.40},
}

// FairPrice is the range for a copy, rounded to 10 taka.
type FairPrice struct{ LowBdt, HighBdt, NewPriceBdt int }

func roundHalfUp(v float64) float64 { return math.Floor(v + 0.5) }

// FairPriceOf is FairPrice.of: nil without a new price to compare with.
func FairPriceOf(newPriceBdt int, condition string, flags int) *FairPrice {
	if newPriceBdt <= 0 {
		return nil
	}
	s, ok := shares[condition]
	if !ok {
		return nil
	}
	off := float64(flags) * perFlag
	price := func(share float64) int {
		clamped := math.Min(math.Max(share-off, minShare), maxShare)
		return int(roundHalfUp(float64(newPriceBdt)*clamped/10)) * 10
	}
	return &FairPrice{LowBdt: price(s[0]), HighBdt: price(s[1]), NewPriceBdt: newPriceBdt}
}

// Verdict compares an asking price with the range.
func (f FairPrice) Verdict(askingBdt int) string {
	switch {
	case askingBdt >= f.NewPriceBdt:
		return VerdictAboveNew
	case askingBdt > f.HighBdt:
		return VerdictHigh
	case askingBdt < f.LowBdt:
		return VerdictLow
	}
	return VerdictFair
}
