package catalogadmin

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// SaveCollection is CatalogAdminFakeApi.saveCollection.
func (h *Handler) SaveCollection(w http.ResponseWriter, r *http.Request) {
	var d ListDraft
	if !httpx.Decode(w, r, &d) {
		return
	}
	out, err := h.S.SaveCollection(r.Context(), d)
	h.reply(w, r, out, err)
}

// DeleteCollection is CatalogAdminFakeApi.deleteCollection.
func (h *Handler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	var in idBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	out, err := h.S.DeleteCollection(r.Context(), in.ID)
	h.reply(w, r, out, err)
}

// SaveBooklist is CatalogAdminFakeApi.saveBooklist.
func (h *Handler) SaveBooklist(w http.ResponseWriter, r *http.Request) {
	var d ListDraft
	if !httpx.Decode(w, r, &d) {
		return
	}
	out, err := h.S.SaveBooklist(r.Context(), d)
	h.reply(w, r, out, err)
}

// DeleteBooklist is CatalogAdminFakeApi.deleteBooklist.
func (h *Handler) DeleteBooklist(w http.ResponseWriter, r *http.Request) {
	var in idBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	out, err := h.S.DeleteBooklist(r.Context(), in.ID)
	h.reply(w, r, out, err)
}

// IsbnLookup is CatalogToolsFakeApi.isbnLookup.
func (h *Handler) IsbnLookup(w http.ResponseWriter, r *http.Request) {
	res, err := h.S.LookUpIsbn(r.Context(), httpx.Query(r, "isbn"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if res == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, res)
}

// LowStock is CatalogToolsFakeApi.lowStock.
func (h *Handler) LowStock(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.LowStock(r.Context())
	h.reply(w, r, list, err)
}

// EditionStock is CatalogToolsFakeApi.editionStock.
func (h *Handler) EditionStock(w http.ResponseWriter, r *http.Request) {
	var in StockResult
	if !httpx.Decode(w, r, &in) {
		return
	}
	out, err := h.S.SetStock(r.Context(), in.EditionID, in.Stock)
	h.reply(w, r, out, err)
}

// Import is CatalogToolsFakeApi.importBooks.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Books []ImportRow `json:"books"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	out, err := h.S.Import(r.Context(), in.Books)
	h.reply(w, r, out, err)
}
