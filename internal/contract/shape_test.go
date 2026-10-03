package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

func parse(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestShape(t *testing.T) {
	cases := []struct {
		name, want, got string
		problems        int
	}{
		{"same", `{"a":1,"b":"x","c":[{"d":true}]}`, `{"a":9,"b":"y","c":[{"d":false},{"d":true}]}`, 0},
		{"extra keys are additive", `{"a":1}`, `{"a":1,"accessToken":"t"}`, 0},
		{"missing key", `{"a":1,"b":2}`, `{"a":1}`, 1},
		{"wrong kind", `{"a":1}`, `{"a":"1"}`, 1},
		{"null where golden is null", `null`, `null`, 0},
		{"object where golden is null", `null`, `{"a":1}`, 1},
		{"null where golden is object", `{"a":1}`, `null`, 1},
		{"optional field absent in golden", `{"a":null}`, `{"a":"anything"}`, 0},
		{"optional field missing in answer", `{"a":null}`, `{}`, 0},
		{"empty golden array accepts any", `[]`, `[1,2]`, 0},
		{"empty answer fails", `[{"a":1}]`, `[]`, 1},
		{"first element decides", `[{"a":1}]`, `[{"a":2},{"zzz":1}]`, 0},
		{"nested path reported", `{"x":{"y":[{"z":1}]}}`, `{"x":{"y":[{"z":"s"}]}}`, 1},
	}
	for _, c := range cases {
		got := Shape(parse(t, c.want), parse(t, c.got))
		if len(got) != c.problems {
			t.Errorf("%s: %d problems %v, want %d", c.name, len(got), got, c.problems)
		}
	}
	msg := Shape(parse(t, `{"x":{"y":[{"z":1}]}}`), parse(t, `{"x":{"y":[{"z":"s"}]}}`))
	if len(msg) != 1 || !strings.HasPrefix(msg[0], "$.x.y[0].z") {
		t.Errorf("path in message: %v", msg)
	}
}
