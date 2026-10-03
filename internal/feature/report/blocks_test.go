package report_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func TestReportsFollowTheRules(t *testing.T) {
	e := testenv.New(t)
	send := func(body map[string]any) testenv.Result { return e.Call("reader", "POST", "/reports", body) }
	first := send(map[string]any{"kind": "listing", "targetId": "p2p-1", "reason": "fake"})
	if first.Status != 200 || first.Obj(t)["status"] != "open" || first.Obj(t)["reporterId"] != nil {
		t.Fatalf("report: %s", first.Raw)
	}
	again := send(map[string]any{"kind": "listing", "targetId": "p2p-1", "reason": "spam"})
	if again.Obj(t)["id"] != first.Obj(t)["id"] {
		t.Error("the open report of the same reader on the same thing is returned, not a second one")
	}
	for name, c := range map[string]struct {
		body map[string]any
		code string
	}{
		"other without a note": {map[string]any{"kind": "user", "targetId": "p-rafi", "reason": "other"}, "report_invalid"},
		"unknown reason":       {map[string]any{"kind": "user", "targetId": "p-rafi", "reason": "rude"}, "report_invalid"},
		"own listing":          {map[string]any{"kind": "listing", "targetId": "p2p-7", "reason": "spam"}, "report_target_unknown"},
		"unknown listing":      {map[string]any{"kind": "listing", "targetId": "p2p-nope", "reason": "spam"}, "report_target_unknown"},
		"myself":               {map[string]any{"kind": "user", "targetId": "u_reader", "reason": "spam"}, "report_target_unknown"},
	} {
		if r := send(c.body); r.Body != nil || r.Refusal != c.code {
			t.Errorf("%s: %q", name, r.Refusal)
		}
	}
	if r := e.Call("", "POST", "/reports", map[string]any{}); r.Status != 401 {
		t.Errorf("guest: %d", r.Status)
	}
}

func TestReportsAreRateLimitedPerReader(t *testing.T) {
	e := testenv.New(t)
	limited := false
	for i := 0; i < int(e.Deps.Cfg.ReportRatePerHour)+2; i++ {
		r := e.Call("reader", "POST", "/reports", map[string]any{"kind": "user", "targetId": "p-rafi", "reason": "spam"})
		if r.Status == 429 {
			limited = true
		}
	}
	if !limited {
		t.Error("too many reports in an hour must be refused with 429")
	}
}

func TestBlocksNewestFirst(t *testing.T) {
	e := testenv.New(t)
	e.Call("reader", "POST", "/blocks/add", map[string]any{"readerId": "p-rafi"})
	list := e.Call("reader", "POST", "/blocks/add", map[string]any{"readerId": "p-sadia"}).List(t)
	if len(list) != 2 || list[0]["id"] != "p-sadia" || list[0]["name"] != "Sadia" {
		t.Fatalf("blocks: %v", list)
	}
	for name, id := range map[string]string{"myself": "u_reader", "unknown": "p-nobody"} {
		if r := e.Call("reader", "POST", "/blocks/add", map[string]any{"readerId": id}); r.Body != nil || r.Refusal != "block_invalid" {
			t.Errorf("%s: %q", name, r.Refusal)
		}
	}
	if got := e.Call("reader", "POST", "/blocks/remove", map[string]any{"readerId": "p-sadia"}).List(t); len(got) != 1 {
		t.Errorf("unblocked: %v", got)
	}
	if got := e.Call("support", "GET", "/blocks", nil).List(t); len(got) != 0 {
		t.Error("blocks are per reader")
	}
	blocked, err := e.Deps.Report.IsBlocked(t.Context(), "p-rafi", "u_reader")
	if err != nil || !blocked {
		t.Errorf("IsBlocked works both ways: %v %v", blocked, err)
	}
}
