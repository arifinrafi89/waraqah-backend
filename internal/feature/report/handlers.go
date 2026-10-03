package report

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the report and block endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("report endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

func viewer(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

// Report is ReportFakeApi.report: body `{kind, targetId, reason, note?}`; the report.
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind     string `json:"kind"`
		TargetID string `json:"targetId"`
		Reason   string `json:"reason"`
		Note     string `json:"note"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	rep, err := h.S.Send(r.Context(), viewer(r), in.Kind, in.TargetID, in.Reason, in.Note)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, rep)
}

// Blocked is ReportFakeApi.blocked.
func (h *Handler) Blocked(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.Blocked(r.Context(), viewer(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

func (h *Handler) change(w http.ResponseWriter, r *http.Request, f func(*Service, *http.Request, string) ([]Blocked, error)) {
	var in struct {
		ReaderID string `json:"readerId"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	list, err := f(h.S, r, in.ReaderID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// Block is ReportFakeApi.block: body `{readerId}`; the blocked list.
func (h *Handler) Block(w http.ResponseWriter, r *http.Request) {
	h.change(w, r, func(s *Service, r *http.Request, id string) ([]Blocked, error) {
		return s.Block(r.Context(), viewer(r), id)
	})
}

// Unblock is ReportFakeApi.unblock: body `{readerId}`; the blocked list.
func (h *Handler) Unblock(w http.ResponseWriter, r *http.Request) {
	h.change(w, r, func(s *Service, r *http.Request, id string) ([]Blocked, error) {
		return s.Unblock(r.Context(), viewer(r), id)
	})
}

// Routes registers the endpoints. Sending reports is rate limited per reader (REPORT_RATE_PER_HOUR).
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware, perUser httpx.Middleware) {
	send := perUser(http.HandlerFunc(h.Report))
	r.Handle("POST /reports", a.Me(send))                              // ReportFakeApi.report
	r.Handle("GET /blocks", a.Me(http.HandlerFunc(h.Blocked)))         // ReportFakeApi.blocked
	r.Handle("POST /blocks/add", a.Me(http.HandlerFunc(h.Block)))      // ReportFakeApi.block
	r.Handle("POST /blocks/remove", a.Me(http.HandlerFunc(h.Unblock))) // ReportFakeApi.unblock
}
