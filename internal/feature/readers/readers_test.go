package readers_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func TestReaderPages(t *testing.T) {
	e := testenv.New(t)
	p := e.Call("reader", "GET", "/readers/detail?id=p-nabila", nil).Obj(t)
	if p["isFollowing"] != true || p["isMe"] != false || p["biteCount"] != 2.0 || p["followers"] != 1.0 || p["area"] != "Banani" {
		t.Errorf("Nabila: %v", p)
	}
	if got := e.Call("", "GET", "/readers/detail?id=p-nabila", nil).Obj(t); got["isFollowing"] != false {
		t.Error("a guest follows nobody")
	}
	private := e.Call("reader", "GET", "/readers/detail?id=p-rafi", nil).Obj(t)
	if private["profileVisible"] != false || private["area"] != "" || private["memberSince"] != nil {
		t.Errorf("a private profile shows only the name: %v", private)
	}
	me := e.Call("reader", "GET", "/readers/detail?id=u_reader", nil).Obj(t)
	if me["isMe"] != true || me["followers"] != 2.0 || me["following"] != 3.0 {
		t.Errorf("own page: %v", me)
	}
	if got := e.Call("", "GET", "/readers/detail?id=nobody", nil); got.Body != nil {
		t.Error("unknown reader is null")
	}
}

func TestFollow(t *testing.T) {
	e := testenv.New(t)
	p := e.Call("reader", "POST", "/readers/follow", map[string]any{"id": "p-arif", "follow": true}).Obj(t)
	if p["isFollowing"] != true || p["followers"] != 1.0 {
		t.Fatalf("follow: %v", p)
	}
	if p := e.Call("reader", "POST", "/readers/follow", map[string]any{"id": "p-arif", "follow": false}).Obj(t); p["isFollowing"] != false {
		t.Errorf("unfollow: %v", p)
	}
	for _, c := range []struct{ id, code string }{{"u_reader", "follow_refused"}, {"nobody", "reader_unknown"}} {
		if r := e.Call("reader", "POST", "/readers/follow", map[string]any{"id": c.id, "follow": true}); r.Refusal != c.code {
			t.Errorf("%s: %q", c.id, r.Refusal)
		}
	}
	e.Call("reader", "POST", "/blocks/add", map[string]any{"readerId": "p-rakib"})
	if r := e.Call("reader", "POST", "/readers/follow", map[string]any{"id": "p-rakib", "follow": true}); r.Refusal != "follow_refused" {
		t.Errorf("a blocked reader: %q", r.Refusal)
	}
	if r := e.Call("", "POST", "/readers/follow", map[string]any{"id": "p-arif", "follow": true}); r.Status != 401 {
		t.Errorf("guest follow: %d", r.Status)
	}
}
