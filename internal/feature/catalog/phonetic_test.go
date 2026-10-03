package catalog

import "testing"

// Ported from test/phonetic_key_test.dart.
func TestSpellingsGiveOneKey(t *testing.T) {
	for _, s := range []string{"sapiens", "Sapiyens", "স্যাপিয়েন্স"} {
		if got := PhoneticKey(s); got != "spns" {
			t.Errorf("PhoneticKey(%q) = %q, want spns", s, got)
		}
	}
}

func TestKnownPairsMeet(t *testing.T) {
	pairs := [][2]string{
		{"atomic habits", "অ্যাটমিক হ্যাবিটস"},
		{"alkemist", "alchemist"},
		{"rahik", "raheeq"},
		{"humayun", "হুমায়ূন"},
		{"zero to one", "জিরো টু ওয়ান"},
		{"matilda", "মাটিল্ডা"},
		{"sherlock", "শার্লক"},
		{"bukhari", "বুখারী"},
		{"hobbit", "হবিট"},
		{"harry potter", "হ্যারি পটার"},
		{"salihin", "সালিহীন"},
		{"dhaka", "ঢাকা"},
		{"harari", "হারারি"},
		{"class 9", "class ৯"},
	}
	for _, p := range pairs {
		if PhoneticKey(p[0]) != PhoneticKey(p[1]) {
			t.Errorf("%q (%q) and %q (%q) should meet", p[0], PhoneticKey(p[0]), p[1], PhoneticKey(p[1]))
		}
	}
}

func TestPunctuationBreaksWordsApostrophesVanish(t *testing.T) {
	if got := PhoneticKey("Sapiens: A Brief"); got != "spns a brf" {
		t.Errorf("got %q", got)
	}
	if PhoneticKey("O'Reilly") != PhoneticKey("oreilly") {
		t.Error("apostrophe should vanish")
	}
}

func TestDifferentBooksKeepDifferentKeys(t *testing.T) {
	keys := map[string]bool{}
	for _, title := range []string{"Sapiens", "Satanic", "Atomic Habits", "Matilda", "Harari", "Harry"} {
		keys[PhoneticKey(title)] = true
	}
	if len(keys) != 6 {
		t.Errorf("keys collide: %v", keys)
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"spns", "spnj", 1}, {"", "abc", 3}, {"kitten", "sitting", 3}}
	for _, c := range cases {
		if got := Levenshtein(c.a, c.b); got != c.want {
			t.Errorf("Levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
