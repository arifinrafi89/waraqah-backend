package checkout

import "testing"

func cartOf(price int, ebook bool) []CartLine {
	return []CartLine{{UnitPriceBdt: price, Quantity: 1, IsEbook: ebook}}
}

// Ported from test/checkout_totals_test.dart.
func TestDeliveryFee(t *testing.T) {
	if got := Compute(Input{Lines: cartOf(500, false), InsideDhaka: true}).DeliveryFeeBdt; got != 60 {
		t.Errorf("inside Dhaka: %d", got)
	}
	out := Compute(Input{Lines: cartOf(500, false)})
	if out.DeliveryFeeBdt != 120 || out.TotalBdt() != 620 {
		t.Errorf("outside Dhaka: %+v", out)
	}
	if got := Compute(Input{Lines: cartOf(1500, false)}).DeliveryFeeBdt; got != 0 {
		t.Errorf("free from 1500: %d", got)
	}
	ebooks := Compute(Input{Lines: cartOf(400, true)})
	if ebooks.NeedsDelivery || ebooks.DeliveryFeeBdt != 0 {
		t.Errorf("ebooks only: %+v", ebooks)
	}
}

func TestCoupons(t *testing.T) {
	max150 := 150
	percent := Coupon{Kind: PercentOff, Value: 10, MaxDiscountBdt: &max150}
	amount := Coupon{Kind: AmountOff, Value: 100, MinOrderBdt: 1000}
	free := Coupon{Kind: FreeDelivery}
	off := func(price int, c Coupon) int {
		return Compute(Input{Lines: cartOf(price, false), Coupon: &c}).CouponDiscountBdt
	}
	for price, want := range map[int]int{590: 59, 1400: 140, 3000: 150} {
		if got := off(price, percent); got != want {
			t.Errorf("percent off %d: %d want %d", price, got, want)
		}
	}
	if off(999, amount) != 0 || off(1000, amount) != 100 {
		t.Error("amount off needs the minimum order")
	}
	got := Compute(Input{Lines: cartOf(500, false), Coupon: &free})
	if got.CouponDiscountBdt != 120 || got.TotalBdt() != 500 {
		t.Errorf("free delivery: %+v", got)
	}
}

// Ported from the totals cases of test/loyalty_test.dart.
func TestPointsComeOffTheBooks(t *testing.T) {
	got := Compute(Input{Lines: cartOf(590, false), InsideDhaka: true, PointsBalance: 217, UsePoints: true})
	if got.PointsDiscountBdt != 118 || got.TotalBdt() != 590+60-118 || got.BooksPaidBdt() != 472 {
		t.Errorf("points: %+v", got)
	}
	free := Coupon{Kind: FreeDelivery}
	if got := Compute(Input{Lines: cartOf(590, false), Coupon: &free}); got.BooksPaidBdt() != 590 {
		t.Errorf("free delivery must not lower what the books cost: %+v", got)
	}
}

// Ported from test/wallet_rules_test.dart.
func TestTheWalletPaysLast(t *testing.T) {
	totals := func(balance int, use bool) Totals {
		return Compute(Input{Lines: cartOf(500, false), InsideDhaka: true, WalletBalance: balance, UseWallet: use})
	}
	if a := totals(180, true); a.WalletBdt != 180 || a.TotalBdt() != 380 {
		t.Errorf("180: %+v", a)
	}
	if a := totals(900, true); a.WalletBdt != 560 || a.TotalBdt() != 0 {
		t.Errorf("900: %+v", a)
	}
	if totals(180, false).WalletBdt != 0 || totals(180, true).BeforeWalletBdt() != 560 {
		t.Error("wallet off / before wallet")
	}
}

func TestGiftWrapOnlyForDeliveredOrders(t *testing.T) {
	if got := Compute(Input{Lines: cartOf(500, false), InsideDhaka: true, GiftWrap: true}); got.GiftWrapBdt != 40 || got.TotalBdt() != 600 {
		t.Errorf("wrapped: %+v", got)
	}
	if got := Compute(Input{Lines: cartOf(500, true), GiftWrap: true}); got.GiftWrapBdt != 0 {
		t.Errorf("an eBook order is not wrapped: %+v", got)
	}
}
