package assistant

import (
	"regexp"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
)

// Basket is AssistantBasketModel: the Editions to add to the cart and what they cost together.
type Basket struct {
	EditionIDs []string `json:"editionIds"`
	TotalBdt   int      `json:"totalBdt"`
}

// Reply is AssistantReplyModel: the words and the catalog Books recommended, by id.
type Reply struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	BookIDs []string `json:"bookIds"`
	Basket  *Basket  `json:"basket,omitempty"`

	intent Intent // what the reader asked for, for Gemini's wording; never sent
	picks  []Pick
}

var banglaLetter = regexp.MustCompile(`[অ-হ]`)

// digits is AssistantBrain._digits: Bangla replies write numbers in Bangla digits.
func digits(text string) string {
	if !banglaLetter.MatchString(text) {
		return text
	}
	var b strings.Builder
	for _, r := range text {
		if r >= '0' && r <= '9' {
			b.WriteRune([]rune(banglaDigits)[r-'0'])
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func reply(text string, bookIDs []string) Reply {
	if bookIDs == nil {
		bookIDs = []string{}
	}
	return Reply{Text: digits(text), BookIDs: bookIDs}
}

// Greeting is AssistantBrain.greeting.
func Greeting(lang string) Reply { return reply(greetingText(lang == "bn"), nil) }

// Ask is AssistantBrain.ask: works out what the reader asked for, picks from the catalog (books,
// in storefront order) and answers in their language. The id is set by the caller.
func Ask(books []catalog.Book, prompt string, history []string, lang string) Reply {
	bn := lang == "bn"
	words := strings.ToLower(prompt)
	has := func(terms ...string) bool { return hasAny(words, terms...) }
	// Selling is about the reader's own books, not the catalog.
	if has("sell", "বিক্রি") {
		return reply(sellText(bn), nil)
	}
	in := Parse(prompt, history)
	if in.SearchesBooks() {
		picks := PicksFor(books, in)
		if len(picks) == 0 {
			return reply(noneText(bn), nil)
		}
		ids := make([]string, len(picks))
		for i, p := range picks {
			ids[i] = p.Book.ID
		}
		if !in.Basket {
			r := reply(foundText(bn, in), ids)
			r.intent, r.picks = in, picks
			return r
		}
		b := &Basket{EditionIDs: make([]string, len(picks))}
		for i, p := range picks {
			b.EditionIDs[i] = p.Edition.ID
			b.TotalBdt += p.Edition.PriceBdt
		}
		r := reply(basketText(bn, in, len(picks), b.TotalBdt), ids)
		r.Basket, r.intent, r.picks = b, in, picks
		return r
	}
	switch {
	case has("hello", "hi ", "salam", "সালাম"):
		return reply(helloText(bn), nil)
	case has("exam", "study", "prep", "পরীক্ষা"):
		return reply(studyText(bn), nil)
	case has("thank", "ধন্যবাদ"):
		return reply(thanksText(bn), nil)
	}
	return reply(fallbackText(bn), nil)
}
