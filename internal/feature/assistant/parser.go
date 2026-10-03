package assistant

import (
	"regexp"
	"strconv"
	"strings"
)

// Port of AssistantParser (assistant_parser.dart): reads a prompt in plain English, Banglish or
// Bangla, such as "short seerah for beginners in Bangla", "Books for Class 9 under ৳1,000" or
// "৫০০ টাকার নিচে বই".

const banglaDigits = "০১২৩৪৫৬৭৮৯"

var (
	thousands   = regexp.MustCompile(`(\d),(\d{3})`)
	authorRe    = regexp.MustCompile(`\bby\s+(.+)$`)
	priceBefore = regexp.MustCompile(`(?:under|below|less than|within|up to|max)\s*(?:৳|tk|taka|bdt)?\s*(\d+)`)
	priceAfter  = regexp.MustCompile(`(\d+)\s*(?:৳|tk|taka|টাকা)?\s*(?:র|এর|-এর)?\s*(?:নিচে|মধ্যে)`)
	classRe     = regexp.MustCompile(`class\s*(\d{1,2})`)
	classBnRe   = regexp.MustCompile(`(\d{1,2})\s*(?:ম|তম)?\s*শ্রেণি`)
)

// normalize is AssistantParser._normalize: lower case, Bangla digits as ASCII, "1,000" as "1000".
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if i := strings.IndexRune(banglaDigits, r); i >= 0 {
			b.WriteByte(byte('0' + i/len("০")))
			continue
		}
		b.WriteRune(r)
	}
	return thousands.ReplaceAllString(b.String(), "$1$2")
}

func hasAny(text string, terms ...string) bool {
	for _, t := range terms {
		if strings.Contains(text, t) {
			return true
		}
	}
	return false
}

func price(text string) *int {
	m := priceBefore.FindStringSubmatch(text)
	if m == nil {
		m = priceAfter.FindStringSubmatch(text)
	}
	if m == nil {
		return nil
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return nil
	}
	return &n
}

func classLevel(text string) *int {
	m := classRe.FindStringSubmatch(text)
	if m == nil {
		m = classBnRe.FindStringSubmatch(text)
	}
	if m == nil {
		return nil
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 6 || n > 12 {
		return nil
	}
	return &n
}

// Parse is AssistantParser.parse.
func Parse(prompt string, history []string) Intent {
	text := normalize(prompt)
	has := func(terms ...string) bool { return hasAny(text, terms...) }
	in := Intent{MaxPrice: price(text), ClassLevel: classLevel(text)}
	switch {
	case has("ssc", "এসএসসি"):
		in.Exam = "ssc"
	case has("hsc", "এইচএসসি"):
		in.Exam = "hsc"
	case has("admission", "ভর্তি"):
		in.Exam = "admission"
	case has("bcs", "বিসিএস"):
		in.Exam = "bcs"
	}
	switch {
	case has("bangla", "bengali", "বাংলা"):
		in.Language = "bangla"
	case has("english", "ইংরেজি"):
		in.Language = "english"
	case has("arabic", "আরবি"):
		in.Language = "arabic"
	}
	switch {
	case has("ebook", "e-book", "ই-বুক"):
		in.Format = "ebook"
	case has("hardcover", "hard cover"):
		in.Format = "hardcover"
	case has("paperback"):
		in.Format = "paperback"
	}
	author := authorRe.FindStringSubmatch(strings.TrimSpace(prompt))
	switch {
	case has("quran", "qur'an", "tafsir", "কুরআন", "তাফসির"):
		in.Kind = Quran
	case has("hadith", "hadees", "হাদিস"):
		in.Kind = Hadith
	case has("seerah", "sirah", "prophet", "সীরাহ", "সিরাত"):
		in.Kind = Seerah
	case has("islamic history", "muslim history", "ইসলামের ইতিহাস"):
		in.Kind = IslamicHistory
	case has("fiqh", "aqeedah", "islamic studies"):
		in.Kind = IslamicStudies
	case has("islam", "muslim", "ইসলাম"):
		in.Kind = IslamicRecommendations
	case author != nil:
		in.Kind = AuthorSearch
	case in.ClassLevel != nil || in.Exam != "":
		in.Kind = Academic
	case in.MaxPrice != nil:
		in.Kind = PriceFilteredSearch
	case has("book", "read", "suggest", "recommend", "বই") || in.Language != "" || in.Format != "":
		in.Kind = GeneralRecommendations
	case has("more", "another", "আরও") && historyMentionsBooks(history):
		in.Kind = GeneralRecommendations
	default:
		in.Kind = Other
	}
	if author != nil {
		in.Query = strings.TrimSpace(author[1])
	}
	in.Basket = in.ClassLevel != nil || in.Exam != "" || has("basket", "list", "all books", "সব বই", "তালিকা")
	return in
}

func historyMentionsBooks(history []string) bool {
	for _, h := range history {
		if strings.Contains(normalize(h), "book") {
			return true
		}
	}
	return false
}
