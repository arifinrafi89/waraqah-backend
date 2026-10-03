package catalog

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the catalog endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler around a service.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("catalog endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// answer writes v, or null when v is nil.
func answer[T any](w http.ResponseWriter, v *T) {
	if v == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, v)
}

func (h *Handler) snapshot(w http.ResponseWriter, r *http.Request) (*Snapshot, bool) {
	snap, err := h.S.Store.Snapshot(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return nil, false
	}
	return snap, true
}

func csv(v string) map[string]bool {
	out := map[string]bool{}
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out[s] = true
		}
	}
	return out
}

func intParam(r *http.Request, key string) *int {
	if n, err := strconv.Atoi(httpx.Query(r, key)); err == nil {
		return &n
	}
	return nil
}

// filtersFrom reads the /books query. Hidden books need includeHidden=true and a catalog staff token.
func filtersFrom(r *http.Request) Filters {
	f := Filters{
		Query: httpx.Query(r, "q"), Category: httpx.Query(r, "category"), Section: httpx.Query(r, "section"),
		Author: httpx.Query(r, "author"), Publisher: httpx.Query(r, "publisher"), Class: intParam(r, "class"),
		Exam: httpx.Query(r, "exam"), Subject: httpx.Query(r, "subject"), Sort: httpx.Query(r, "sort"),
		MinPrice: intParam(r, "minPrice"), MaxPrice: intParam(r, "maxPrice"),
		Formats: csv(httpx.Query(r, "format")), Languages: csv(httpx.Query(r, "language")),
		InStock: httpx.QueryBool(r, "inStock"),
	}
	if v, err := strconv.ParseFloat(httpx.Query(r, "minRating"), 64); err == nil {
		f.MinRating = &v
	}
	if u, ok := auth.UserFrom(r.Context()); ok && u.Role.CanManageCatalog() {
		f.IncludeHidden = httpx.QueryBool(r, "includeHidden")
	}
	return f
}

// Books is BookFakeApi.books.
func (h *Handler) Books(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	f := filtersFrom(r)
	var sold map[string]int
	if f.Sort == "bestselling" {
		var err error
		if sold, err = h.S.Sold(r.Context()); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	httpx.JSON(w, snap.Search(f, sold))
}

// Book is BookFakeApi.book: one book, even a hidden one.
func (h *Handler) Book(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	if b, found := snap.Book(httpx.Query(r, "id")); found {
		httpx.JSON(w, b)
		return
	}
	httpx.Null(w)
}

// BookDetails is BookFakeApi.bookDetails.
func (h *Handler) BookDetails(w http.ResponseWriter, r *http.Request) {
	d, err := h.S.Details(r.Context(), httpx.Query(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	answer(w, d)
}

// LookInside is BookFakeApi.lookInside.
func (h *Handler) LookInside(w http.ResponseWriter, r *http.Request) {
	l, err := h.S.LookInside(r.Context(), httpx.Query(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	answer(w, l)
}

// PriceLows is BookFakeApi.priceLows: an object of lows by Edition id (empty for an unknown book).
func (h *Handler) PriceLows(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	b, _ := snap.Book(httpx.Query(r, "id"))
	lows, err := h.S.PriceLows(r.Context(), b)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, lows)
}

// Series is BookFakeApi.series: the series a book is in.
func (h *Handler) Series(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	answer(w, h.S.SeriesOfBook(snap, httpx.Query(r, "id")))
}

// SeriesDetail is BookFakeApi.seriesDetail.
func (h *Handler) SeriesDetail(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	answer(w, h.S.SeriesByID(snap, httpx.Query(r, "id")))
}

// UsedOptions is BookFakeApi.usedOptions; null for an unknown book.
func (h *Handler) UsedOptions(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	b, found := snap.Book(httpx.Query(r, "id"))
	if !found {
		httpx.Null(w)
		return
	}
	u, err := h.S.UsedOptionsOf(r.Context(), b)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, u)
}

func typed(r *http.Request) string {
	return strings.ToLower(strings.TrimSpace(httpx.Query(r, "q")))
}

// Suggest is BookSuggestFakeApi.suggest.
func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	if snap, ok := h.snapshot(w, r); ok {
		httpx.JSON(w, snap.Suggest(typed(r)))
	}
}

// DidYouMean is BookSuggestFakeApi.didYouMean: the one closest title, or null.
func (h *Handler) DidYouMean(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	if t := snap.DidYouMean(typed(r)); t != nil {
		httpx.JSON(w, *t)
		return
	}
	httpx.Null(w)
}
