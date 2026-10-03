package checkout

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the checkout endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("checkout endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// Coupon is CheckoutFakeApi.coupon: `?code=EID100`; the coupon (expired too, so the app can say
// why) or null.
func (h *Handler) Coupon(w http.ResponseWriter, r *http.Request) {
	c, err := h.S.FindCoupon(r.Context(), h.S.DB.Q(), httpx.Query(r, "code"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if c == nil {
		httpx.Null(w)
		return
	}
	httpx.JSON(w, c)
}

// PlaceOrder is CheckoutFakeApi.placeOrder.
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in PlaceInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	receipt, err := h.S.Place(r.Context(), u.ID, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, receipt)
}

// Coupons is CouponAdminFakeApi.coupons: every coupon, newest first.
func (h *Handler) Coupons(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.Coupons(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// CreateCoupon is CouponAdminFakeApi.create: the updated list.
func (h *Handler) CreateCoupon(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in CouponModel
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.CreateCoupon(r.Context(), u.ID, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Routes registers the checkout endpoints; the coupon admin ones need the orders permission.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	staff := func(f http.HandlerFunc) http.Handler { return a.StaffCan(auth.PermOrders, f) }
	r.Handle("GET /coupons/check", a.Me(http.HandlerFunc(h.Coupon)))     // CheckoutFakeApi.coupon
	r.Handle("POST /orders/place", a.Me(http.HandlerFunc(h.PlaceOrder))) // CheckoutFakeApi.placeOrder
	r.Handle("GET /admin/coupons", staff(h.Coupons))                     // CouponAdminFakeApi.coupons
	r.Handle("POST /admin/coupons/create", staff(h.CreateCoupon))        // CouponAdminFakeApi.create
}
