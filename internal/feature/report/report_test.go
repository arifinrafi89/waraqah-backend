package report

import (
	"strings"
	"testing"
)

// Ported from test/report_rules_test.dart.
func TestOtherNeedsANoteAndNotesStayShort(t *testing.T) {
	if !Check("spam", "") {
		t.Error("spam needs no note")
	}
	if Check("other", "  ") {
		t.Error("something else needs a note")
	}
	if !Check("other", "Sells PDFs") {
		t.Error("a note is enough")
	}
	if Check("fake", strings.Repeat("x", 501)) {
		t.Error("501 characters is too long")
	}
	if !Check("fake", strings.Repeat("x", 500)) {
		t.Error("500 characters is fine")
	}
}
