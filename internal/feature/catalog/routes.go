package catalog

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Routes registers the catalog endpoints. Each line names the fake API constant it serves.
// Public endpoints still read an optional token (staff see hidden books, booklists know isMine).
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	pub := func(f http.HandlerFunc) http.Handler { return a.Public(f) }
	me := func(f http.HandlerFunc) http.Handler { return a.Me(f) }

	r.Handle("GET /books", pub(h.Books))                         // BookFakeApi.books
	r.Handle("GET /books/detail", pub(h.Book))                   // BookFakeApi.book
	r.Handle("GET /books/details", pub(h.BookDetails))           // BookFakeApi.bookDetails
	r.Handle("GET /books/look-inside", pub(h.LookInside))        // BookFakeApi.lookInside
	r.Handle("GET /books/price-lows", pub(h.PriceLows))          // BookFakeApi.priceLows
	r.Handle("GET /books/series", pub(h.Series))                 // BookFakeApi.series
	r.Handle("GET /books/used-options", pub(h.UsedOptions))      // BookFakeApi.usedOptions
	r.Handle("GET /books/suggest", pub(h.Suggest))               // BookSuggestFakeApi.suggest
	r.Handle("GET /books/did-you-mean", pub(h.DidYouMean))       // BookSuggestFakeApi.didYouMean
	r.Handle("GET /books/questions", pub(h.BookQuestions))       // BookQuestionsFakeApi.questions
	r.Handle("POST /books/questions/ask", me(h.Ask))             // BookQuestionsFakeApi.ask
	r.Handle("POST /books/questions/answer", me(h.Answer))       // BookQuestionsFakeApi.answer
	r.Handle("GET /categories", pub(h.Categories))               // BookFakeApi.categories
	r.Handle("GET /subjects", pub(h.Subjects))                   // BookFakeApi.subjects
	r.Handle("GET /authors/detail", pub(h.Author))               // BookFakeApi.author
	r.Handle("GET /publishers/detail", pub(h.Publisher))         // BookFakeApi.publisher
	r.Handle("GET /series/detail", pub(h.SeriesDetail))          // BookFakeApi.seriesDetail
	r.Handle("GET /collections", pub(h.Collections))             // CollectionFakeApi.collections
	r.Handle("GET /collections/detail", pub(h.CollectionDetail)) // CollectionFakeApi.detail
	r.Handle("GET /experts", pub(h.Experts))                     // CollectionFakeApi.experts
	r.Handle("GET /experts/detail", pub(h.ExpertDetail))         // CollectionFakeApi.expert
	r.Handle("GET /booklists", pub(h.Booklists))                 // BooklistFakeApi.booklists
	r.Handle("GET /booklists/detail", pub(h.BooklistDetail))     // BooklistFakeApi.detail
	r.Handle("POST /booklists/mine/save", me(h.SaveMine))        // BooklistFakeApi.saveMine
	r.Handle("POST /booklists/mine/delete", me(h.DeleteMine))    // BooklistFakeApi.deleteMine
}
