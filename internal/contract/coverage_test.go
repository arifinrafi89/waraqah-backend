package contract_test

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/arifinrafi89/waraqah-backend/internal/app"
	"github.com/arifinrafi89/waraqah-backend/internal/contract"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
)

var appendixRow = regexp.MustCompile("^\\|\\s*(GET|POST)(?: \\(SSE\\))?\\s*\\|\\s*`([^`]+)`\\s*\\|")

// TestRouteCoverage checks that every endpoint in Appendix A of BACKEND_PLAN.md is registered on
// the mux or listed in pending.txt, and that nothing registered is still listed as pending.
func TestRouteCoverage(t *testing.T) {
	plan, err := os.ReadFile("../../BACKEND_PLAN.md")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := contract.LoadPending(pendingFile)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.LoadFrom(os.LookupEnv)
	deps, err := app.NewDeps(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := app.Mux(deps)

	rows := map[string]bool{}
	for _, line := range regexp.MustCompile("\r?\n").Split(string(plan), -1) {
		if m := appendixRow.FindStringSubmatch(line); m != nil {
			rows[m[1]+" "+m[2]] = true
		}
	}
	if len(rows) != 174 {
		t.Fatalf("Appendix A lists %d endpoints, want 174", len(rows))
	}
	for key := range rows {
		method, path, _ := strings.Cut(key, " ")
		_, pattern := mux.Handler(httptest.NewRequest(method, apiPrefix+path, nil))
		registered := pattern != "/" && pattern != ""
		switch {
		case !registered && !pending[key]:
			t.Errorf("%s is neither registered nor in pending.txt", key)
		case registered && pending[key]:
			t.Errorf("%s is registered: remove it from pending.txt", key)
		}
	}
	for key := range pending {
		if !rows[key] {
			t.Errorf("pending.txt lists %q which is not in Appendix A", key)
		}
	}
}
