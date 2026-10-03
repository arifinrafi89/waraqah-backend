package reviews

import (
	"strings"
	"testing"
)

// The rules part of test/reviews_api_test.dart.
func TestReviewRules(t *testing.T) {
	cases := []struct {
		stars int
		text  string
		want  Problem
	}{
		{0, "", ProblemNoStars},
		{6, "great", ProblemNoStars},
		{1, "", ProblemNone},
		{5, strings.Repeat("a", 1000), ProblemNone},
		{5, strings.Repeat("a", 1001), ProblemTooLong},
		{4, strings.Repeat("ক্ষ", 1000), ProblemNone},
	}
	for _, c := range cases {
		if got := Check(c.stars, c.text); got != c.want {
			t.Errorf("Check(%d, %d chars) = %q, want %q", c.stars, len(c.text), got, c.want)
		}
	}
}

func TestAverageRoundsToOneDecimal(t *testing.T) {
	for _, c := range []struct {
		stars []int
		want  float64
	}{{nil, 0}, {[]int{5, 4}, 4.5}, {[]int{5, 4, 5}, 4.7}, {[]int{4, 4, 5}, 4.3}} {
		sum := 0
		for _, s := range c.stars {
			sum += s
		}
		avg := 0.0
		if len(c.stars) > 0 {
			avg = float64(sum) / float64(len(c.stars))
		}
		if got := roundRating(avg); got != c.want {
			t.Errorf("%v: %v, want %v", c.stars, got, c.want)
		}
	}
}
