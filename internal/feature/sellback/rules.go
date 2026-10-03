package sellback

import "math"

// Port of SellBackRules (sell_back_rules.dart): Sell Back prices, from the cheapest printed
// Edition's new price.

// Conditions are the BookCondition names.
var Conditions = []string{"likeNew", "veryGood", "good", "acceptable"}

// payShare is what Waraqah pays, as a share of new, by condition.
var payShare = map[string]float64{"likeNew": 0.35, "veryGood": 0.30, "good": 0.25, "acceptable": 0.15}

// resellShare is what Waraqah resells it for as Certified Used.
var resellShare = map[string]float64{"likeNew": 0.65, "veryGood": 0.55, "good": 0.45, "acceptable": 0.35}

const (
	// PerFlag is what each flag (highlighting, notes, damage) takes off what is paid.
	PerFlag = 0.05
	// MinQuoteBdt is the smallest quote.
	MinQuoteBdt = 30
	// MinAddress is the shortest pickup address.
	MinAddress = 5
)

// round10 is Dart's `(bdt / 10).round() * 10`: half away from zero.
func round10(bdt float64) int { return int(math.Round(bdt/10)) * 10 }

// Quote is SellBackRules.quote: the instant quote, rounded to 10 taka, never under 30.
func Quote(newPriceBdt int, condition string, flags int) int {
	share := math.Min(math.Max(payShare[condition]-float64(flags)*PerFlag, 0.05), 1.0)
	return max(round10(float64(newPriceBdt)*share), MinQuoteBdt)
}

// ResellPrice is SellBackRules.resellPrice: the Certified Used price after grading.
func ResellPrice(newPriceBdt int, condition string) int {
	return round10(float64(newPriceBdt) * resellShare[condition])
}

func validCondition(c string) bool {
	_, ok := payShare[c]
	return ok
}
