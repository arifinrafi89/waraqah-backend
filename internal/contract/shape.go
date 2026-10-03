// Package contract checks that every endpoint answers in the same shape as the fake API.
// The goldens in testdata/contract/ come from `make contract-export`.
package contract

import (
	"fmt"
	"sort"
)

// kind names the JSON kind of a decoded value.
func kind(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	}
	return fmt.Sprintf("%T", v)
}

// Shape compares got against the golden want and returns one message per difference, each with
// its JSON path. Values may differ; what must match is:
//   - the top-level null (a golden null means the endpoint answers null),
//   - every key of a golden object (extra keys in got are fine: additive fields),
//   - the JSON kind of every value,
//   - for arrays, the shape of the first element (an empty golden array accepts any array).
//
// Inside an object a golden null means "optional, any value".
func Shape(want, got any) []string {
	if want == nil {
		if got != nil {
			return []string{"$: golden is null but the answer is " + kind(got)}
		}
		return nil
	}
	return shape("$", want, got)
}

func shape(path string, want, got any) []string {
	if want == nil {
		return nil // an optional value in the golden: anything goes
	}
	if got == nil {
		return []string{fmt.Sprintf("%s: want %s, got null", path, kind(want))}
	}
	if kind(want) != kind(got) {
		return []string{fmt.Sprintf("%s: want %s, got %s", path, kind(want), kind(got))}
	}
	switch w := want.(type) {
	case map[string]any:
		g := got.(map[string]any)
		keys := make([]string, 0, len(w))
		for k := range w {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var out []string
		for _, k := range keys {
			gv, ok := g[k]
			if !ok {
				if w[k] != nil {
					out = append(out, fmt.Sprintf("%s.%s: missing (want %s)", path, k, kind(w[k])))
				}
				continue
			}
			out = append(out, shape(path+"."+k, w[k], gv)...)
		}
		return out
	case []any:
		g := got.([]any)
		if len(w) == 0 {
			return nil
		}
		if len(g) == 0 {
			return []string{path + ": want a non-empty array, got an empty one (is the seed data loaded?)"}
		}
		return shape(path+"[0]", w[0], g[0])
	}
	return nil
}
