package handledsale

import "math"

// The money in a Waraqah-handled sale: a port of SaleMath (sale_math.dart).
const (
	// DeliveryBdt is the courier delivery, paid by the buyer.
	DeliveryBdt = 80
	// FeeShare is Waraqah's share, taken from the seller's price.
	FeeShare = 0.05
	// MinFeeBdt is the smallest fee.
	MinFeeBdt = 10
	// MaxDisputeNote is the longest note a dispute may carry.
	MaxDisputeNote = 300
	// MaxDisputePhotos is the most photos a dispute may carry.
	MaxDisputePhotos = 3
)

// FeeFor is SaleMath.feeFor: 5% of the price, rounded half away from zero like Dart, at least 10.
func FeeFor(priceBdt int) int {
	fee := int(math.Round(float64(priceBdt) * FeeShare))
	return max(fee, MinFeeBdt)
}

// SellerGets is SaleMath.sellerGets.
func SellerGets(priceBdt int) int { return priceBdt - FeeFor(priceBdt) }

// BuyerPays is SaleMath.buyerPays.
func BuyerPays(priceBdt int) int { return priceBdt + DeliveryBdt }
