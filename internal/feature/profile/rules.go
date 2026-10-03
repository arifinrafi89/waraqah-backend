package profile

import (
	"regexp"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// ProfileProblem says why a profile can not be saved ("" means none).
type ProfileProblem string

// The profile problems, named as in the Dart enum.
const (
	NameLength ProfileProblem = "nameLength"
	PhoneBad   ProfileProblem = "phone"
)

// Port of ProfileRules (lib/features/profile/domain/entities/profile_rules.dart).
const (
	minName = 2
	maxName = 60
)

var (
	bdMobile  = regexp.MustCompile(`^(?:\+?880|0)(1[3-9]\d{8})$`)
	spaceDash = regexp.MustCompile(`[\s-]`)
)

// Mobile returns raw as 01... when it is a Bangladesh mobile number (01..., +880... or 880...).
func Mobile(raw string) (string, bool) {
	m := bdMobile.FindStringSubmatch(spaceDash.ReplaceAllString(raw, ""))
	if m == nil {
		return "", false
	}
	return "0" + m[1], true
}

// CheckProfile returns the first problem with name and phone (the phone may be blank).
func CheckProfile(name, phone string) ProfileProblem {
	n := textutil.TrimLen(name)
	if n < minName || n > maxName {
		return NameLength
	}
	if strings.TrimSpace(phone) != "" {
		if _, ok := Mobile(phone); !ok {
			return PhoneBad
		}
	}
	return ""
}
