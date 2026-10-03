package p2p

import "testing"

// Ported from test/listing_rules_test.dart and test/fair_price_test.dart.
func TestDraftNeedsTitleReviewNeedsPhotos(t *testing.T) {
	book := Fields{Title: "SICP", PriceBdt: 300}
	covers := book
	covers.Photos = []string{"front", "back"}
	withFlag := func(f Fields, flag string) Fields { f.Flags = []string{flag}; return f }

	cases := []struct {
		name   string
		f      Fields
		submit bool
		want   Problem
	}{
		{"blank title", Fields{Title: " "}, false, ProblemNoTitle},
		{"draft is fine", book, false, ProblemNone},
		{"price too high", Fields{Title: "x", PriceBdt: 60000}, false, ProblemPriceTooHigh},
		{"no price", Fields{Title: "x", Photos: []string{"front", "back"}}, true, ProblemNoPrice},
		{"no front", book, true, ProblemNeedFront},
		{"no back", Fields{Title: "x", PriceBdt: 1, Photos: []string{"front"}}, true, ProblemNeedBack},
		{"ready", covers, true, ProblemNone},
		{"damage needs a photo", withFlag(covers, "damage"), true, ProblemNeedDamagePhoto},
	}
	for _, c := range cases {
		if got := Check(c.f, c.submit); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
	long := make([]byte, MaxTitle+1)
	for i := range long {
		long[i] = 'a'
	}
	if Check(Fields{Title: string(long)}, false) != ProblemTitleTooLong {
		t.Error("title over 120")
	}
	long = append(long, long...)
	long = append(long, long...)
	long = append(long, long...)
	if Check(Fields{Title: "x", Note: string(long)}, false) != ProblemNoteTooLong {
		t.Error("note over 500")
	}
	if CanEdit(Live) || !CanEdit(Draft) || !CanEdit(ChangesRequested) || !CanEdit(Rejected) || CanEdit(InReview) {
		t.Error("only draft, changesRequested and rejected can be edited")
	}
}

func TestFairRangeFollowsPriceConditionAndFlags(t *testing.T) {
	if f := FairPriceOf(500, "likeNew", 0); f == nil || f.LowBdt != 300 || f.HighBdt != 380 {
		t.Errorf("like new: %v", f)
	}
	if f := FairPriceOf(500, "good", 1); f == nil || f.LowBdt != 180 || f.HighBdt != 250 {
		t.Errorf("good with a flag: %v", f)
	}
	if FairPriceOf(0, "good", 0) != nil {
		t.Error("no new price, no range")
	}
}

func TestAskingPriceVerdict(t *testing.T) {
	fair := FairPrice{300, 380, 500}
	for asking, want := range map[int]string{250: VerdictLow, 320: VerdictFair, 450: VerdictHigh, 500: VerdictAboveNew} {
		if got := fair.Verdict(asking); got != want {
			t.Errorf("%d: got %s want %s", asking, got, want)
		}
	}
}
