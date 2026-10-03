package textutil

import "testing"

func TestLenMatchesDart(t *testing.T) {
	cases := map[string]int{
		"":         0,
		"Nadia":    5,
		"নাদিয়া":  7, // Bangla letters and vowel signs are one UTF-16 unit each
		"😀":        2, // an emoji is a surrogate pair in Dart
		"a😀b":      4,
		"  trim  ": 8,
	}
	for in, want := range cases {
		if got := Len(in); got != want {
			t.Errorf("Len(%q) = %d, want %d", in, got, want)
		}
	}
	if TrimLen("  trim  ") != 4 {
		t.Error("TrimLen")
	}
}
