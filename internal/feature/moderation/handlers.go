package moderation

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the Moderation Center.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("moderation endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

func answer[T any](h *Handler, w http.ResponseWriter, r *http.Request, v T, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, v)
}

// Queue is ModerationFakeApi.queue: the listings waiting for approval.
func (h *Handler) Queue(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Queue(r.Context())
	answer(h, w, r, v, err)
}

// Decide is ModerationFakeApi.decide: body `{listingId, decision, reason?, by}`. `by` is ignored:
// the log records the staff member of the token.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		ListingID string `json:"listingId"`
		Decision  string `json:"decision"`
		Reason    string `json:"reason"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Decide(r.Context(), u.ID, in.ListingID, in.Decision, in.Reason)
	answer(h, w, r, v, err)
}

// Reports is ModerationFakeApi.reports: open reports, one per reported thing.
func (h *Handler) Reports(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.OpenReports(r.Context())
	answer(h, w, r, v, err)
}

// Act is ModerationFakeApi.act: body `{reportId, action, by}`; `by` is ignored.
func (h *Handler) Act(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	var in struct {
		ReportID string `json:"reportId"`
		Action   string `json:"action"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Act(r.Context(), u.ID, in.ReportID, in.Action)
	answer(h, w, r, v, err)
}

// Log is ModerationFakeApi.log: the audit log, newest first.
func (h *Handler) Log(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.AuditLog(r.Context())
	answer(h, w, r, v, err)
}

// Routes registers the endpoints; all need the moderate permission.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	staff := func(f http.HandlerFunc) http.Handler { return a.StaffCan(auth.PermModerate, f) }
	r.Handle("GET /moderation/listings", staff(h.Queue))          // ModerationFakeApi.queue
	r.Handle("POST /moderation/listings/decide", staff(h.Decide)) // ModerationFakeApi.decide
	r.Handle("GET /moderation/reports", staff(h.Reports))         // ModerationFakeApi.reports
	r.Handle("POST /moderation/reports/act", staff(h.Act))        // ModerationFakeApi.act
	r.Handle("GET /moderation/log", staff(h.Log))                 // ModerationFakeApi.log
}
