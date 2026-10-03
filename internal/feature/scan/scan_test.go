package scan_test

import (
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/testenv"
)

func TestScanLookup(t *testing.T) {
	e := testenv.New(t)
	got := e.Call("", "GET", "/scan/lookup?isbn=9789840001491", nil).Obj(t)
	if got["bookId"] == nil || got["isbn"] != "9789840001491" || got["newPriceBdt"].(float64) <= 0 || got["title"] == "" {
		t.Errorf("found: %v", got)
	}
	if r := e.Call("", "GET", "/scan/lookup?isbn=9780000000002", nil); r.Status != 200 || r.Body != nil {
		t.Errorf("unknown isbn: %d %s", r.Status, r.Raw)
	}
	// a hidden book is not offered to the scanner
	book := got["bookId"].(string)
	e.Call("catalog", "POST", "/admin/catalog/books/hide", map[string]any{"id": book, "hidden": true})
	if r := e.Call("", "GET", "/scan/lookup?isbn=9789840001491", nil); r.Body != nil {
		t.Errorf("hidden book scanned: %s", r.Raw)
	}
}
