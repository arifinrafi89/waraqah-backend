package shelves_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func shelfOf(t *testing.T, list []map[string]any) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, e := range list {
		out[e["book"].(map[string]any)["id"].(string)] = e["shelf"].(string)
	}
	return out
}

func TestDeliveredBooksLandOnWantToReadOnce(t *testing.T) {
	e := testenv.New(t)
	got := shelfOf(t, e.Call("reader", "GET", "/shelves", nil).List(t))
	if got["bk-atomic"] != "wantToRead" || got["bk-sapiens"] != "wantToRead" || got["bk-hobbit"] != "reading" || len(got) != 8 {
		t.Fatalf("shelves: %v", got)
	}
	if got := shelfOf(t, e.Call("reader", "POST", "/shelves/move", map[string]any{"bookId": "bk-atomic"}).List(t)); got["bk-atomic"] != "" {
		t.Error("no shelf takes the Book off")
	}
	if got := shelfOf(t, e.Call("reader", "GET", "/shelves", nil).List(t)); got["bk-atomic"] != "" {
		t.Error("a delivered Book comes back only once")
	}
	if r := e.Call("reader", "POST", "/shelves/move", map[string]any{"bookId": "bk-nope", "shelf": "reading"}); r.Body != nil || r.Refusal != "book_unknown" {
		t.Errorf("unknown book: %s", r.Raw)
	}
	list := e.Call("reader", "POST", "/shelves/move", map[string]any{"bookId": "bk-sapiens", "shelf": "finished"}).List(t)
	if list[0]["shelf"] != "finished" || list[0]["progress"] != 100.0 || list[0]["finishedAt"] == nil {
		t.Errorf("finished moves to the top at 100%%: %v", list[0])
	}
	if got := e.Call("", "GET", "/shelves", nil).List(t); len(got) != 0 {
		t.Error("guest")
	}
}

func TestProgressStatsAndGoal(t *testing.T) {
	e := testenv.New(t)
	before := e.Call("reader", "GET", "/reading/stats", nil).Obj(t)
	if before["goal"] != 12.0 || before["finishedThisYear"] != 4.0 || before["streakDays"] != 3.0 || before["readToday"] != false {
		t.Fatalf("seeded stats: %v", before)
	}
	if r := e.Call("reader", "POST", "/shelves/progress", map[string]any{"bookId": "bk-hobbit", "percent": 50, "pagesRead": 400, "totalPages": 310}); r.Refusal != "progress_invalid" {
		t.Errorf("more pages than the book: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/shelves/progress", map[string]any{"bookId": "bk-riyad", "percent": 10}); r.Refusal != "not_on_shelf" {
		t.Errorf("not on a shelf: %q", r.Refusal)
	}
	list := e.Call("reader", "POST", "/shelves/progress", map[string]any{"bookId": "bk-hobbit", "percent": 0, "pagesRead": 155, "totalPages": 310}).List(t)
	for _, en := range list {
		if en["book"].(map[string]any)["id"] == "bk-hobbit" && en["progress"] != 50.0 {
			t.Errorf("pages set the percentage: %v", en["progress"])
		}
	}
	after := e.Call("reader", "GET", "/reading/stats", nil).Obj(t)
	if after["readToday"] != true || after["streakDays"] != 4.0 {
		t.Errorf("moving forward counts today: %v", after)
	}
	e.Call("reader", "POST", "/shelves/progress", map[string]any{"bookId": "bk-hobbit", "percent": 100})
	if got := e.Call("reader", "GET", "/reading/stats", nil).Obj(t); got["finishedThisYear"] != 5.0 {
		t.Errorf("100%% finishes the book: %v", got)
	}
	if r := e.Call("reader", "POST", "/reading/goal", map[string]any{"goal": 400}); r.Refusal != "goal_invalid" {
		t.Errorf("goal over 365: %q", r.Refusal)
	}
	if got := e.Call("reader", "POST", "/reading/goal", map[string]any{"goal": 24}).Obj(t); got["goal"] != 24.0 {
		t.Errorf("goal: %v", got)
	}
	guest := e.Call("", "GET", "/reading/stats", nil).Obj(t)
	if guest["goal"] != nil || len(guest["perMonth"].([]any)) != 12 {
		t.Errorf("guest stats: %v", guest)
	}
}
