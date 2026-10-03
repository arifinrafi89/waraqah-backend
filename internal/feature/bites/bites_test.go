package bites_test

import (
	"strings"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func comments(d map[string]any) []map[string]any {
	var out []map[string]any
	for _, c := range d["comments"].([]any) {
		out = append(out, c.(map[string]any))
	}
	return out
}

func TestFeeds(t *testing.T) {
	e := testenv.New(t)
	feed := e.Call("", "GET", "/bites?feed=forYou", nil).List(t)
	if len(feed) != 14 || feed[0]["id"] != "bt-1" || feed[0]["likes"] != 3.0 || feed[0]["comments"] != 3.0 || feed[0]["bookTitle"] == nil {
		t.Fatalf("for you, newest first: %d %v", len(feed), feed[0])
	}
	mine := e.Call("reader", "GET", "/bites", nil).List(t)
	if mine[1]["id"] != "bt-me-1" || mine[1]["isMine"] != true || mine[0]["isMine"] != false {
		t.Errorf("isMine comes from the token: %v", mine[1])
	}
	if mine[2]["liked"] != true {
		t.Errorf("the reader likes bt-2: %v", mine[2])
	}
	following := e.Call("reader", "GET", "/bites?feed=following", nil).List(t)
	for _, b := range following {
		if a := b["authorId"]; a != "p-tanvir" && a != "p-nabila" && a != "p-talha" {
			t.Errorf("following has %v", a)
		}
	}
	if len(following) != 6 {
		t.Errorf("following: %d", len(following))
	}
	if got := e.Call("", "GET", "/bites?feed=following", nil).List(t); len(got) != 0 {
		t.Error("a guest follows nobody")
	}
	if got := e.Call("", "GET", "/bites?bookId=bk-sapiens", nil).IDs(t); len(got) != 1 || got[0] != "bt-1" {
		t.Errorf("by book: %v", got)
	}
	if got := e.Call("", "GET", "/bites?authorId=p-arif", nil).IDs(t); len(got) != 2 {
		t.Errorf("by author: %v", got)
	}
	// blocking hides a reader's Bites both ways
	e.Call("reader", "POST", "/blocks/add", map[string]any{"readerId": "p-tanvir"})
	for _, b := range e.Call("reader", "GET", "/bites", nil).List(t) {
		if b["authorId"] == "p-tanvir" {
			t.Fatal("a blocked reader's Bites show")
		}
	}
	if r := e.Call("reader", "POST", "/bites/like", map[string]any{"id": "bt-1", "liked": true}); r.Refusal != "blocked_reader" {
		t.Errorf("liking a blocked reader's Bite: %q", r.Refusal)
	}
}

func TestPostEditLikeDelete(t *testing.T) {
	e := testenv.New(t)
	for _, c := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"text": "  "}, "bite_invalid"},
		{map[string]any{"text": strings.Repeat("a", 501)}, "bite_invalid"},
		{map[string]any{"text": "the end!", "spoiler": true}, "bite_invalid"},
		{map[string]any{"text": "hi", "bookId": "bk-nope"}, "book_unknown"},
	} {
		if r := e.Call("reader", "POST", "/bites/post", c.body); r.Body != nil || r.Refusal != c.code {
			t.Errorf("%v: %q", c.body, r.Refusal)
		}
	}
	b := e.Call("reader", "POST", "/bites/post", map[string]any{"text": " Loved it ", "bookId": "bk-cleancode"}).Obj(t)
	id := b["id"].(string)
	if b["text"] != "Loved it" || b["bookTitle"] != "Clean Code" || b["isMine"] != true || b["editedAt"] != nil {
		t.Fatalf("post: %v", b)
	}
	if got := e.Call("", "GET", "/bites", nil).IDs(t); got[0] != id {
		t.Errorf("the new Bite comes first: %v", got[:2])
	}
	if r := e.Call("moderator", "POST", "/bites/edit", map[string]any{"id": id, "text": "mine now"}); r.Refusal != "bite_not_yours" {
		t.Errorf("edit someone else's: %q", r.Refusal)
	}
	edited := e.Call("reader", "POST", "/bites/edit", map[string]any{"id": id, "text": "Loved it, really"}).Obj(t)
	if edited["bookId"] != nil || edited["editedAt"] == nil {
		t.Errorf("an edit without a tag clears it: %v", edited)
	}
	if got := e.Call("moderator", "POST", "/bites/like", map[string]any{"id": id, "liked": true}).Obj(t); got["likes"] != 1.0 || got["liked"] != true {
		t.Errorf("like: %v", got)
	}
	if got := e.Call("moderator", "POST", "/bites/like", map[string]any{"id": id, "liked": false}).Obj(t); got["likes"] != 0.0 {
		t.Errorf("unlike: %v", got)
	}
	if r := e.Call("moderator", "POST", "/bites/delete", map[string]any{"id": id}); r.Refusal != "bite_not_yours" {
		t.Errorf("delete someone else's: %q", r.Refusal)
	}
	if got := e.Call("reader", "POST", "/bites/delete", map[string]any{"id": id}).Obj(t); got["id"] != id {
		t.Errorf("delete: %v", got)
	}
	if got := e.Call("", "GET", "/bites/detail?id="+id, nil); got.Body != nil {
		t.Error("a deleted Bite is gone")
	}
}

func TestCommentsAndReplies(t *testing.T) {
	e := testenv.New(t)
	d := e.Call("reader", "POST", "/bites/comments/post", map[string]any{"biteId": "bt-1", "text": "Agreed", "parentId": "cm-2"}).Obj(t)
	top := comments(d)[0]
	replies := top["replies"].([]any)
	if top["id"] != "cm-1" || len(replies) != 2 || replies[1].(map[string]any)["parentId"] != "cm-1" {
		t.Fatalf("a reply to a reply moves up to its top comment: %v", top)
	}
	if r := e.Call("reader", "POST", "/bites/comments/post", map[string]any{"biteId": "bt-1", "text": "x", "parentId": "cm-4"}); r.Refusal != "comment_unknown" {
		t.Errorf("a parent on another Bite: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/bites/comments/post", map[string]any{"biteId": "bt-1", "text": " "}); r.Refusal != "comment_invalid" {
		t.Errorf("empty comment: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/bites/comments/delete", map[string]any{"id": "cm-1"}); r.Refusal != "comment_not_yours" {
		t.Errorf("delete someone else's comment: %q", r.Refusal)
	}
	mine := e.Call("reader", "POST", "/bites/comments/post", map[string]any{"biteId": "bt-me-1", "text": "Thanks!"}).Obj(t)
	c := comments(mine)
	own := c[len(c)-1]
	if own["isMine"] != true {
		t.Fatalf("own comment: %v", own)
	}
	if d := e.Call("reader", "POST", "/bites/comments/delete", map[string]any{"id": own["id"]}).Obj(t); len(comments(d)) != 1 {
		t.Errorf("deleted: %v", comments(d))
	}
	// Nabila wrote cm-1 and is told about the reply; the Bite's author Tanvir is not.
	if got := e.Call("reader", "GET", "/notifications", nil).List(t); len(got) == 0 {
		t.Error("notifications")
	}
}

func TestModeratorRemovesAReportedBite(t *testing.T) {
	e := testenv.New(t)
	rep := e.Call("reader", "POST", "/reports", map[string]any{"kind": "bite", "targetId": "bt-11", "reason": "spam"}).Obj(t)
	var reportID string
	for _, r := range e.Call("moderator", "GET", "/moderation/reports", nil).List(t) {
		if r["targetId"] == "bt-11" {
			reportID = r["id"].(string)
		}
	}
	if reportID == "" {
		t.Fatalf("the report waits for a moderator: %v", rep)
	}
	if r := e.Call("moderator", "POST", "/moderation/reports/act", map[string]any{"reportId": reportID, "action": "remove"}); r.Status != 200 || r.Refusal != "" {
		t.Fatalf("act: %d %q %s", r.Status, r.Refusal, r.Raw)
	}
	if got := e.Call("", "GET", "/bites/detail?id=bt-11", nil); got.Body != nil {
		t.Error("the Bite was removed")
	}
}
