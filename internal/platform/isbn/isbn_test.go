package isbn

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"9789840001774", "9789840001774", true},
		{"978-984-0001-774", "9789840001774", true},
		{" 978 984 0001 774 ", "9789840001774", true},
		{"0-306-40615-2", "9780306406157", true}, // an ISBN-10 becomes its ISBN-13
		{"0306406152", "9780306406157", true},
		{"080442957X", "9780804429573", true}, // X check digit
		{"9789840001775", "", false},          // bad check digit
		{"1234567890123", "", false},          // not Bookland
		{"0306406153", "", false},
		{"12345", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := Normalize(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("Normalize(%q) = %q %v, want %q %v", c.in, got, ok, c.want, c.ok)
		}
	}
}
