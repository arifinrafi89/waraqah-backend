package cart

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the cart endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.S.Log.Error("cart endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// Cart is CartFakeApi.cart. A guest gets the empty cart.
func (h *Handler) Cart(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, Cart{Lines: []Line{}})
		return
	}
	c, err := h.S.Get(r.Context(), u.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, c)
}

// Add is CartFakeApi.add. An item that cannot be added leaves the cart as it is, and the answer
// carries the refusal code in X-Waraqah-Error (the app always expects a cart).
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	c, refused, err := h.S.Add(r.Context(), u.ID, in.Kind, in.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if refused != "" {
		w.Header().Set(httpx.ErrorHeader, refused)
	}
	httpx.JSON(w, c)
}

// Update is CartFakeApi.update.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		LineID   string `json:"lineId"`
		Quantity int    `json:"quantity"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	c, err := h.S.SetQuantity(r.Context(), u.ID, in.LineID, in.Quantity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, c)
}

// Remove is CartFakeApi.remove.
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		LineID string `json:"lineId"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	c, err := h.S.Remove(r.Context(), u.ID, in.LineID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, c)
}

// Routes registers the cart endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /cart", a.Me(http.HandlerFunc(h.Cart)))           // CartFakeApi.cart
	r.Handle("POST /cart/add", a.Me(http.HandlerFunc(h.Add)))       // CartFakeApi.add
	r.Handle("POST /cart/update", a.Me(http.HandlerFunc(h.Update))) // CartFakeApi.update
	r.Handle("POST /cart/remove", a.Me(http.HandlerFunc(h.Remove))) // CartFakeApi.remove
}
