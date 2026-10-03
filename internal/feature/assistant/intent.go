package assistant

// Kind is AssistantIntentKind: what the reader asked for.
type Kind string

// The kinds, named as AssistantIntentKind in the app.
const (
	GeneralRecommendations Kind = "generalRecommendations"
	IslamicRecommendations Kind = "islamicRecommendations"
	Quran                  Kind = "quran"
	Hadith                 Kind = "hadith"
	Seerah                 Kind = "seerah"
	IslamicHistory         Kind = "islamicHistory"
	IslamicStudies         Kind = "islamicStudies"
	IslamicSelfDevelopment Kind = "islamicSelfDevelopment"
	AuthorSearch           Kind = "authorSearch"
	PriceFilteredSearch    Kind = "priceFilteredSearch"
	Academic               Kind = "academic"
	Other                  Kind = "other"
)

// Intent is AssistantIntent: a topic, and any budget, Class, Exam, language or format. A Basket
// request wants a set of Books that fits the budget, ready for the cart. Exam, Language and
// Format are the Dart enum names ("" when not asked for).
type Intent struct {
	Kind       Kind
	Query      string
	MaxPrice   *int
	ClassLevel *int
	Exam       string
	Language   string
	Format     string
	Basket     bool
}

// SearchesBooks is AssistantIntent.searchesBooks.
func (i Intent) SearchesBooks() bool { return i.Kind != Other }

// IsIslamic is AssistantIntent.isIslamic.
func (i Intent) IsIslamic() bool {
	switch i.Kind {
	case IslamicRecommendations, Quran, Hadith, Seerah, IslamicHistory, IslamicStudies, IslamicSelfDevelopment:
		return true
	}
	return false
}
