package assistant_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/assistant"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func books(t *testing.T, e *testenv.Env) *catalog.Snapshot {
	t.Helper()
	snap, err := e.Deps.Catalog.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

// Ported from test/ai_assistant_test.dart ("the assistant recommends only catalog Books that fit").
func TestRecommendsOnlyCatalogBooksThatFit(t *testing.T) {
	e := testenv.New(t)
	snap := books(t, e)
	hadith := assistant.Ask(snap.Books, "Hadith collections", nil, "en")
	if len(hadith.BookIDs) == 0 {
		t.Fatal("no hadith books")
	}
	for _, id := range hadith.BookIDs {
		b, _ := snap.Book(id)
		if b.CategoryID != "cat-islamic-studies" || b.Hidden {
			t.Errorf("%s does not fit", id)
		}
	}
	cheap := assistant.Ask(snap.Books, "Books under ৳500", nil, "en")
	if len(cheap.BookIDs) == 0 || !strings.Contains(cheap.Text, "Waraqah's catalog") {
		t.Fatalf("cheap: %+v", cheap)
	}
	for _, id := range cheap.BookIDs {
		if b, _ := snap.Book(id); b.FromPriceBdt() > 500 {
			t.Errorf("%s costs %d", id, b.FromPriceBdt())
		}
	}
}

// Ported from test/ai_assistant_test.dart ("it answers small talk without Books, in Bangla too").
func TestSmallTalk(t *testing.T) {
	e := testenv.New(t)
	snap := books(t, e)
	hi := assistant.Ask(snap.Books, "salam!", nil, "en")
	if len(hi.BookIDs) != 0 || !strings.HasPrefix(hi.Text, "Wa alaikum") {
		t.Errorf("hi: %+v", hi)
	}
	if !strings.Contains(assistant.Greeting("bn").Text, "ওয়ারাকাহ") {
		t.Error("Bangla greeting")
	}
	if got := assistant.Ask(snap.Books, "sell my old books", nil, "en"); !strings.Contains(got.Text, "P2P") {
		t.Errorf("sell: %q", got.Text)
	}
}

// Ported from test/smarter_ai_test.dart (baskets and Bangla Editions).
func TestBasketsAndLanguages(t *testing.T) {
	e := testenv.New(t)
	snap := books(t, e)
	r := assistant.Ask(snap.Books, "Books for Class 9 under ৳1,000", nil, "en")
	if r.Basket == nil || len(r.Basket.EditionIDs) == 0 {
		t.Fatalf("class 9 basket: %+v", r)
	}
	total := 0
	for _, id := range r.Basket.EditionIDs {
		book, ed, ok, _ := e.Deps.Catalog.FindEdition(context.Background(), id)
		if !ok || !containsInt(book.Classes, 9) {
			t.Errorf("%s is not for Class 9", id)
		}
		total += ed.PriceBdt
	}
	if r.Basket.TotalBdt != total || total > 1000 || !strings.Contains(r.Text, "basket of books for Class 9") {
		t.Errorf("basket total %d of %d: %q", r.Basket.TotalBdt, total, r.Text)
	}
	seerah := assistant.Ask(snap.Books, "short seerah for beginners in Bangla", nil, "en")
	if len(seerah.BookIDs) == 0 || !strings.Contains(seerah.Text, "in Bangla") {
		t.Fatalf("seerah: %+v", seerah)
	}
	for _, id := range seerah.BookIDs {
		b, _ := snap.Book(id)
		bangla := false
		for _, ed := range b.Editions {
			bangla = bangla || ed.Language == "bangla"
		}
		if !bangla {
			t.Errorf("%s has no Bangla Edition", id)
		}
	}
	bn := assistant.Ask(snap.Books, "৯ম শ্রেণির বই ১০০০ টাকার মধ্যে", nil, "bn")
	if bn.Basket == nil || strings.ContainsAny(bn.Text, "0123456789") {
		t.Errorf("Bangla basket writes Bangla digits: %q", bn.Text)
	}
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

type fakeGemini struct {
	text string
	err  error
	asks []string
}

func (f *fakeGemini) Word(_ context.Context, prompt string) (string, error) {
	f.asks = append(f.asks, prompt)
	return f.text, f.err
}

func TestEndpointsAndGeminiWording(t *testing.T) {
	e := testenv.New(t)
	if got := e.Call("", "GET", "/assistant/greeting?lang=en", nil).Obj(t); !strings.HasPrefix(got["text"].(string), "Assalamu") {
		t.Errorf("greeting: %v", got)
	}
	if r := e.Call("", "POST", "/assistant/ask", map[string]any{"prompt": "hi"}); r.Status != 401 {
		t.Errorf("guest ask: %d", r.Status)
	}
	plain := e.Call("reader", "POST", "/assistant/ask", map[string]any{"prompt": "Hadith collections", "lang": "en"}).Obj(t)
	g := &fakeGemini{text: "  A warm line about these hadith books.  "}
	e.Deps.Assistant.Gemini = g
	worded := e.Call("reader", "POST", "/assistant/ask", map[string]any{"prompt": "Hadith collections", "lang": "en"}).Obj(t)
	if worded["text"] != "A warm line about these hadith books." || len(g.asks) != 1 {
		t.Errorf("worded: %v", worded["text"])
	}
	if len(worded["bookIds"].([]any)) != len(plain["bookIds"].([]any)) {
		t.Error("Gemini never changes the books")
	}
	g.text, g.err = "", context.DeadlineExceeded
	if got := e.Call("reader", "POST", "/assistant/ask", map[string]any{"prompt": "Hadith collections", "lang": "en"}).Obj(t); got["text"] != plain["text"] {
		t.Errorf("a failure keeps the rule-based text: %v", got["text"])
	}
	e.Call("reader", "POST", "/assistant/ask", map[string]any{"prompt": "thanks", "lang": "en"})
	if len(g.asks) != 2 {
		t.Error("small talk is never sent to Gemini")
	}
}
