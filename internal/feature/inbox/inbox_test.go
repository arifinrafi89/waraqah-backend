package inbox_test

import (
	"testing"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func msgs(th map[string]any) []map[string]any {
	var out []map[string]any
	for _, m := range th["messages"].([]any) {
		out = append(out, m.(map[string]any))
	}
	return out
}

func TestSeededThreadsFromTheTokenSide(t *testing.T) {
	e := testenv.New(t)
	list := e.Call("reader", "GET", "/inbox", nil).List(t)
	if len(list) != 4 || list[0]["id"] != "th-sadia" {
		t.Fatalf("newest activity first: %v", ids(list))
	}
	byID := map[string]map[string]any{}
	for _, th := range list {
		byID[th["id"].(string)] = th
		if len(th["messages"].([]any)) != 1 {
			t.Errorf("the list carries only the last message: %v", th["id"])
		}
	}
	if byID["th-sadia"]["role"] != "seller" || byID["th-sadia"]["unread"] != 2.0 || byID["th-nabila"]["role"] != "buyer" || byID["th-nabila"]["unread"] != 1.0 {
		t.Errorf("role and unread come from the token: %v %v", byID["th-sadia"], byID["th-nabila"])
	}
	if byID["th-nabila"]["dealHere"] != true || byID["th-talha"]["dealHere"] != true || byID["th-sadia"]["dealHere"] != false {
		t.Error("dealHere")
	}
	// the other side of the same thread sees the opposite
	if got := e.Call("moderator", "GET", "/inbox/thread?id=th-sadia", nil); got.Body != nil {
		t.Error("someone outside the thread sees nothing")
	}
	if got := e.Call("reader", "GET", "/inbox?listingId=p2p-7", nil).List(t); len(got) != 2 {
		t.Errorf("by listing: %v", ids(got))
	}
	if got := e.Call("reader", "GET", "/inbox/thread?id=th-nabila", nil).Obj(t); len(msgs(got)) != 4 || msgs(got)[0]["offer"] == nil {
		t.Errorf("whole thread: %v", got)
	}
	if got := e.Call("", "GET", "/inbox", nil).List(t); len(got) != 0 {
		t.Error("guest")
	}
}

func TestReadingChattingAndOffers(t *testing.T) {
	e := testenv.New(t)
	if th := e.Call("reader", "POST", "/inbox/read", map[string]any{"threadId": "th-sadia"}).Obj(t); th["unread"] != 0.0 {
		t.Errorf("read: %v", th["unread"])
	}
	for _, c := range []struct{ id, text, code string }{
		{"th-sadia", " ", "message_invalid"}, {"th-nope", "hi", "thread_unknown"},
	} {
		if r := e.Call("reader", "POST", "/inbox/send", map[string]any{"threadId": c.id, "text": c.text}); r.Body != nil || r.Refusal != c.code {
			t.Errorf("%q: %q", c.text, r.Refusal)
		}
	}
	// the demo reader Sadia answers once, at once in tests
	got := e.Call("reader", "POST", "/inbox/send", map[string]any{"threadId": "th-sadia", "text": "Yes, it is"}).Obj(t)
	if m := msgs(got); m[len(m)-1]["text"] != "Thanks for getting back to me!" || m[len(m)-1]["from"] != "them" {
		t.Errorf("bot answer: %v", m[len(m)-1])
	}
	if again := e.Call("reader", "POST", "/inbox/send", map[string]any{"threadId": "th-sadia", "text": "More"}).Obj(t); len(msgs(again)) != len(msgs(got))+1 {
		t.Error("the bot answers only once per thread")
	}
	// a buyer's offer
	offer := func(listing string, amount int, handover string) testenv.Result {
		return e.Call("reader", "POST", "/inbox/offer", map[string]any{"listingId": listing, "amountBdt": amount, "handover": handover})
	}
	for name, c := range map[string]struct {
		r    testenv.Result
		code string
	}{
		"fixed price":  {offer("p2p-6", 200, "meetup"), "offer_invalid"},
		"above asking": {offer("p2p-1", 999, "meetup"), "offer_invalid"},
		"bad handover": {offer("p2p-1", 300, "drone"), "offer_invalid"},
		"own listing":  {offer("p2p-7", 300, "meetup"), "listing_own"},
		"not live":     {offer("p2p-4", 300, "meetup"), "listing_unavailable"},
		"unknown":      {offer("p2p-nope", 300, "meetup"), "listing_unknown"},
	} {
		if c.r.Body != nil || c.r.Refusal != c.code {
			t.Errorf("%s: %q", name, c.r.Refusal)
		}
	}
	th := offer("p2p-1", 300, "meetup").Obj(t)
	last := msgs(th)
	if th["role"] != "buyer" || last[len(last)-2]["offer"] == nil && last[len(last)-1]["offer"] == nil {
		t.Errorf("offer thread: %v", th)
	}
	if r := offer("p2p-1", 310, "meetup"); r.Refusal != "offer_pending" {
		t.Errorf("one offer at a time: %q", r.Refusal)
	}
}

func TestSellerDecidesReservesSellsAndBothRate(t *testing.T) {
	e := testenv.New(t)
	call := func(path string, body map[string]any) testenv.Result { return e.Call("reader", "POST", path, body) }
	if r := call("/inbox/offer/decide", map[string]any{"threadId": "th-sadia", "offerId": "o-nope", "accept": true}); r.Refusal != "offer_unknown" {
		t.Errorf("unknown offer: %q", r.Refusal)
	}
	if r := call("/inbox/offer/decide", map[string]any{"threadId": "th-nabila", "offerId": "o-nabila", "accept": true}); r.Body != nil {
		t.Error("only the seller decides")
	}
	th := call("/inbox/offer/decide", map[string]any{"threadId": "th-sadia", "offerId": "o-sadia", "accept": true}).Obj(t)
	if th["dealHere"] != true || th["listing"].(map[string]any)["status"] != "reserved" {
		t.Fatalf("accepting reserves the book for Sadia: %v", th)
	}
	// Rafi hears the book is reserved elsewhere; his offer can not be accepted now
	rafi := e.Call("reader", "GET", "/inbox/thread?id=th-rafi", nil).Obj(t)
	if m := msgs(rafi); m[len(m)-1]["from"] != "system" || m[len(m)-1]["event"] != "reservedElsewhere" {
		t.Errorf("tell the others: %v", m[len(m)-1])
	}
	if r := call("/inbox/offer/decide", map[string]any{"threadId": "th-rafi", "offerId": "o-rafi", "accept": true}); r.Refusal != "listing_unavailable" {
		t.Errorf("second accept: %q", r.Refusal)
	}
	// the deal with Sadia falls through: the book is on sale again for everyone
	if r := call("/inbox/listing/release", map[string]any{"threadId": "th-sadia"}); r.Obj(t)["listing"].(map[string]any)["status"] != "live" {
		t.Fatalf("release: %s", r.Raw)
	}
	if r := call("/inbox/listing/sold", map[string]any{"threadId": "th-sadia"}); r.Refusal != "deal_unknown" {
		t.Errorf("not reserved any more: %q", r.Refusal)
	}
	if rafi := e.Call("reader", "GET", "/inbox/thread?id=th-rafi", nil).Obj(t); msgs(rafi)[len(msgs(rafi))-1]["event"] != "madeAvailable" {
		t.Error("everyone talking about it is told")
	}
	// Rafi's offer is accepted instead, then sold
	call("/inbox/offer/decide", map[string]any{"threadId": "th-rafi", "offerId": "o-rafi", "accept": true})
	if r := call("/inbox/rate", map[string]any{"threadId": "th-rafi", "stars": 5}); r.Refusal != "rating_not_allowed" {
		t.Errorf("rating before the sale: %q", r.Refusal)
	}
	sold := call("/inbox/listing/sold", map[string]any{"threadId": "th-rafi"}).Obj(t)
	if sold["listing"].(map[string]any)["status"] != "sold" || sold["theirRating"] != 5.0 {
		t.Errorf("sold, and the demo buyer rated the seller: %v %v", sold["listing"], sold["theirRating"])
	}
	if sadia := e.Call("reader", "GET", "/inbox/thread?id=th-sadia", nil).Obj(t); msgs(sadia)[len(msgs(sadia))-1]["event"] != "soldElsewhere" {
		t.Error("the others are told it was sold")
	}
	for _, c := range []struct {
		stars int
		code  string
	}{{0, "rating_invalid"}, {6, "rating_invalid"}} {
		if r := call("/inbox/rate", map[string]any{"threadId": "th-rafi", "stars": c.stars}); r.Refusal != c.code {
			t.Errorf("%d stars: %q", c.stars, r.Refusal)
		}
	}
	rated := call("/inbox/rate", map[string]any{"threadId": "th-rafi", "stars": 4, "comment": "Good"}).Obj(t)
	if rated["myRating"] != 4.0 {
		t.Errorf("rated: %v", rated["myRating"])
	}
	if r := call("/inbox/rate", map[string]any{"threadId": "th-rafi", "stars": 4}); r.Refusal != "rating_not_allowed" {
		t.Errorf("once each: %q", r.Refusal)
	}
	// the buyer of a sold book can rate too (Talha sold p2p-5 to the reader)
	if r := call("/inbox/rate", map[string]any{"threadId": "th-talha", "stars": 5}); r.Status != 200 || r.Obj(t)["myRating"] != 5.0 {
		t.Errorf("buyer rates: %s", r.Raw)
	}
	// and the ratings reach the seller page
	if me := e.Call("", "GET", "/p2p/seller?id=p-rafi", nil).Obj(t); me["ratingCount"] != 1.0 {
		t.Errorf("Rafi was rated once: %v", me["ratingCount"])
	}
}

func TestBlockingStopsMessagesBothWays(t *testing.T) {
	e := testenv.New(t)
	e.Call("reader", "POST", "/blocks/add", map[string]any{"readerId": "p-sadia"})
	if r := e.Call("reader", "POST", "/inbox/send", map[string]any{"threadId": "th-sadia", "text": "hi"}); r.Refusal != "blocked_reader" {
		t.Errorf("send: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/inbox/offer/decide", map[string]any{"threadId": "th-sadia", "offerId": "o-sadia", "accept": true}); r.Refusal != "blocked_reader" {
		t.Errorf("accept: %q", r.Refusal)
	}
	if r := e.Call("reader", "POST", "/inbox/offer/decide", map[string]any{"threadId": "th-sadia", "offerId": "o-sadia", "accept": false}); r.Status != 200 || r.Body == nil {
		t.Errorf("a blocked buyer's offer can still be declined: %s", r.Raw)
	}
	e.Call("reader", "POST", "/blocks/remove", map[string]any{"readerId": "p-sadia"})
	// the other direction: Tanvir blocks the reader (listing p2p-1 is Tanvir's)
	if _, err := e.Deps.Report.Block(t.Context(), "p-tanvir", "u_reader"); err != nil {
		t.Fatal(err)
	}
	if r := e.Call("reader", "POST", "/inbox/open", map[string]any{"listingId": "p2p-1"}); r.Refusal != "blocked_reader" {
		t.Errorf("open: %q", r.Refusal)
	}
}

func TestOpenReusesTheThreadAndRefusesClosedListings(t *testing.T) {
	e := testenv.New(t)
	open := func(id string) testenv.Result {
		return e.Call("reader", "POST", "/inbox/open", map[string]any{"listingId": id})
	}
	first := open("p2p-1").Obj(t)
	if first["role"] != "buyer" || len(msgs(first)) != 0 {
		t.Fatalf("a new thread: %v", first)
	}
	if open("p2p-1").Obj(t)["id"] != first["id"] {
		t.Error("the buyer's thread is reused")
	}
	if got := e.Call("reader", "GET", "/inbox", nil).List(t); len(got) != 4 {
		t.Error("a thread without messages is not listed")
	}
	if r := open("p2p-2"); r.Refusal != "listing_closed" {
		t.Errorf("sold: %q", r.Refusal)
	}
	if r := open("p2p-7"); r.Refusal != "listing_own" {
		t.Errorf("own: %q", r.Refusal)
	}
}

func TestLiveEventsReachBothReaders(t *testing.T) {
	e := testenv.New(t)
	events, cancel := e.Deps.SSE.Subscribe("inbox:p-sadia")
	defer cancel()
	e.Call("reader", "POST", "/inbox/read", map[string]any{"threadId": "th-sadia"})
	select {
	case ev := <-events:
		if string(ev) == "" {
			t.Error("empty event")
		}
	case <-time.After(time.Second):
		t.Fatal("no live event for the other reader")
	}
}

func TestRemovedMessagesLeaveTheThread(t *testing.T) {
	e := testenv.New(t)
	e.Call("moderator", "POST", "/reports", map[string]any{"kind": "message", "targetId": "m-s11", "reason": "spam"})
	rep := e.Call("moderator", "GET", "/moderation/reports", nil).List(t)
	var id string
	for _, r := range rep {
		if r["targetId"] == "m-s11" {
			id = r["id"].(string)
			if r["ownerName"] != "Talha" {
				t.Errorf("owner: %v", r["ownerName"])
			}
		}
	}
	if id == "" {
		t.Fatal("the message report must appear")
	}
	e.Call("moderator", "POST", "/moderation/reports/act", map[string]any{"reportId": id, "action": "remove"})
	th := e.Call("reader", "GET", "/inbox/thread?id=th-talha", nil).Obj(t)
	for _, m := range msgs(th) {
		if m["id"] == "m-s11" {
			t.Error("a removed message must be gone")
		}
	}
}

func ids(l []map[string]any) []string {
	out := make([]string, len(l))
	for i, m := range l {
		out[i] = m["id"].(string)
	}
	return out
}
