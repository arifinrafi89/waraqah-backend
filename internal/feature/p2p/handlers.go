package p2p

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the marketplace endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("marketplace endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// viewer is the signed-in reader of the request, or "" for a guest.
func viewer(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

// Listings is P2pFakeApi.listings: `?available=true`, `?limit=4`.
func (h *Handler) Listings(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(httpx.Query(r, "limit"))
	list, err := h.S.Open(r.Context(), viewer(r), httpx.Query(r, "available") == "true", limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Mine is P2pFakeApi.mine.
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.Mine(r.Context(), viewer(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// ForBook is P2pFakeApi.forBook: `?bookId=bk-cleancode`.
func (h *Handler) ForBook(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.ForBook(r.Context(), viewer(r), httpx.Query(r, "bookId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Listing is P2pFakeApi.listing: `?id=p2p-1`; the listing or null.
func (h *Handler) Listing(w http.ResponseWriter, r *http.Request) {
	l, err := h.S.One(r.Context(), viewer(r), httpx.Query(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if l == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, l)
}

// Seller is P2pFakeApi.seller: `?id=p-nabila`; the seller page or null.
func (h *Handler) Seller(w http.ResponseWriter, r *http.Request) {
	p, err := h.S.Seller(r.Context(), viewer(r), httpx.Query(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if p == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, p)
}

// Save is P2pFakeApi.save: the listing, or null when it breaks ListingRules.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	var in SaveInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	l, err := h.S.Save(r.Context(), viewer(r), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, l)
}

// Routes registers the marketplace endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /p2p/listings", a.Public(http.HandlerFunc(h.Listings)))         // P2pFakeApi.listings
	r.Handle("GET /p2p/listings/mine", a.Me(http.HandlerFunc(h.Mine)))            // P2pFakeApi.mine
	r.Handle("GET /p2p/listings/for-book", a.Public(http.HandlerFunc(h.ForBook))) // P2pFakeApi.forBook
	r.Handle("GET /p2p/listing", a.Public(http.HandlerFunc(h.Listing)))           // P2pFakeApi.listing
	r.Handle("GET /p2p/seller", a.Public(http.HandlerFunc(h.Seller)))             // P2pFakeApi.seller
	r.Handle("POST /p2p/listings/save", a.Me(http.HandlerFunc(h.Save)))           // P2pFakeApi.save
}
