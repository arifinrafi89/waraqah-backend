// Package checkout turns the cart into an order: coupons, the totals, points, the wallet, gifts.
package checkout

import (
	"github.com/arifinrafi89/waraqah-backend/internal/feature/loyalty"
)

// Gift wrap and delivery prices (checkout_totals.dart, gift.dart).
const (
	FreeDeliveryFromBdt = 1500
	GiftWrapFeeBdt      = 40
	GiftMaxMessage      = 150
	deliveryInsideBdt   = 60
	deliveryOutsideBdt  = 120
)

// Coupon kinds, as the Dart enum values.
const (
	PercentOff   = "percentOff"
	AmountOff    = "amountOff"
	FreeDelivery = "freeDelivery"
)

// Coupon is the part of a coupon that the totals need.
type Coupon struct {
	Kind           string
	Value          int
	MinOrderBdt    int
	MaxDiscountBdt *int
}

// CartLine is the part of a cart line that the totals need.
type CartLine struct {
	UnitPriceBdt int
	Quantity     int
	IsEbook      bool
}

// Input is what an order is priced from.
type Input struct {
	Lines         []CartLine
	InsideDhaka   bool
	Coupon        *Coupon
	PointsBalance int
	UsePoints     bool
	GiftWrap      bool
	WalletBalance int
	UseWallet     bool
}

// Totals is CheckoutTotals: what an order costs, worked out the same way on the phone and the server:
//   - delivery is 60 taka inside Dhaka, 120 outside, free from 1,500, and not charged when the
//     order is only eBooks
//   - a coupon only works once the subtotal reaches its minimum
//   - points pay for part of the books, never delivery
//   - gift wrap is 40 taka, for orders that get delivered
//   - the wallet pays last, for whatever is left, delivery included
type Totals struct {
	SubtotalBdt       int
	DeliveryFeeBdt    int
	CouponDiscountBdt int
	NeedsDelivery     bool
	PointsDiscountBdt int
	CouponOnBooksBdt  int
	GiftWrapBdt       int
	WalletBdt         int
}

// Subtotal is the sum of the lines.
func Subtotal(lines []CartLine) int {
	sum := 0
	for _, l := range lines {
		sum += l.UnitPriceBdt * l.Quantity
	}
	return sum
}

// Compute is CheckoutTotals.of.
func Compute(in Input) Totals {
	subtotal := Subtotal(in.Lines)
	needsDelivery := false
	for _, l := range in.Lines {
		if !l.IsEbook {
			needsDelivery = true
		}
	}
	fee := 0
	if needsDelivery && subtotal < FreeDeliveryFromBdt {
		fee = deliveryOutsideBdt
		if in.InsideDhaka {
			fee = deliveryInsideBdt
		}
	}
	couponOff := 0
	if in.Coupon != nil {
		couponOff = discount(*in.Coupon, subtotal, fee)
	}
	booksOff := couponOff
	if in.Coupon != nil && in.Coupon.Kind == FreeDelivery {
		booksOff = 0
	}
	wrap := 0
	if in.GiftWrap && needsDelivery {
		wrap = GiftWrapFeeBdt
	}
	points := 0
	if in.UsePoints {
		points = loyalty.Usable(in.PointsBalance, subtotal-booksOff)
	}
	beforeWallet := subtotal + fee + wrap - couponOff - points
	wallet := 0
	if in.UseWallet {
		wallet = min(max(in.WalletBalance, 0), beforeWallet)
	}
	return Totals{SubtotalBdt: subtotal, DeliveryFeeBdt: fee, CouponDiscountBdt: couponOff, NeedsDelivery: needsDelivery,
		CouponOnBooksBdt: booksOff, GiftWrapBdt: wrap, PointsDiscountBdt: points, WalletBdt: wallet}
}

// BeforeWalletBdt is what the order comes to before the wallet.
func (t Totals) BeforeWalletBdt() int {
	return t.SubtotalBdt + t.DeliveryFeeBdt + t.GiftWrapBdt - t.CouponDiscountBdt - t.PointsDiscountBdt
}

// TotalBdt is what is left to pay with the chosen method.
func (t Totals) TotalBdt() int { return t.BeforeWalletBdt() - t.WalletBdt }

// BooksPaidBdt is what is paid for the books themselves; points are earned on this.
func (t Totals) BooksPaidBdt() int {
	return max(0, t.SubtotalBdt-t.CouponOnBooksBdt-t.PointsDiscountBdt)
}

func discount(c Coupon, subtotal, fee int) int {
	if subtotal < c.MinOrderBdt {
		return 0
	}
	switch c.Kind {
	case PercentOff:
		limit := subtotal
		if c.MaxDiscountBdt != nil {
			limit = *c.MaxDiscountBdt
		}
		return min(subtotal*c.Value/100, limit)
	case AmountOff:
		return min(c.Value, subtotal)
	case FreeDelivery:
		return fee
	}
	return 0
}
