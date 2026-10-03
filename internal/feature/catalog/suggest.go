package catalog

import (
	"regexp"
	"sort"
	"strings"
)

var banglaScript = regexp.MustCompile("[ঀ-৿]")

// suggestRank: 0 typed as written, 1 sounds like the start, 2 sounds like a later word.
func suggestRank(spelling, query, sound string) (int, bool) {
	if strings.HasPrefix(strings.ToLower(spelling), query) {
		return 0, true
	}
	key := PhoneticKey(spelling)
	if strings.HasPrefix(key, sound) {
		return 1, true
	}
	if strings.Contains(" "+key, " "+sound) {
		return 2, true
	}
	return 0, false
}

// Suggest is BookSuggestFakeApi._suggest: up to 5 book titles and author names for the query,
// best first, in the script the reader typed (a Bangla query gets Bangla titles when there are
// some). The query is trimmed and lower case.
func (s *Snapshot) Suggest(query string) []string {
	sound := PhoneticKey(query)
	if len(sound) < 2 {
		return []string{}
	}
	bangla := banglaScript.MatchString(query)
	type hit struct {
		rank  int
		shown string
	}
	var hits []hit
	consider := func(spellings []*string, shown string) {
		best, found := 0, false
		for _, sp := range spellings {
			if sp == nil {
				continue
			}
			if r, ok := suggestRank(*sp, query, sound); ok && (!found || r < best) {
				best, found = r, true
			}
		}
		if found {
			hits = append(hits, hit{best, shown})
		}
	}
	pick := func(en string, bn *string) string {
		if bangla && bn != nil {
			return *bn
		}
		return en
	}
	for _, b := range s.Books {
		if b.Hidden {
			continue
		}
		title := b.Title
		consider([]*string{&title, b.TitleBn, b.ShortTitle}, pick(b.Title, b.TitleBn))
	}
	for _, a := range s.Authors {
		name := a.Name
		consider([]*string{&name, a.NameBn}, pick(a.Name, a.NameBn))
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank < hits[j].rank
		}
		return hits[i].shown < hits[j].shown
	})
	seen := map[string]bool{}
	out := []string{}
	for _, h := range hits {
		if !seen[h.shown] && len(out) < 5 {
			seen[h.shown] = true
			out = append(out, h.shown)
		}
	}
	return out
}

// DidYouMean is BookSuggestFakeApi._didYouMean: the one title closest to the query, or nil. It
// compares keys with the whole title and with its first words (as many as the query has). Up to
// 2 edits away, 1 for keys under 5 letters; keys under 3 letters never guess.
func (s *Snapshot) DidYouMean(query string) *string {
	sound := PhoneticKey(query)
	if len(sound) < 3 {
		return nil
	}
	words := len(strings.Split(sound, " "))
	best := 3
	if len(sound) < 5 {
		best = 2
	}
	var title *string
	bangla := banglaScript.MatchString(query)
	for _, b := range s.Books {
		if b.Hidden {
			continue
		}
		for _, sp := range []*string{&b.Title, b.TitleBn, b.ShortTitle} {
			if sp == nil {
				continue
			}
			key := PhoneticKey(*sp)
			parts := strings.Split(key, " ")
			if len(parts) > words {
				parts = parts[:words]
			}
			start := strings.Join(parts, " ")
			edits := min(Levenshtein(sound, key), Levenshtein(sound, start))
			if edits < best {
				best = edits
				shown := b.Title
				if bangla && b.TitleBn != nil {
					shown = *b.TitleBn
				}
				title = &shown
			}
		}
	}
	return title
}
