package assistant

import (
	"slices"
	"sort"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
)

// Pick is AssistantPick: a catalog Book and the Edition the assistant would buy, the cheapest
// orderable one in the language and format asked for.
type Pick struct {
	Book    catalog.Book
	Edition catalog.Edition
}

// PicksFor is assistantPicksFor (assistant_catalog.dart): the Books that fit the intent, on the
// storefront and within the budget. A basket gets the cheapest first, added while the total stays
// in budget (up to eight); otherwise the best rated four. books is the catalog in storefront order.
func PicksFor(books []catalog.Book, in Intent) []Pick {
	var picks []Pick
	for _, b := range books {
		if b.Hidden || !fits(b, in) {
			continue
		}
		if e, ok := edition(b, in); ok {
			picks = append(picks, Pick{Book: b, Edition: e})
		}
	}
	if !in.Basket {
		var out []Pick
		for _, p := range picks {
			if in.MaxPrice == nil || p.Edition.PriceBdt <= *in.MaxPrice {
				out = append(out, p)
			}
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i].Book.Rating > out[j].Book.Rating })
		return out[:min(4, len(out))]
	}
	sort.SliceStable(picks, func(i, j int) bool { return picks[i].Edition.PriceBdt < picks[j].Edition.PriceBdt })
	var basket []Pick
	total := 0
	for _, p := range picks {
		if len(basket) == 8 {
			break
		}
		if in.MaxPrice != nil && total+p.Edition.PriceBdt > *in.MaxPrice {
			continue
		}
		basket = append(basket, p)
		total += p.Edition.PriceBdt
	}
	return basket
}

func fits(b catalog.Book, in Intent) bool {
	text := strings.ToLower(strings.Join(append([]string{b.Title, b.Author}, b.Tags...), " "))
	if in.IsIslamic() && b.CategoryID != "cat-islamic-studies" {
		return false
	}
	if in.ClassLevel != nil && !slices.Contains(b.Classes, *in.ClassLevel) {
		return false
	}
	if in.Exam != "" && !slices.Contains(b.Exams, in.Exam) {
		return false
	}
	switch in.Kind {
	case Quran:
		return strings.Contains(text, "quran") || strings.Contains(text, "tafsir")
	case Hadith:
		return strings.Contains(text, "hadith")
	case Seerah:
		return strings.Contains(text, "seerah") || strings.Contains(text, "sealed nectar")
	case IslamicHistory:
		return strings.Contains(text, "history")
	case AuthorSearch:
		return strings.Contains(strings.ToLower(b.Author), strings.ToLower(in.Query))
	}
	return true
}

func edition(b catalog.Book, in Intent) (catalog.Edition, bool) {
	var best catalog.Edition
	found := false
	for _, e := range b.Editions {
		if !e.IsOrderable() || (in.Language != "" && e.Language != in.Language) || (in.Format != "" && e.Format != in.Format) {
			continue
		}
		if !found || e.PriceBdt < best.PriceBdt {
			best, found = e, true
		}
	}
	return best, found
}
