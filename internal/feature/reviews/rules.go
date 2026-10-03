package reviews

import "github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"

// Port of ReviewRules (review_rules.dart): one review per reader per Book, 1 to 5 stars and an
// optional text of up to 1,000 characters (graphemes).

// MaxText is the longest review text.
const MaxText = 1000

// Problem is why a review cannot be saved (ReviewProblem); "" when it can.
type Problem string

// The problems Check finds.
const (
	ProblemNone    Problem = ""
	ProblemNoStars Problem = "noStars"
	ProblemTooLong Problem = "tooLong"
)

// Check is ReviewRules.check.
func Check(stars int, text string) Problem {
	if stars < 1 || stars > 5 {
		return ProblemNoStars
	}
	if textutil.Graphemes(text) > MaxText {
		return ProblemTooLong
	}
	return ProblemNone
}
