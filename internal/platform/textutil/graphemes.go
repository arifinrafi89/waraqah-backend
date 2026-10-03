package textutil

import (
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
)

// linkers are the Indic viramas of Unicode 15.1's conjunct rule (GB9c, InCB=Linker): Devanagari,
// Bengali, Gujarati, Oriya, Telugu and Malayalam.
var linkers = map[rune]bool{0x094D: true, 0x09CD: true, 0x0ACD: true, 0x0B4D: true, 0x0C4D: true, 0x0D4D: true}

// Graphemes is Dart's `text.trim().characters.length`: user-perceived characters, so a Bangla
// conjunct like ক্ষ, a vowel sign or an emoji with its skin tone counts once (Bite and review
// limits). uniseg follows Unicode 15.0, so the 15.1 conjunct rule is added here: a cluster
// ending in a virama joins the next one when that starts with a letter of the same script.
func Graphemes(s string) int {
	g := uniseg.NewGraphemes(strings.TrimSpace(s))
	n := 0
	var joinBlock rune = -1 // the script block a pending virama joins into, or -1
	for g.Next() {
		runes := g.Runes()
		first := runes[0]
		if joinBlock < 0 || !unicode.IsLetter(first) || first&^0x7F != joinBlock {
			n++
		}
		joinBlock = -1
		for i := len(runes) - 1; i >= 0; i-- {
			r := runes[i]
			if linkers[r] {
				joinBlock = r &^ 0x7F
				break
			}
			if r != 0x200D && r != 0x200C && !unicode.Is(unicode.Mn, r) {
				break // only marks or joiners may follow the virama
			}
		}
	}
	return n
}
