package orders

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the order endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("orders endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

func (h *Handler) reply(w http.ResponseWriter, r *http.Request, v any, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, v)
}

type numberBody struct {
	Number string `json:"number"`
}

// Orders is OrderFakeApi.orders. A guest gets an empty list.
func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, []Order{})
		return
	}
	list, err := h.S.ListOf(r.Context(), u.ID)
	h.reply(w, r, list, err)
}

// Details is OrderFakeApi.details: the order of the reader, or null.
func (h *Handler) Details(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.Null(w)
		return
	}
	o, err := h.S.Details(r.Context(), u.ID, httpx.Query(r, "number"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if o == nil {
		httpx.Refuse(w, r, httpx.ErrOrderUnknown)
		return
	}
	httpx.JSON(w, o)
}

// Cancel is OrderFakeApi.cancel.
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in numberBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	o, err := h.S.Cancel(r.Context(), u.ID, in.Number)
	h.reply(w, r, o, err)
}

// Return is OrderFakeApi.requestReturn.
func (h *Handler) Return(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		Number string   `json:"number"`
		Reason string   `json:"reason"`
		Note   string   `json:"note"`
		Photos []string `json:"photos"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	o, err := h.S.RequestReturn(r.Context(), u.ID, in.Number, in.Reason, in.Note, in.Photos)
	h.reply(w, r, o, err)
}

// Reorder is OrderFakeApi.reorder.
func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in numberBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	res, err := h.S.Reorder(r.Context(), u.ID, in.Number)
	h.reply(w, r, res, err)
}

// AdminOrders is OrderAdminFakeApi.orders: every order, newest first.
func (h *Handler) AdminOrders(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.ListAll(r.Context())
	h.reply(w, r, list, err)
}

// Advance is OrderAdminFakeApi.advance.
func (h *Handler) Advance(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Number string `json:"number"`
		Status string `json:"status"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	o, err := h.S.Advance(r.Context(), in.Number, in.Status)
	h.reply(w, r, o, err)
}

// DecideReturn is OrderAdminFakeApi.decideReturn.
func (h *Handler) DecideReturn(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Number  string `json:"number"`
		Approve bool   `json:"approve"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	o, err := h.S.DecideReturn(r.Context(), in.Number, in.Approve)
	h.reply(w, r, o, err)
}

// Routes registers the order endpoints; the admin ones need the orders permission (support, super admin).
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	staff := func(f http.HandlerFunc) http.Handler { return a.StaffCan(auth.PermOrders, f) }
	r.Handle("GET /orders", a.Me(http.HandlerFunc(h.Orders)))           // OrderFakeApi.orders
	r.Handle("GET /orders/details", a.Me(http.HandlerFunc(h.Details)))  // OrderFakeApi.details
	r.Handle("POST /orders/cancel", a.Me(http.HandlerFunc(h.Cancel)))   // OrderFakeApi.cancel
	r.Handle("POST /orders/return", a.Me(http.HandlerFunc(h.Return)))   // OrderFakeApi.requestReturn
	r.Handle("POST /orders/reorder", a.Me(http.HandlerFunc(h.Reorder))) // OrderFakeApi.reorder
	r.Handle("GET /admin/orders", staff(h.AdminOrders))                 // OrderAdminFakeApi.orders
	r.Handle("POST /admin/orders/advance", staff(h.Advance))            // OrderAdminFakeApi.advance
	r.Handle("POST /admin/orders/return", staff(h.DecideReturn))        // OrderAdminFakeApi.decideReturn
}
