package sellback

import "testing"

// Ported from test/sell_back_test.dart ("the quote follows the new price, condition and flags").
func TestQuoteAndResellPrice(t *testing.T) {
	cases := []struct {
		name      string
		got, want int
	}{
		{"quote(1000, likeNew)", Quote(1000, "likeNew", 0), 350},
		{"quote(1000, likeNew, flags: 1)", Quote(1000, "likeNew", 1), 300},
		{"quote(100, acceptable)", Quote(100, "acceptable", 0), 30},
		{"resellPrice(1000, good)", ResellPrice(1000, "good"), 450},
		{"quote(430, good) as the zero-to-one golden", Quote(430, "good", 0), 110},
		{"quote(520, veryGood) as the Sapiens seed", Quote(520, "veryGood", 0), 160},
		{"many flags never go under 5%", Quote(1000, "acceptable", 9), 50},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

// Ported from test/finished_it_test.dart (the unit part).
func TestFinishedItOffers(t *testing.T) {
	price := 500
	o := FinishedItOffersOf(&price)
	if o == nil {
		t.Fatal("no offers for 500")
	}
	if o.Readers.LowBdt != 300 || o.Readers.HighBdt != 380 {
		t.Errorf("readers = %d..%d, want 300..380", o.Readers.LowBdt, o.Readers.HighBdt)
	}
	if o.SellBackBdt != 180 {
		t.Errorf("sell back = %d, want 180", o.SellBackBdt)
	}
	if FinishedItOffersOf(nil) != nil {
		t.Error("no new price should give no offers")
	}
}
