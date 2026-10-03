package profile

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Addresses is ProfileFakeApi.addresses. A guest gets an empty list.
func (h *Handler) Addresses(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, []Address{})
		return
	}
	list, err := h.S.Addresses(r.Context(), u.ID)
	h.list(w, r, list, err)
}

// SaveAddress is ProfileFakeApi.saveAddress.
func (h *Handler) SaveAddress(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in Address
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.SaveAddress(r.Context(), u.ID, in)
	h.list(w, r, list, err)
}

// DefaultAddress is ProfileFakeApi.defaultAddress.
func (h *Handler) DefaultAddress(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in idBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.MakeDefault(r.Context(), u.ID, in.ID)
	h.list(w, r, list, err)
}

// DeleteAddress is ProfileFakeApi.deleteAddress.
func (h *Handler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in idBody
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := h.S.DeleteAddress(r.Context(), u.ID, in.ID)
	h.list(w, r, list, err)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, list []Address, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}
