package donate

import (
	"strings"
	"testing"
)

// Ported from test/donation_places_test.dart (PlaceRules).
func valid() PlaceDraft {
	return PlaceDraft{Name: "Aloghar Library", Kind: "library", District: "Rangpur", Area: "Pirgachha",
		Story: "A reading room for village children.", Needs: []NeedDraft{{BookID: "bk-hobbit", Wanted: 5}}}
}

func TestPlaceRules(t *testing.T) {
	if p := Check(valid()); p != ProblemNone {
		t.Fatalf("a good place: %q", p)
	}
	cases := map[string]struct {
		change func(*PlaceDraft)
		want   Problem
	}{
		"short name":  {func(d *PlaceDraft) { d.Name = " ab " }, ProblemName},
		"long name":   {func(d *PlaceDraft) { d.Name = strings.Repeat("x", MaxName+1) }, ProblemName},
		"district":    {func(d *PlaceDraft) { d.District = "  " }, ProblemDistrict},
		"area":        {func(d *PlaceDraft) { d.Area = "" }, ProblemArea},
		"short story": {func(d *PlaceDraft) { d.Story = "too short" }, ProblemStory},
		"long story":  {func(d *PlaceDraft) { d.Story = strings.Repeat("x", MaxStory+1) }, ProblemStory},
		"no needs":    {func(d *PlaceDraft) { d.Needs = nil }, ProblemNoNeeds},
		"zero copies": {func(d *PlaceDraft) { d.Needs[0].Wanted = 0 }, ProblemBadCount},
		"too many":    {func(d *PlaceDraft) { d.Needs[0].Wanted = MaxCopies + 1 }, ProblemBadCount},
	}
	for name, c := range cases {
		d := valid()
		c.change(&d)
		if got := Check(d); got != c.want {
			t.Errorf("%s: got %q want %q", name, got, c.want)
		}
	}
	d := valid()
	d.Needs[0].Wanted = MaxCopies
	if Check(d) != ProblemNone {
		t.Error("100 copies is allowed")
	}
}
