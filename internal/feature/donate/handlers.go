package donate

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the donate endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("donate endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// Recipients is DonateFakeApi.recipients.
func (h *Handler) Recipients(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.All(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Recipient is DonateFakeApi.recipient: `?id=rc-aloghar`; the place or null.
func (h *Handler) Recipient(w http.ResponseWriter, r *http.Request) {
	p, err := h.S.One(r.Context(), httpx.Query(r, "id"))
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

// Give is DonateFakeApi.give.
func (h *Handler) Give(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in GiveInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	d, err := h.S.Give(r.Context(), u.ID, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, d)
}

// Save is DonateAdminFakeApi.save: every place, or null.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	var in PlaceDraft
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.Save(r.Context(), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Remove is DonateAdminFakeApi.remove: the places left, or null.
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.Remove(r.Context(), in.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Routes registers the donate endpoints; the admin ones need the orders permission.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	staff := func(f http.HandlerFunc) http.Handler { return a.StaffCan(auth.PermOrders, f) }
	r.Handle("GET /donate/recipients", a.Public(http.HandlerFunc(h.Recipients))) // DonateFakeApi.recipients
	r.Handle("GET /donate/recipient", a.Public(http.HandlerFunc(h.Recipient)))   // DonateFakeApi.recipient
	r.Handle("POST /donate/give", a.Me(http.HandlerFunc(h.Give)))                // DonateFakeApi.give
	r.Handle("POST /admin/donate/places/save", staff(h.Save))                    // DonateAdminFakeApi.save
	r.Handle("POST /admin/donate/places/remove", staff(h.Remove))                // DonateAdminFakeApi.remove
}
