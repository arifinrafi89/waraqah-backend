package catalog

import (
	"regexp"
	"strings"
)

// PhoneticKey is a port of PhoneticKey (lib/features/catalog/data/sources/phonetic_key.dart): one
// rough sound key per name, so Bangla script, Banglish and English spellings meet. "sapiens",
// "sapiyens" and the Bangla spelling all give "spns".
func PhoneticKey(text string) string {
	latin := toLatin(cleanForKey(strings.ToLower(text)))
	var words []string
	for _, w := range strings.Split(latin, " ") {
		if f := foldWord(w); f != "" {
			words = append(words, f)
		}
	}
	return strings.Join(words, " ")
}

var (
	nonKeyChars = regexp.MustCompile("[^a-z0-9ঀ-৿]+")
	joiners     = strings.NewReplacer("\u200C", "", "\u200D", "", "'", "", "\u2019", "")
	// Precompose nukta letters, then drop ya-phala (the order is the Dart one).
	nuktas = strings.NewReplacer("য়", "য়", "ড়", "ড়", "ঢ়", "ঢ়", "্য", "")
)

// cleanForKey precomposes nukta letters, drops ya-phala and joiners, and turns punctuation into
// word breaks (apostrophes vanish: "o'reilly" is "oreilly").
func cleanForKey(text string) string {
	return nonKeyChars.ReplaceAllString(joiners.Replace(nuktas.Replace(text)), " ")
}

func toLatin(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch {
		case r >= 0x09E6 && r <= 0x09EF:
			b.WriteByte(byte('0' + (r - 0x09E6)))
		default:
			if s, ok := banglaToLatin[r]; ok {
				b.WriteString(s)
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

var foldPairs = [][2]string{
	{"kh", "k"}, {"gh", "g"}, {"ch", "c"}, {"jh", "j"}, {"th", "t"},
	{"dh", "d"}, {"ph", "f"}, {"bh", "b"}, {"sh", "s"},
	// "alchemist" is "alkemist": every c ends up a k.
	{"c", "k"}, {"z", "j"}, {"v", "b"}, {"q", "k"}, {"w", ""}, {"y", ""},
}

// foldWord folds the sounds of one word, collapses doubled letters before the vowels go (so
// "harari" is "hrr" and "harry" is "hr"), and keeps a leading vowel as "a".
func foldWord(word string) string {
	w := word
	for _, p := range foldPairs {
		w = strings.ReplaceAll(w, p[0], p[1])
	}
	if w == "" {
		return w
	}
	var collapsed []rune
	for _, r := range w {
		if n := len(collapsed); n > 0 && collapsed[n-1] == r {
			continue
		}
		collapsed = append(collapsed, r)
	}
	lead := ""
	if strings.ContainsRune("aeiou", collapsed[0]) {
		lead = "a"
	}
	var out strings.Builder
	out.WriteString(lead)
	for _, r := range collapsed {
		if !strings.ContainsRune("aeiou", r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// banglaToLatin maps each Bangla letter to Latin (an empty value drops it).
var banglaToLatin = buildBanglaMap(
	"ক=k খ=kh গ=g ঘ=gh ঙ=ng চ=ch ছ=ch জ=j ঝ=jh ঞ=n ট=t ঠ=th ড=d ঢ=dh " +
		"ণ=n ত=t থ=th দ=d ধ=dh ন=n প=p ফ=f ব=b ভ=bh ম=m য=j র=r ল=l " +
		"শ=sh ষ=sh স=s হ=h ড়=r ঢ়=r য়=y ৎ=t ং=ng ঃ=h ঁ= " +
		"্= ়= অ=a আ=a া=a ই=i ি=i ঈ=i ী=i উ=u ু=u ঊ=u ূ=u " +
		"ঋ=ri ৃ=ri এ=e ে=e ঐ=oi ৈ=oi ও=o ো=o ঔ=ou ৌ=ou")

func buildBanglaMap(spec string) map[rune]string {
	m := map[rune]string{}
	for _, pair := range strings.Split(spec, " ") {
		k, v, _ := strings.Cut(pair, "=")
		for _, r := range k {
			m[r] = v
			break
		}
	}
	return m
}

// Levenshtein counts the edits (insert, delete, swap one letter) that turn a into b.
func Levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		row := make([]int, len(br)+1)
		row[0] = i
		for j := 1; j <= len(br); j++ {
			swap := prev[j-1]
			if ar[i-1] != br[j-1] {
				swap++
			}
			row[j] = min(swap, min(row[j-1], prev[j])+1)
		}
		prev = row
	}
	return prev[len(br)]
}
