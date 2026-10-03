package bites

import "github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"

// Port of BiteRules (bite_rules.dart): limits the composer shows and the server checks. Lengths
// count graphemes, so a Bangla conjunct like ক্ষ is one character.
const (
	MaxLength  = 500
	MaxComment = 300
	MaxQuote   = 300
)

// Problem is why a Bite or comment cannot be saved (BiteProblem); "" when it can.
type Problem string

// The problems the rules find.
const (
	ProblemNone             Problem = ""
	ProblemEmpty            Problem = "empty"
	ProblemTooLong          Problem = "tooLong"
	ProblemSpoilerNeedsBook Problem = "spoilerNeedsBook"
)

// Length is BiteRules.length: the trimmed text in graphemes.
func Length(text string) int { return textutil.Graphemes(text) }

// Check is BiteRules.check: 1 to 500 characters, and a spoiler must tag a Book.
func Check(text, bookID string, spoiler bool) Problem {
	if p := limit(text, MaxLength); p != ProblemNone {
		return p
	}
	if spoiler && bookID == "" {
		return ProblemSpoilerNeedsBook
	}
	return ProblemNone
}

// CheckComment is BiteRules.checkComment: a comment or reply is 1 to 300 characters.
func CheckComment(text string) Problem { return limit(text, MaxComment) }

// CheckQuote is BiteRules.checkQuote: a quote card's text is 1 to 300 characters.
func CheckQuote(text string) Problem { return limit(text, MaxQuote) }

func limit(text string, max int) Problem {
	switch n := Length(text); {
	case n == 0:
		return ProblemEmpty
	case n > max:
		return ProblemTooLong
	}
	return ProblemNone
}
