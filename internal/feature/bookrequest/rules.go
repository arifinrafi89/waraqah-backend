// Package bookrequest holds what readers ask for ("I am looking for this book"), matches the
// requests against the copies sellers offer, and tells staff what is in demand.
package bookrequest

import (
	"regexp"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/textutil"
)

// Port of RequestRules (lib/features/book_request/domain/entities/request_rules.dart).
const (
	MaxTitle = 120
	MaxNote  = 300
)

// Draft is what the reader sends.
type Draft struct {
	Title       string  `json:"title"`
	Author      *string `json:"author"`
	BookID      *string `json:"bookId"`
	MaxPriceBdt *int    `json:"maxPriceBdt"`
	Note        *string `json:"note"`
}

// Valid is RequestRules.check: a title of 2 to 120 characters, a maximum price above 0 when given,
// and a note of up to 300 characters.
func Valid(d Draft) bool {
	title := textutil.Len(strings.TrimSpace(d.Title))
	if title < 2 || title > MaxTitle {
		return false
	}
	if d.MaxPriceBdt != nil && *d.MaxPriceBdt <= 0 {
		return false
	}
	return d.Note == nil || textutil.Len(strings.TrimSpace(*d.Note)) <= MaxNote
}

var notWord = regexp.MustCompile("[^a-z0-9ঀ-৿]+")

func words(s string) string {
	return strings.TrimSpace(notWord.ReplaceAllString(strings.ToLower(s), " "))
}

// Matches is RequestRules.matches: whether a used copy titled listingTitle (of listingBookID) is
// the book the request asks for: the same catalog book, or the same words.
func Matches(requestTitle, requestBookID, listingTitle, listingBookID string) bool {
	if requestBookID != "" && requestBookID == listingBookID {
		return true
	}
	wanted, title := words(requestTitle), words(listingTitle)
	return wanted != "" && (strings.Contains(title, wanted) || strings.Contains(wanted, title))
}
