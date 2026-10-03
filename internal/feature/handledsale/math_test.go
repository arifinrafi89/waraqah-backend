package handledsale

import "testing"

// Ported from test/handled_sale_test.dart ("the buyer pays delivery; Waraqah keeps 5% (at least ৳10)").
func TestSaleMath(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"buyerPays(320)", BuyerPays(320), 400},
		{"feeFor(300)", FeeFor(300), 15},
		{"feeFor(100)", FeeFor(100), 10},
		{"sellerGets(300)", SellerGets(300), 285},
		{"feeFor(450) rounds up", FeeFor(450), 23},
		{"feeFor(380)", FeeFor(380), 19},
		{"feeFor(220) rounds half up", FeeFor(220), 11},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}
