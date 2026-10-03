package notifications

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Routes registers the notification endpoints. Each line names the fake API constant it serves.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /notifications", a.Me(http.HandlerFunc(h.Notifications)))     // NotificationFakeApi.notifications
	r.Handle("POST /notifications/read", a.Me(http.HandlerFunc(h.Read)))        // NotificationFakeApi.read
	r.Handle("POST /notifications/read-all", a.Me(http.HandlerFunc(h.ReadAll))) // NotificationFakeApi.readAll
	r.Handle("GET /notifications/live", a.Authed(http.HandlerFunc(h.Live)))     // NotificationFakeApi.live (SSE)
}
