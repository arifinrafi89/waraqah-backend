package bookrequest

import (
	"strings"
	"testing"
)

// Ported from test/book_request_test.dart.
func TestARequestNeedsATitleASensiblePriceAndAShortNote(t *testing.T) {
	zero, one := 0, 1
	long := strings.Repeat("x", 301)
	cases := []struct {
		name string
		d    Draft
		want bool
	}{
		{"blank title", Draft{Title: " "}, false},
		{"one letter", Draft{Title: "x"}, false},
		{"price zero", Draft{Title: "SICP", MaxPriceBdt: &zero}, false},
		{"note too long", Draft{Title: "SICP", Note: &long}, false},
		{"title too long", Draft{Title: strings.Repeat("x", MaxTitle+1)}, false},
		{"fine", Draft{Title: "SICP"}, true},
		{"fine with a price", Draft{Title: "SICP", MaxPriceBdt: &one}, true},
	}
	for _, c := range cases {
		if Valid(c.d) != c.want {
			t.Errorf("%s: want %v", c.name, c.want)
		}
	}
}

func TestAListingMatchesByCatalogBookOrByTitleWords(t *testing.T) {
	if !Matches("Anything", "bk-atomic", "Atomic Habits", "bk-atomic") {
		t.Error("same catalog book")
	}
	if !Matches("clean code", "", "Clean Code", "") {
		t.Error("same words")
	}
	if Matches("clean code", "", "Atomic Habits", "") {
		t.Error("different words")
	}
	if !Matches("Clean", "", "The Clean Code Handbook", "") || !Matches("The Clean Code Handbook", "", "Clean", "") {
		t.Error("either title may contain the other")
	}
	if Matches("!!!", "", "Clean Code", "") {
		t.Error("a title of symbols matches nothing")
	}
}
