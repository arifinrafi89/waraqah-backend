package reviews_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func rating(e *testenv.Env, bookID string) any {
	return e.Call("", "GET", "/books/detail?id="+bookID, nil).Obj(e.T)["rating"]
}

func TestReviewsOfABook(t *testing.T) {
	e := testenv.New(t)
	got := e.Call("", "GET", "/reviews?bookId=bk-cleancode", nil).Obj(t)
	if got["average"] != 4.5 || got["count"] != 2.0 || got["mine"] != nil {
		t.Fatalf("summary: %v", got)
	}
	list := got["reviews"].([]any)
	if first := list[0].(map[string]any); first["id"] != "rv-6" || first["verified"] != false {
		t.Errorf("newest first: %v", first)
	}
	if got := e.Call("", "GET", "/reviews?bookId=bk-nope", nil).Obj(t); got["count"] != 0.0 || got["average"] != 0.0 {
		t.Errorf("no reviews: %v", got)
	}
}

func TestSaveEditAndDelete(t *testing.T) {
	e := testenv.New(t)
	for _, c := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"bookId": "bk-cleancode", "stars": 0, "text": ""}, "review_invalid"},
		{map[string]any{"bookId": "bk-nope", "stars": 4, "text": ""}, "book_unknown"},
	} {
		if r := e.Call("reader", "POST", "/reviews/save", c.body); r.Body != nil || r.Refusal != c.code {
			t.Errorf("%v: %q", c.body, r.Refusal)
		}
	}
	got := e.Call("reader", "POST", "/reviews/save", map[string]any{"bookId": "bk-cleancode", "stars": 5, "text": " Great "}).Obj(t)
	mine := got["mine"].(map[string]any)
	if got["count"] != 3.0 || got["average"] != 4.7 || mine["text"] != "Great" || mine["isMine"] != true {
		t.Fatalf("save: %v", got)
	}
	if rating(e, "bk-cleancode") != 4.7 {
		t.Errorf("the book's rating follows: %v", rating(e, "bk-cleancode"))
	}
	edited := e.Call("reader", "POST", "/reviews/save", map[string]any{"bookId": "bk-cleancode", "stars": 3, "text": "Hm"}).Obj(t)
	if edited["count"] != 3.0 || edited["mine"].(map[string]any)["editedAt"] == nil {
		t.Errorf("saving again edits: %v", edited)
	}
	if got := e.Call("reader", "POST", "/reviews/delete", map[string]any{"bookId": "bk-cleancode"}).Obj(t); got["count"] != 2.0 || got["mine"] != nil {
		t.Errorf("delete: %v", got)
	}
	if rating(e, "bk-cleancode") != 4.5 {
		t.Errorf("the rating goes back: %v", rating(e, "bk-cleancode"))
	}
	if r := e.Call("reader", "POST", "/reviews/delete", map[string]any{"bookId": "bk-cleancode"}); r.Refusal != "review_unknown" {
		t.Errorf("nothing to delete: %q", r.Refusal)
	}
}

func TestVerifiedPurchaseComesFromDeliveredOrders(t *testing.T) {
	e := testenv.New(t)
	// The demo reader's delivered order WQ-100201 has Sapiens; the moderator bought nothing.
	r := e.Call("reader", "POST", "/reviews/save", map[string]any{"bookId": "bk-sapiens", "stars": 5, "text": ""}).Obj(t)
	if r["mine"].(map[string]any)["verified"] != true {
		t.Errorf("delivered: %v", r["mine"])
	}
	m := e.Call("moderator", "POST", "/reviews/save", map[string]any{"bookId": "bk-sapiens", "stars": 2, "text": ""}).Obj(t)
	if m["mine"].(map[string]any)["verified"] != false {
		t.Errorf("not bought: %v", m["mine"])
	}
}

func TestModeratorRemovesAReportedReview(t *testing.T) {
	e := testenv.New(t)
	e.Call("reader", "POST", "/reports", map[string]any{"kind": "review", "targetId": "rv-6", "reason": "offensive"})
	var reportID string
	for _, r := range e.Call("moderator", "GET", "/moderation/reports", nil).List(t) {
		if r["targetId"] == "rv-6" {
			reportID = r["id"].(string)
		}
	}
	if r := e.Call("moderator", "POST", "/moderation/reports/act", map[string]any{"reportId": reportID, "action": "remove"}); r.Status != 200 || r.Refusal != "" {
		t.Fatalf("act: %q %s", r.Refusal, r.Raw)
	}
	if got := e.Call("", "GET", "/reviews?bookId=bk-cleancode", nil).Obj(t); got["count"] != 1.0 {
		t.Errorf("removed: %v", got)
	}
	if rating(e, "bk-cleancode") != 5.0 {
		t.Errorf("rating after removal: %v", rating(e, "bk-cleancode"))
	}
}
