package inbox

import (
	"strings"
	"testing"
)

// Ported from test/inbox_rules_test.dart and test/seller_ratings_test.dart.
func TestOfferRules(t *testing.T) {
	cases := []struct {
		amount, asking int
		negotiable     bool
		want           OfferProblem
	}{
		{0, 300, true, OfferMissing},
		{-5, 300, true, OfferMissing},
		{301, 300, true, OfferAboveAsk},
		{250, 300, true, OfferOK},
		{300, 300, true, OfferOK},
		{250, 300, false, OfferFixedPrice},
		{300, 300, false, OfferOK},
	}
	for _, c := range cases {
		if got := CheckOffer(c.amount, c.asking, c.negotiable); got != c.want {
			t.Errorf("%+v: got %q", c, got)
		}
	}
}

func TestRatingRules(t *testing.T) {
	for stars, ok := range map[int]bool{0: false, 1: true, 5: true, 6: false} {
		if CheckRating(stars, "") != ok {
			t.Errorf("%d stars: %v", stars, !ok)
		}
	}
	if !CheckRating(5, strings.Repeat("x", MaxComment)) || CheckRating(5, strings.Repeat("x", MaxComment+1)) {
		t.Error("a comment is at most 300 characters")
	}
	if !CheckRating(4, "  "+strings.Repeat("x", MaxComment)+"  ") {
		t.Error("the comment is trimmed first")
	}
}

func TestHandover(t *testing.T) {
	if !ValidHandover("meetup") || !ValidHandover("courier") || ValidHandover("meetInPerson") {
		t.Error("offer handover is meetup or courier")
	}
}
