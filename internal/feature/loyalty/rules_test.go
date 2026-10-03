package loyalty

import "testing"

// Ported from the rules case of test/loyalty_test.dart.
func TestEarningAndSpendingRules(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"earned for 1240", EarnedFor(1240), 12},
		{"earned below 100", EarnedFor(99), 0},
		{"earned for a negative amount", EarnedFor(-5), 0},
		{"below the minimum to spend", Usable(49, 1000), 0},
		{"20 percent of 590", Usable(217, 590), 118},
		{"balance is the limit", Usable(60, 1000), 60},
		{"exactly the minimum", Usable(50, 1000), 50},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %d want %d", c.name, c.got, c.want)
		}
	}
}
