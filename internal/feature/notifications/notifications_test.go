package notifications_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/notifications"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/dbtest"
)

type env struct {
	t    *testing.T
	h    http.Handler
	deps *app.Deps
	svc  *notifications.Service
	tok  map[string]string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	d := dbtest.New(t)
	cfg, err := config.LoadFrom(os.LookupEnv)
	if err != nil {
		t.Fatal(err)
	}
	deps, err := app.NewDeps(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), d)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, h: app.Routes(deps), deps: deps, svc: deps.Notifications, tok: map[string]string{}}
	for _, id := range []string{"u_ann", "u_bob"} {
		if _, err := d.Conn.Exec(context.Background(), "INSERT INTO users (id, email, name) VALUES ($1, $2, $3)", id, id+"@example.test", id); err != nil {
			t.Fatal(err)
		}
		e.tok[id], _, _ = deps.JWT.Sign(id, auth.RoleReader)
	}
	return e
}

func (e *env) get(as, path string) []map[string]any {
	e.t.Helper()
	return e.do(as, "GET", path, "")
}

func (e *env) do(as, method, path, body string) []map[string]any {
	e.t.Helper()
	req := httptest.NewRequest(method, "/v1"+path, nil)
	if body != "" {
		req = httptest.NewRequest(method, "/v1"+path, stringsReader(body))
	}
	req.Header.Set("Authorization", "Bearer "+e.tok[as])
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	var out []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return out
}

func send(e *env, user string, kind notifications.Kind) {
	e.t.Helper()
	if err := e.svc.Send(context.Background(), user, kind, map[string]string{"k": "v"}, notifications.To(notifications.TargetOrder, "WQ-1")); err != nil {
		e.t.Fatal(err)
	}
}

func TestListIsNewestFirstAndPerReader(t *testing.T) {
	e := newEnv(t)
	clk := &stepClock{now: time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)}
	e.svc.Clock = clk
	send(e, "u_ann", notifications.KindOrderStatus)
	clk.now = clk.now.Add(time.Hour)
	send(e, "u_ann", notifications.KindAlertTriggered)
	send(e, "u_bob", notifications.KindBanned)
	list := e.get("u_ann", "/notifications")
	if len(list) != 2 || list[0]["kind"] != "alertTriggered" || list[1]["kind"] != "orderStatus" {
		t.Fatalf("ann: %v", list)
	}
	if list[0]["read"] != false || list[0]["target"].(map[string]any)["id"] != "WQ-1" || list[0]["params"].(map[string]any)["k"] != "v" {
		t.Errorf("shape: %v", list[0])
	}
	if got := e.get("u_bob", "/notifications"); len(got) != 1 || got[0]["kind"] != "banned" {
		t.Errorf("bob: %v", got)
	}
}

func TestMutedGroupsAreDroppedButModerationIsNot(t *testing.T) {
	e := newEnv(t)
	e.do("u_ann", "POST", "/profile/prefs/save", `{"muted":["orders","usedBooks","alerts","community"],"profileVisible":true,"activityVisible":true}`)
	for _, k := range []notifications.Kind{notifications.KindOrderStatus, notifications.KindAlertTriggered, notifications.KindNewFollower, notifications.KindSaleSent, notifications.KindListingDecided} {
		send(e, "u_ann", k)
	}
	if got := e.get("u_ann", "/notifications"); len(got) != 0 {
		t.Fatalf("muted notifications were stored: %v", got)
	}
	send(e, "u_ann", notifications.KindModerationWarning)
	send(e, "u_ann", notifications.KindBanned)
	if got := e.get("u_ann", "/notifications"); len(got) != 2 {
		t.Errorf("moderation cannot be muted, got %d", len(got))
	}
	// Muting is per reader.
	send(e, "u_bob", notifications.KindOrderStatus)
	if got := e.get("u_bob", "/notifications"); len(got) != 1 {
		t.Error("bob did not mute")
	}
}

func TestReadMarksAndPublishesUnread(t *testing.T) {
	e := newEnv(t)
	send(e, "u_ann", notifications.KindOrderStatus)
	send(e, "u_ann", notifications.KindAlertTriggered)
	events, cancel := e.deps.SSE.Subscribe(notifications.Topic("u_ann"))
	defer cancel()

	first := e.get("u_ann", "/notifications")[0]["id"].(string)
	list := e.do("u_ann", "POST", "/notifications/read", `{"id":"`+first+`"}`)
	if len(list) != 2 || list[0]["read"] != true || list[1]["read"] != false {
		t.Fatalf("after read: %v", list)
	}
	if got := string(<-events); got != `{"unread":1}` {
		t.Errorf("event %s", got)
	}
	e.do("u_ann", "POST", "/notifications/read-all", "")
	if got := string(<-events); got != `{"unread":0}` {
		t.Errorf("event %s", got)
	}
	// someone else's id is refused with null
	req := httptest.NewRequest("POST", "/v1/notifications/read", stringsReader(`{"id":"`+first+`"}`))
	req.Header.Set("Authorization", "Bearer "+e.tok["u_bob"])
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Waraqah-Error") != "notification_unknown" || rec.Body.String() != "null" {
		t.Errorf("foreign id: %q %q", rec.Header().Get("X-Waraqah-Error"), rec.Body)
	}
}

func TestLiveStreamNeedsASignedInReader(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest("GET", "/v1/notifications/live", nil)
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("guest live: %d", rec.Code)
	}
}

// Ported from test/notification_sale_senders_test.dart and the NotificationSends helpers: same kinds,
// same params.
type recorder struct{ sent []sentOne }

type sentOne struct {
	user   string
	kind   notifications.Kind
	params map[string]string
	target *notifications.Target
}

func (r *recorder) Send(_ context.Context, user string, kind notifications.Kind, params map[string]string, target *notifications.Target) error {
	r.sent = append(r.sent, sentOne{user, kind, params, target})
	return nil
}

func TestSendHelpersFillTheSameParamsAsTheDartSenders(t *testing.T) {
	ctx := context.Background()
	r := &recorder{}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(notifications.OrderChanged(ctx, r, "u1", "WQ-1", "shipped"))
	must(notifications.ReturnDecided(ctx, r, "u1", "WQ-1", true))
	must(notifications.ListingDecided(ctx, r, "u2", "p2p-1", "Atomic Habits", "reject", "Blurry"))
	must(notifications.ListingDecided(ctx, r, "u2", "p2p-1", "Atomic Habits", "approve", ""))
	must(notifications.Warned(ctx, r, "u2", 2, 3))
	must(notifications.Warned(ctx, r, "u2", 3, 3))
	must(notifications.AlertTriggered(ctx, r, "u1", "bk-atomic", "Atomic Habits", false))
	must(notifications.AlertTriggered(ctx, r, "u1", "bk-atomic", "Atomic Habits", true))
	must(notifications.WantedBook(ctx, r, []string{"u3", "u4"}, "Sapiens"))
	must(notifications.SaleSentNotice(ctx, r, "u5", "HS-103", "Clean Code"))
	must(notifications.SaleCompletedNotice(ctx, r, "u6", "HS-101", "Sapiens", 361))
	must(notifications.SaleSettledNotice(ctx, r, "buyer", "seller", "HS-104", "Zero to One", true, 250))
	must(notifications.SellBackPaidNotice(ctx, r, "u1", "SB-203", "Sapiens", 180))
	must(notifications.SellBackReturnedNotice(ctx, r, "u7", "SB-202", "Zero to One"))
	must(notifications.Followed(ctx, r, "u1", "p-arif", "Arif"))
	must(notifications.BiteCommented(ctx, r, "u1", "bt-1", "Nabila", "Two-minute rule"))
	must(notifications.CommentReplied(ctx, r, "u1", "bt-1", "Tanvir"))

	type want struct {
		user   string
		kind   notifications.Kind
		params map[string]string
		target notifications.Target
	}
	wants := []want{
		{"u1", "orderStatus", map[string]string{"number": "WQ-1", "status": "shipped"}, notifications.Target{Kind: "order", ID: "WQ-1"}},
		{"u1", "returnDecided", map[string]string{"number": "WQ-1", "approved": "true"}, notifications.Target{Kind: "order", ID: "WQ-1"}},
		{"u2", "listingDecided", map[string]string{"title": "Atomic Habits", "decision": "reject", "reason": "Blurry"}, notifications.Target{Kind: "listing", ID: "p2p-1"}},
		{"u2", "listingDecided", map[string]string{"title": "Atomic Habits", "decision": "approve"}, notifications.Target{Kind: "listing", ID: "p2p-1"}},
		{"u2", "moderationWarning", map[string]string{"strikes": "2", "max": "3"}, notifications.Target{Kind: "myListings"}},
		{"u2", "banned", map[string]string{}, notifications.Target{}},
		{"u1", "alertTriggered", map[string]string{"title": "Atomic Habits", "reason": "priceDrop"}, notifications.Target{Kind: "book", ID: "bk-atomic"}},
		{"u1", "alertTriggered", map[string]string{"title": "Atomic Habits", "reason": "backInStock"}, notifications.Target{Kind: "book", ID: "bk-atomic"}},
		{"u3", "bookWanted", map[string]string{"title": "Sapiens"}, notifications.Target{Kind: "myListings"}},
		{"u4", "bookWanted", map[string]string{"title": "Sapiens"}, notifications.Target{Kind: "myListings"}},
		{"u5", "saleSent", map[string]string{"title": "Clean Code"}, notifications.Target{Kind: "sale", ID: "HS-103"}},
		{"u6", "saleCompleted", map[string]string{"title": "Sapiens", "amount": "361"}, notifications.Target{Kind: "sale", ID: "HS-101"}},
		{"buyer", "saleSettled", map[string]string{"title": "Zero to One", "outcome": "refund", "role": "buyer", "amount": "250"}, notifications.Target{Kind: "sale", ID: "HS-104"}},
		{"seller", "saleSettled", map[string]string{"title": "Zero to One", "outcome": "refund", "role": "seller", "amount": "250"}, notifications.Target{Kind: "sale", ID: "HS-104"}},
		{"u1", "sellBackPaid", map[string]string{"title": "Sapiens", "amount": "180"}, notifications.Target{Kind: "sellBack", ID: "SB-203"}},
		{"u7", "sellBackReturned", map[string]string{"title": "Zero to One"}, notifications.Target{Kind: "sellBack", ID: "SB-202"}},
		{"u1", "newFollower", map[string]string{"name": "Arif"}, notifications.Target{Kind: "reader", ID: "p-arif"}},
		{"u1", "biteComment", map[string]string{"name": "Nabila", "excerpt": "Two-minute rule"}, notifications.Target{Kind: "bite", ID: "bt-1"}},
		{"u1", "commentReply", map[string]string{"name": "Tanvir"}, notifications.Target{Kind: "bite", ID: "bt-1"}},
	}
	if len(r.sent) != len(wants) {
		t.Fatalf("sent %d, want %d", len(r.sent), len(wants))
	}
	for i, w := range wants {
		got := r.sent[i]
		if got.user != w.user || got.kind != w.kind || !reflect.DeepEqual(got.params, w.params) {
			t.Errorf("#%d: got %s %s %v, want %s %s %v", i, got.user, got.kind, got.params, w.user, w.kind, w.params)
		}
		var gt notifications.Target
		if got.target != nil {
			gt = *got.target
		}
		if gt != w.target {
			t.Errorf("#%d target: got %+v want %+v", i, gt, w.target)
		}
	}
}

func TestKindGroupsMatchTheApp(t *testing.T) {
	cases := map[notifications.Kind]string{
		notifications.KindOrderStatus: "orders", notifications.KindReturnDecided: "orders",
		notifications.KindModerationWarning: "", notifications.KindBanned: "",
		notifications.KindAlertTriggered: "alerts",
		notifications.KindNewFollower:    "community", notifications.KindBiteComment: "community", notifications.KindCommentReply: "community",
		notifications.KindListingDecided: "usedBooks", notifications.KindSaleSent: "usedBooks", notifications.KindSaleCompleted: "usedBooks",
		notifications.KindSaleSettled: "usedBooks", notifications.KindSellBackPaid: "usedBooks", notifications.KindSellBackReturned: "usedBooks",
		notifications.KindBookWanted: "usedBooks",
	}
	for k, want := range cases {
		if got := k.Group(); got != want {
			t.Errorf("%s.Group() = %q, want %q", k, got, want)
		}
	}
}

type stepClock struct{ now time.Time }

func (c *stepClock) Now() time.Time { return c.now }
