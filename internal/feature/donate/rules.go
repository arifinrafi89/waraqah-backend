// Package donate lists the verified places that take donated books and saves donation orders.
package donate

import (
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// Port of PlaceRules (lib/features/donate/domain/entities/donate_place_draft.dart).
const (
	MaxName   = 80
	MaxStory  = 300
	MaxCopies = 100
)

// Kinds a place can be (RecipientKind).
var Kinds = []string{"library", "school", "madrasa", "orphanage"}

// NeedDraft is one book a place asks for, and how many copies.
type NeedDraft struct {
	BookID string `json:"bookId"`
	Wanted int    `json:"wanted"`
}

// PlaceDraft is a verified place as Staff edit it. An empty ID is a new place.
type PlaceDraft struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Kind     string      `json:"kind"`
	District string      `json:"district"`
	Area     string      `json:"area"`
	Story    string      `json:"story"`
	Needs    []NeedDraft `json:"needs"`
}

// Problem is what is wrong with a draft; empty when it can be saved.
type Problem string

// The problems PlaceRules finds, in the order it checks them.
const (
	ProblemNone     Problem = ""
	ProblemName     Problem = "name"
	ProblemDistrict Problem = "district"
	ProblemArea     Problem = "area"
	ProblemStory    Problem = "story"
	ProblemNoNeeds  Problem = "noNeeds"
	ProblemBadCount Problem = "badCount"
)

// Check is PlaceRules.check: what a verified place needs before Staff save it.
func Check(d PlaceDraft) Problem {
	if n := textutil.Len(strings.TrimSpace(d.Name)); n < 3 || n > MaxName {
		return ProblemName
	}
	if strings.TrimSpace(d.District) == "" {
		return ProblemDistrict
	}
	if strings.TrimSpace(d.Area) == "" {
		return ProblemArea
	}
	if n := textutil.Len(strings.TrimSpace(d.Story)); n < 10 || n > MaxStory {
		return ProblemStory
	}
	if len(d.Needs) == 0 {
		return ProblemNoNeeds
	}
	for _, n := range d.Needs {
		if n.Wanted < 1 || n.Wanted > MaxCopies {
			return ProblemBadCount
		}
	}
	return ProblemNone
}

// kindOrDefault is what the app does with an unknown kind: a library.
func kindOrDefault(k string) string {
	for _, v := range Kinds {
		if v == k {
			return k
		}
	}
	return "library"
}
