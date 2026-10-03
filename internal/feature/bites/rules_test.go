package bites

import (
	"strings"
	"testing"
)

// Ported from test/bite_rules_test.dart.
func TestBiteRules(t *testing.T) {
	cases := []struct {
		name string
		got  Problem
		want Problem
	}{
		{"500 characters", Check(strings.Repeat("a", 500), "", false), ProblemNone},
		{"501 characters", Check(strings.Repeat("a", 501), "", false), ProblemTooLong},
		{"blank", Check("   ", "", false), ProblemEmpty},
		{"500 conjuncts", Check(strings.Repeat("ক্ষ", 500), "", false), ProblemNone},
		{"501 conjuncts", Check(strings.Repeat("ক্ষ", 501), "", false), ProblemTooLong},
		{"spoiler without a book", Check("x", "", true), ProblemSpoilerNeedsBook},
		{"spoiler with a book", Check("x", "bk-1", true), ProblemNone},
		{"comment of 300", CheckComment(strings.Repeat("a", 300)), ProblemNone},
		{"comment of 301", CheckComment(strings.Repeat("a", 301)), ProblemTooLong},
		{"empty comment", CheckComment(""), ProblemEmpty},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %q, want %q", c.name, c.got, c.want)
		}
	}
	if n := Length(strings.Repeat("ক্ষ", 3)); n != 3 {
		t.Errorf("a Bangla conjunct counts as one character: %d", n)
	}
}
