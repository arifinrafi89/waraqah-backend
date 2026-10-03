// Package loyalty keeps the Waraqah points of a reader and the rules for earning and spending them.
package loyalty

// Port of LoyaltyRules (lib/features/loyalty/domain/entities/loyalty_rules.dart):
//   - earn 1 point for every 100 taka paid for books (delivery does not count)
//   - spend them at checkout, 1 point = 1 taka off, once you have MinToSpend
//   - points can pay for at most a fifth of the price of the books
const (
	BdtPerPointEarned = 100
	MinToSpend        = 50
	// maxSharePercent is the share of the books price that points can pay for.
	maxSharePercent = 20
)

// EarnedFor is the points earned for paying booksBdt for books.
func EarnedFor(booksBdt int) int { return max(0, booksBdt) / BdtPerPointEarned }

// Usable is the most points an order of booksBdt can use from balance; 0 below MinToSpend.
func Usable(balance, booksBdt int) int {
	if balance < MinToSpend {
		return 0
	}
	return min(balance, max(0, booksBdt)*maxSharePercent/100)
}
