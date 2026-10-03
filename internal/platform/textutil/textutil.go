// Package textutil holds the small string helpers the ported Dart rules need.
package textutil

import "strings"

// Len is the length Dart's String.length gives: UTF-16 code units. The rules classes limit
// names, notes and reasons by it, so the limits mean the same thing on both sides.
func Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// TrimLen is Len of the trimmed string, for the common "name.trim().length" check.
func TrimLen(s string) int { return Len(strings.TrimSpace(s)) }
