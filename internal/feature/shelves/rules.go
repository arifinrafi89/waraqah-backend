package shelves

import "time"

// Port of ProgressRules (progress_rules.dart): what a progress update needs, checked in the app
// and by the server.
const (
	MaxPages = 5000
	MinGoal  = 1
	MaxGoal  = 365
)

// Shelves, named as Shelf in the app.
const (
	WantToRead = "wantToRead"
	Reading    = "reading"
	Finished   = "finished"
)

// Problem is why a progress update cannot be saved (ProgressProblem); "" when it can.
type Problem string

// The problems Check finds.
const (
	ProblemNone       Problem = ""
	ProblemBadPercent Problem = "badPercent"
	ProblemBadPages   Problem = "badPages"
)

// Update is ProgressUpdate: a percentage, or the pages read of the Book's total.
type Update struct {
	BookID     string `json:"bookId"`
	Percent    int    `json:"percent"`
	PagesRead  *int   `json:"pagesRead"`
	TotalPages *int   `json:"totalPages"`
}

// PercentOf is ProgressRules.percentOf: rounded down, 0 to 100.
func PercentOf(pagesRead, totalPages int) int {
	if totalPages <= 0 {
		return 0
	}
	return min(max(pagesRead*100/totalPages, 0), 100)
}

// Check is ProgressRules.check.
func Check(u Update) Problem {
	if u.PagesRead != nil || u.TotalPages != nil {
		if u.PagesRead == nil || u.TotalPages == nil || *u.TotalPages < 1 || *u.TotalPages > MaxPages ||
			*u.PagesRead < 0 || *u.PagesRead > *u.TotalPages {
			return ProblemBadPages
		}
	}
	if u.Percent < 0 || u.Percent > 100 {
		return ProblemBadPercent
	}
	return ProblemNone
}

// GoalIsValid is ProgressRules.goalIsValid: 1 to 365 books a year.
func GoalIsValid(goal int) bool { return goal >= MinGoal && goal <= MaxGoal }

// day is a calendar day in the app's time zone.
func day(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }

// Streak is ProgressRules.streak: days in a row the reader read, ending today, or yesterday when
// they have not read yet today. days and now are in the app's time zone.
func Streak(days map[time.Time]bool, now time.Time) int {
	d := day(now)
	if !days[d] {
		d = d.AddDate(0, 0, -1)
	}
	n := 0
	for days[d] {
		n++
		d = d.AddDate(0, 0, -1)
	}
	return n
}
