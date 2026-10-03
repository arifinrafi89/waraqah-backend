package wishlist_test

import (
	"reflect"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func save(e *testenv.Env, as, book string) testenv.Result {
	return e.Call(as, "POST", "/wishlist/save", map[string]any{"bookId": book})
}

func TestSaveOrderAndRemove(t *testing.T) {
	e := testenv.New(t)
	if got := e.Call("", "GET", "/wishlist", nil).List(t); len(got) != 0 {
		t.Errorf("guest: %v", got)
	}
	save(e, "reader", "bk-atomic")
	save(e, "reader", "bk-sapiens")
	save(e, "reader", "bk-hobbit")
	if got := save(e, "reader", "bk-atomic").IDs(t); !reflect.DeepEqual(got, []string{"bk-atomic", "bk-hobbit", "bk-sapiens"}) {
		t.Errorf("saving twice keeps one copy, moved to the top: %v", got)
	}
	if got := save(e, "reader", "bk-nope").IDs(t); len(got) != 3 {
		t.Errorf("an unknown book changes nothing: %v", got)
	}
	if got := e.Call("reader", "POST", "/wishlist/remove", map[string]any{"bookId": "bk-hobbit"}).IDs(t); !reflect.DeepEqual(got, []string{"bk-atomic", "bk-sapiens"}) {
		t.Errorf("after remove: %v", got)
	}
	if got := e.Call("moderator", "GET", "/wishlist", nil).List(t); len(got) != 0 {
		t.Error("a wishlist must not leak to another reader")
	}
}

func TestShareLinks(t *testing.T) {
	e := testenv.New(t)
	save(e, "reader", "bk-atomic")
	if r := e.Call("reader", "POST", "/wishlist/share", map[string]any{"ownerName": "  "}); r.Refusal != "wishlist_name_missing" || r.Body != nil {
		t.Errorf("blank name: %q", r.Refusal)
	}
	first := e.Call("reader", "POST", "/wishlist/share", map[string]any{"ownerName": "Rafiq"}).Obj(t)
	again := e.Call("reader", "POST", "/wishlist/share", map[string]any{"ownerName": "Rafiq A."}).Obj(t)
	if first["id"] != again["id"] || again["ownerName"] != "Rafiq A." {
		t.Errorf("the link is reused and the name updated: %v %v", first, again)
	}
	// a stranger sees the list as it is now, without signing in
	save(e, "reader", "bk-sapiens")
	seen := e.Call("", "GET", "/wishlist/shared?id="+first["id"].(string), nil).Obj(t)
	if len(seen["books"].([]any)) != 2 || seen["ownerName"] != "Rafiq A." {
		t.Errorf("shared: %v", seen)
	}
	other := e.Call("moderator", "POST", "/wishlist/share", map[string]any{"ownerName": "Mod"}).Obj(t)
	if other["id"] == first["id"] || len(other["books"].([]any)) != 0 {
		t.Errorf("each reader has their own link: %v", other)
	}
	if r := e.Call("", "GET", "/wishlist/shared?id=wl-nobody", nil); r.Body != nil {
		t.Error("unknown link")
	}
	if demo := e.Call("", "GET", "/wishlist/shared?id=wl-nabila", nil).Obj(t); demo["ownerName"] != "Nabila" || len(demo["books"].([]any)) != 3 {
		t.Errorf("demo list: %v", demo)
	}
}
