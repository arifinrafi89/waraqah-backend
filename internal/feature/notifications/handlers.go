package notifications

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the notification endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler around a service.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) internal(w http.ResponseWriter, r *http.Request, err error) {
	h.S.Log.Error("notifications endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

func (h *Handler) answerList(w http.ResponseWriter, r *http.Request, userID string) {
	list, err := h.S.List(r.Context(), userID)
	if err != nil {
		h.internal(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Notifications is NotificationFakeApi.notifications. A guest gets an empty list.
func (h *Handler) Notifications(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.JSON(w, []Notification{})
		return
	}
	h.answerList(w, r, u.ID)
}

// Read is NotificationFakeApi.read.
func (h *Handler) Read(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		ID string `json:"id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	found, err := h.S.MarkRead(r.Context(), u.ID, in.ID)
	if err != nil {
		h.internal(w, r, err)
		return
	}
	if !found {
		httpx.Refuse(w, r, httpx.ErrNotificationUnknown)
		return
	}
	h.answerList(w, r, u.ID)
}

// ReadAll is NotificationFakeApi.readAll.
func (h *Handler) ReadAll(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if err := h.S.MarkAllRead(r.Context(), u.ID); err != nil {
		h.internal(w, r, err)
		return
	}
	h.answerList(w, r, u.ID)
}

// Live is NotificationFakeApi.live: one data: {"unread": n} event per change.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	h.S.SSE.Serve(w, r, Topic(u.ID))
}
