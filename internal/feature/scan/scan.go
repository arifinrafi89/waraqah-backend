// Package scan serves the barcode scanner lookup: the book with an edition of a given ISBN.
package scan

import (
	"context"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalog"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// ScannedBook is ScannedBookModel.
type ScannedBook struct {
	BookID      string  `json:"bookId"`
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	ISBN        string  `json:"isbn"`
	CoverSeed   int     `json:"coverSeed"`
	CoverURL    *string `json:"coverUrl"`
	NewPriceBdt int     `json:"newPriceBdt"`
}

// Handler is the HTTP side of the scan endpoint.
type Handler struct{ Books catalog.Books }

// Find returns the book with an edition of that ISBN, or nil. Hidden books are not offered.
func (h *Handler) Find(ctx context.Context, isbn string) (*ScannedBook, error) {
	books, err := h.Books.Visible(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range books {
		for _, e := range b.Editions {
			if e.ISBN != nil && *e.ISBN == isbn {
				return &ScannedBook{BookID: b.ID, Title: b.Title, Author: b.Author, ISBN: isbn,
					CoverSeed: b.CoverSeed, CoverURL: b.CoverURL, NewPriceBdt: b.FromPriceBdt()}, nil
			}
		}
	}
	return nil, nil
}

// LookUp is ScanFakeApi.lookUp: `?isbn=9789840001491`, or null when Waraqah does not have it.
func (h *Handler) LookUp(w http.ResponseWriter, r *http.Request) {
	found, err := h.Find(r.Context(), httpx.Query(r, "isbn"))
	if err != nil {
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
		return
	}
	if found == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, found)
}

// Routes registers the scan endpoint (public).
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /scan/lookup", a.Public(http.HandlerFunc(h.LookUp))) // ScanFakeApi.lookUp
}
