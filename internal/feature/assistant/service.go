// Package assistant is the AI assistant: a rule-based port of the app's assistant brain that picks
// Books from Waraqah's own catalog, with Gemini (optional) only wording the reply around the Books
// already picked. Port of assistant_parser.dart, assistant_catalog.dart, assistant_replies.dart
// and assistant_brain.dart.
package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/gemini"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/ids"
)

// Catalog is the part of catalog.Books the assistant reads: the whole catalog in storefront order.
type Catalog interface {
	Snapshot(ctx context.Context) (*catalog.Snapshot, error)
}

// maxWorded is the longest reply Gemini may give; anything longer keeps the rule-based text.
const maxWorded = 600

// Service answers the assistant endpoints.
type Service struct {
	Catalog Catalog
	Gemini  gemini.Client // nil without GEMINI_API_KEY: rule-based replies only
	Log     *slog.Logger
}

// Greeting is the opening line in the reader's language.
func (s *Service) Greeting(lang string) Reply {
	r := Greeting(lang)
	r.ID = ids.New("ai")
	return r
}

// Ask answers a prompt. Books, prices and stock always come from the catalog; with Gemini the
// words around the picked Books may be rephrased, never the Books themselves.
func (s *Service) Ask(ctx context.Context, prompt string, history []string, lang string) (Reply, error) {
	snap, err := s.Catalog.Snapshot(ctx)
	if err != nil {
		return Reply{}, err
	}
	r := Ask(snap.Books, prompt, history, lang)
	r.ID = ids.New("ai")
	if s.Gemini != nil && len(r.picks) > 0 {
		if text, ok := s.word(ctx, prompt, lang, r); ok {
			r.Text = text
		}
	}
	return r, nil
}

// word asks Gemini for a friendlier reply about exactly the picked Books. Any failure, an empty or
// too long answer keeps the rule-based text.
func (s *Service) word(ctx context.Context, prompt, lang string, r Reply) (string, bool) {
	var list strings.Builder
	for _, p := range r.picks {
		fmt.Fprintf(&list, "- %s by %s, ৳%d\n", p.Book.Title, p.Book.Author, p.Edition.PriceBdt)
	}
	language := "English"
	if lang == "bn" {
		language = "Bangla"
	}
	ask := fmt.Sprintf("You are the book assistant of Waraqah, a Bangladeshi bookshop. A reader asked: %q.\n"+
		"Our catalog search already picked these books (the app shows them as cards below your words):\n%s"+
		"Write one or two short, warm sentences in %s introducing these picks. Do not name, add or suggest any "+
		"other book, do not state prices or stock, no lists, no markdown.", prompt, list.String(), language)
	if r.Basket != nil {
		ask += fmt.Sprintf(" Mention that all %d books together cost ৳%d and can be added to the cart below.", len(r.picks), r.Basket.TotalBdt)
	}
	text, err := s.Gemini.Word(ctx, ask)
	if err != nil {
		s.Log.Warn("gemini wording failed, keeping the rule-based reply", "error", err)
		return "", false
	}
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) > maxWorded {
		return "", false
	}
	return digits(text), true
}
