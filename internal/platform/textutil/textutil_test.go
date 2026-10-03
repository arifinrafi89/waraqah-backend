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

func TestGraphemes(t *testing.T) {
	for s, want := range map[string]int{"": 0, "  abc  ": 3, "👍🏽": 1, "বই পড়ি": 5, "ক্ষ": 1, "ক্ষক্ষক্ষ": 3,
		"স্ত্রী": 1, "ক্ a": 3, "नमस्ते": 3} {
		if got := Graphemes(s); got != want {
			t.Errorf("Graphemes(%q) = %d, want %d", s, got, want)
		}
	}
}
