package p2p_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/cloudinary"
	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

// A 1x1 PNG, the smallest picture the server accepts.
const tiny = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func ids(l []map[string]any) []string {
	out := make([]string, len(l))
	for i, m := range l {
		out[i] = m["id"].(string)
	}
	return out
}

func TestPublicListsAndPerViewerFields(t *testing.T) {
	e := testenv.New(t)
	open := e.Call("", "GET", "/p2p/listings", nil).List(t)
	if len(open) != 8 || open[0]["id"] != "p2p-1" {
		t.Fatalf("on sale or reserved, newest first: %v", ids(open))
	}
	for _, l := range open {
		if l["isMine"] != false || l["isMyDeal"] != false {
			t.Errorf("a guest owns nothing: %v", l["id"])
		}
	}
	avail := e.Call("reader", "GET", "/p2p/listings?available=true", nil).List(t)
	if len(avail) != 3 {
		t.Errorf("available (live, not mine): %v", ids(avail))
	}
	if got := e.Call("", "GET", "/p2p/listings?limit=2", nil).List(t); len(got) != 2 {
		t.Errorf("limit: %d", len(got))
	}
	mine := e.Call("reader", "GET", "/p2p/listings/mine", nil).List(t)
	if len(mine) != 6 {
		t.Errorf("own listings in any status: %v", ids(mine))
	}
	for _, l := range mine {
		if l["isMine"] != true {
			t.Errorf("isMine comes from the token: %v", l["id"])
		}
	}
	if deal := e.Call("reader", "GET", "/p2p/listing?id=p2p-4", nil).Obj(t); deal["isMyDeal"] != true {
		t.Errorf("reserved for me: %v", deal)
	}
	if r := e.Call("", "GET", "/p2p/listing?id=p2p-nope", nil); r.Status != 200 || r.Body != nil {
		t.Errorf("unknown: %d %s", r.Status, r.Raw)
	}
	if got := e.Call("", "GET", "/p2p/listings/for-book?bookId=bk-cleancode", nil).List(t); len(got) != 1 || got[0]["id"] != "p2p-1" {
		t.Errorf("for book: %v", ids(got))
	}
	if cat := e.Call("", "GET", "/p2p/listing?id=p2p-1", nil).Obj(t); cat["categoryId"] != "cat-programming" || cat["section"] == nil {
		t.Errorf("category and section: %v %v", cat["categoryId"], cat["section"])
	}
	if got := e.Call("", "GET", "/p2p/listings/mine", nil).List(t); len(got) != 0 {
		t.Error("a guest has no listings")
	}
}

func TestSellerPage(t *testing.T) {
	e := testenv.New(t)
	s := e.Call("", "GET", "/p2p/seller?id=p-nabila", nil).Obj(t)
	if s["booksSold"] != 18.0 || s["ratingCount"] != 4.0 {
		t.Errorf("seller: %v %v", s["booksSold"], s["ratingCount"])
	}
	if avg := s["ratingAverage"].(float64); avg < 4.7 || avg > 4.8 {
		t.Errorf("average: %v", avg)
	}
	if r := e.Call("", "GET", "/p2p/seller?id=p-nobody", nil); r.Body != nil {
		t.Error("unknown seller")
	}
	// the demo reader sold 2 before the records and has 1 sold listing (p2p-hs-2)
	if me := e.Call("reader", "GET", "/p2p/seller?id=u_reader", nil).Obj(t); me["booksSold"] != 3.0 {
		t.Errorf("books sold counts the old and the new: %v", me["booksSold"])
	}
}

func TestSaveDraftSendAndEditRules(t *testing.T) {
	e := testenv.New(t)
	save := func(as string, body map[string]any) testenv.Result {
		return e.Call(as, "POST", "/p2p/listings/save", body)
	}
	draft := save("reader", map[string]any{"title": "SICP", "flags": []string{"notes", "bogus"}}).Obj(t)
	if draft["status"] != "draft" || draft["isMine"] != true || len(draft["flags"].([]any)) != 1 {
		t.Fatalf("draft: %v", draft)
	}
	id := draft["id"].(string)
	if r := save("reader", map[string]any{"id": id, "title": "SICP", "priceBdt": 300, "submit": true}); r.Body != nil || r.Refusal != "listing_invalid" {
		t.Errorf("no photos: %q", r.Refusal)
	}
	sent := save("reader", map[string]any{"id": id, "title": "SICP", "priceBdt": 300, "submit": true,
		"photoData": map[string]any{"front": tiny, "back": tiny}}).Obj(t)
	if sent["id"] != id || sent["status"] != "inReview" || len(sent["photos"].([]any)) != 2 {
		t.Fatalf("sent: %v", sent)
	}
	fake := e.Deps.Images.(*cloudinary.Fake)
	if !fake.Has(e.Deps.Cfg.CloudinaryFolder + "/listing/" + id + "/front") {
		t.Error("the photo reaches the uploader")
	}
	var queue []string
	for _, q := range e.Call("moderator", "GET", "/moderation/listings", nil).List(t) {
		queue = append(queue, q["id"].(string))
	}
	if !contains(queue, id) {
		t.Errorf("queue: %v", queue)
	}
	if r := save("reader", map[string]any{"id": id, "title": "SICP 2"}); r.Refusal != "listing_not_editable" {
		t.Errorf("in review: %q", r.Refusal)
	}
	if r := save("reader", map[string]any{"id": "p2p-1", "title": "Mine now"}); r.Refusal != "listing_unknown" {
		t.Errorf("someone else: %q", r.Refusal)
	}
	if r := save("reader", map[string]any{"title": "Bad photo", "photoData": map[string]any{"front": "AA=="}}); r.Refusal != "listing_photo_invalid" {
		t.Errorf("photo: %q", r.Refusal)
	}
	if r := save("reader", map[string]any{"title": "x", "priceBdt": 60000}); r.Refusal != "listing_invalid" {
		t.Errorf("price: %q", r.Refusal)
	}
	if r := save("", map[string]any{"title": "Guest"}); r.Status != 401 {
		t.Errorf("guest: %d", r.Status)
	}
}

func TestSentBackListingKeepsPhotosAndRemovedSlotsAreDeleted(t *testing.T) {
	e := testenv.New(t)
	save := func(body map[string]any) testenv.Result { return e.Call("reader", "POST", "/p2p/listings/save", body) }
	base := map[string]any{"id": "p2p-changes-1", "title": "Head First Java", "priceBdt": 420}
	kept := save(merge(base, map[string]any{"photos": []string{"front"}})).Obj(t)
	if kept["status"] != "changesRequested" || kept["rejectionReason"] == nil || len(kept["photos"].([]any)) != 1 {
		t.Fatalf("saving a draft keeps the answer of the moderator: %v", kept)
	}
	if r := save(merge(base, map[string]any{"photos": []string{"front"}, "submit": true})); r.Refusal != "listing_invalid" {
		t.Errorf("back cover missing: %q", r.Refusal)
	}
	sent := save(merge(base, map[string]any{"photos": []string{"front"}, "submit": true, "photoData": map[string]any{"back": tiny}})).Obj(t)
	if sent["status"] != "inReview" || sent["rejectionReason"] != nil || len(sent["photos"].([]any)) != 2 {
		t.Fatalf("sending clears the reason: %v", sent)
	}
	// once rejected, a slot the seller drops is deleted from Cloudinary
	e.Call("moderator", "POST", "/moderation/listings/decide", map[string]any{"listingId": "p2p-changes-1", "decision": "reject", "reason": "No"})
	save(merge(base, map[string]any{"photos": []string{"back"}, "photoData": map[string]any{"front": tiny}}))
	got := save(merge(base, map[string]any{"photos": []string{"front"}})).Obj(t)
	if ph := got["photos"].([]any); len(ph) != 1 || ph[0] != "front" {
		t.Fatalf("only the kept slot stays: %v", ph)
	}
	fake := e.Deps.Images.(*cloudinary.Fake)
	if len(fake.Destroyed()) == 0 {
		t.Error("the asset of a removed slot must be destroyed")
	}
}

func TestBlocksHideListings(t *testing.T) {
	e := testenv.New(t)
	e.Call("reader", "POST", "/blocks/add", map[string]any{"readerId": "p-tanvir"})
	for _, l := range e.Call("reader", "GET", "/p2p/listings", nil).List(t) {
		if l["sellerId"] == "p-tanvir" {
			t.Errorf("a blocked seller stays out of the marketplace: %v", l["id"])
		}
	}
	if got := e.Call("reader", "GET", "/p2p/listings/for-book?bookId=bk-cleancode", nil).List(t); len(got) != 0 {
		t.Errorf("and out of the book page: %v", ids(got))
	}
	// the other way round: the blocked seller does not see the reader either
	if got := e.Call("reader", "GET", "/p2p/listings", nil).List(t); len(got) != 6 {
		t.Errorf("listings: %d", len(got))
	}
	e.Call("reader", "POST", "/blocks/remove", map[string]any{"readerId": "p-tanvir"})
	if got := e.Call("reader", "GET", "/p2p/listings", nil).List(t); len(got) != 8 {
		t.Errorf("unblocked: %d", len(got))
	}
}

func merge(a, b map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func contains(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

func TestDeletedAccountsLeaveTheMarketplace(t *testing.T) {
	e := testenv.New(t)
	e.Call("reader", "POST", "/p2p/listings/save", map[string]any{"title": "With photo", "photoData": map[string]any{"front": tiny}})
	if r := e.Call("reader", "POST", "/auth/delete", map[string]any{}); r.Status != 200 {
		t.Fatalf("delete: %d %s", r.Status, r.Raw)
	}
	for _, l := range e.Call("", "GET", "/p2p/listings", nil).List(t) {
		if l["sellerId"] == "u_reader" {
			t.Errorf("a deleted reader still sells: %v", l["id"])
		}
	}
	if len(e.Deps.Images.(*cloudinary.Fake).Destroyed()) != 1 {
		t.Error("their photos are deleted")
	}
}
