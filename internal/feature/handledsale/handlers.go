package handledsale

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the handled sale endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("handled sale endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

func viewer(r *http.Request) auth.User {
	u, _ := auth.UserFrom(r.Context())
	return u
}

// answer writes v, null when v is a nil pointer, or the refusal.
func answer[T any](h *Handler, w http.ResponseWriter, r *http.Request, v *T, err error) {
	switch {
	case err != nil:
		h.fail(w, r, err)
	case v == nil:
		httpx.Null(w)
	default:
		httpx.JSON(w, v)
	}
}

// Buy is HandledSaleFakeApi.buy: body `{listingId, method}`.
func (h *Handler) Buy(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ListingID string `json:"listingId"`
		Method    string `json:"method"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Buy(r.Context(), viewer(r).ID, in.ListingID, in.Method)
	answer(h, w, r, v, err)
}

// Mine is HandledSaleFakeApi.mine: the reader's sales, newest first.
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Mine(r.Context(), viewer(r).ID)
	answer(h, w, r, &v, err)
}

// Sale is HandledSaleFakeApi.sale: `?id=HS-101`, or null.
func (h *Handler) Sale(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Get(r.Context(), viewer(r).ID, httpx.Query(r, "id"))
	answer(h, w, r, v, err)
}

// Step is HandledSaleFakeApi.step: body `{id, step}`.
func (h *Handler) Step(w http.ResponseWriter, r *http.Request) {
	var in struct{ ID, Step string }
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Step(r.Context(), viewer(r).ID, in.ID, in.Step)
	answer(h, w, r, v, err)
}

// Dispute is HandledSaleFakeApi.dispute: body `{id, reason, note?, photos}`.
func (h *Handler) Dispute(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     string   `json:"id"`
		Reason string   `json:"reason"`
		Note   *string  `json:"note"`
		Photos []string `json:"photos"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Dispute(r.Context(), viewer(r).ID, in.ID, in.Reason, in.Note, in.Photos)
	answer(h, w, r, v, err)
}

// Earnings is HandledSaleFakeApi.earnings.
func (h *Handler) Earnings(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Earnings(r.Context(), viewer(r).ID)
	answer(h, w, r, &v, err)
}

// Payout is HandledSaleFakeApi.payout: answers the earnings, or null when nothing is ready.
func (h *Handler) Payout(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Payout(r.Context(), viewer(r).ID)
	answer(h, w, r, v, err)
}

// Disputes is HandledSaleFakeApi.disputes: open disputes, for moderators.
func (h *Handler) Disputes(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Disputes(r.Context())
	answer(h, w, r, &v, err)
}

// Settle is HandledSaleFakeApi.settle: body `{id, refund, by}`; `by` is ignored, the audit log
// names the staff member of the token. Answers the open disputes.
func (h *Handler) Settle(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     string `json:"id"`
		Refund bool   `json:"refund"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Settle(r.Context(), viewer(r).ID, in.ID, in.Refund)
	if err == nil && v == nil {
		v = []Dispute{}
	}
	answer(h, w, r, &v, err)
}

// Live is HandledSaleFakeApi.live: server-sent events `{seq, saleId}` for the reader's sales, plus
// the disputes for moderators.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	u := viewer(r)
	topics := []string{"sales:" + u.ID}
	if u.Role.CanModerate() {
		topics = append(topics, ModeratorsTopic)
	}
	h.S.SSE.ServeTopics(w, r, topics...)
}

// Routes registers the handled sale endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	me := func(f http.HandlerFunc) http.Handler { return a.Me(f) }
	moderate := func(f http.HandlerFunc) http.Handler { return a.StaffCan(auth.PermModerate, f) }
	r.Handle("POST /sales/buy", me(h.Buy))                          // HandledSaleFakeApi.buy
	r.Handle("GET /sales/mine", me(h.Mine))                         // HandledSaleFakeApi.mine
	r.Handle("GET /sales/detail", me(h.Sale))                       // HandledSaleFakeApi.sale
	r.Handle("POST /sales/step", me(h.Step))                        // HandledSaleFakeApi.step
	r.Handle("POST /sales/dispute", me(h.Dispute))                  // HandledSaleFakeApi.dispute
	r.Handle("GET /sales/earnings", me(h.Earnings))                 // HandledSaleFakeApi.earnings
	r.Handle("POST /sales/payout", me(h.Payout))                    // HandledSaleFakeApi.payout
	r.Handle("GET /sales/disputes", moderate(h.Disputes))           // HandledSaleFakeApi.disputes
	r.Handle("POST /sales/disputes/settle", moderate(h.Settle))     // HandledSaleFakeApi.settle
	r.Handle("GET /sales/live", a.Authed(http.HandlerFunc(h.Live))) // HandledSaleFakeApi.live (SSE)
}
