package contract

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Request is the request a golden was recorded with.
type Request struct {
	Method string         `json:"method"`
	Path   string         `json:"path"`
	Query  map[string]any `json:"query"`
	Body   any            `json:"body"`
	As     string         `json:"as"`
	SSE    bool           `json:"sse"`
}

// Golden is one recorded request and the fake API's answer.
type Golden struct {
	Request  Request `json:"request"`
	Response any     `json:"response"`
	SSE      bool    `json:"sse"`
	File     string  `json:"-"`
}

// Key is "METHOD /path": what pending.txt lists.
func (r Request) Key() string { return r.Method + " " + r.Path }

// URL is the request path with its query string.
func (r Request) URL(prefix string) string {
	q := url.Values{}
	for k, v := range r.Query {
		q.Set(k, fmt.Sprint(v))
	}
	u := prefix + r.Path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

// LoadGoldens reads the goldens in the order the exporter ran them (_index.json), so a
// replay meets the same state the fake API had.
func LoadGoldens(dir string) ([]Golden, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "_index.json"))
	if err != nil {
		return nil, fmt.Errorf("read _index.json: %w (run make contract-export)", err)
	}
	var index []struct {
		File string `json:"file"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, err
	}
	out := make([]Golden, 0, len(index))
	for _, e := range index {
		b, err := os.ReadFile(filepath.Join(dir, e.File))
		if err != nil {
			return nil, err
		}
		var g Golden
		if err := json.Unmarshal(b, &g); err != nil {
			return nil, fmt.Errorf("%s: %w", e.File, err)
		}
		g.File = e.File
		out = append(out, g)
	}
	return out, nil
}

// LoadPending reads pending.txt: endpoints not built yet, one "METHOD /path" per line.
func LoadPending(path string) (map[string]bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out[line] = true
		}
	}
	return out, nil
}

// SortedKeys lists a set in a stable order.
func SortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
