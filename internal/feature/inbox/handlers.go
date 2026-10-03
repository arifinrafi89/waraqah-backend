package inbox

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/moderation"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the inbox endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("inbox endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

func viewer(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

func answer[T any](h *Handler, w http.ResponseWriter, r *http.Request, v T, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, v)
}

// Threads is InboxFakeApi.threads: `?listingId=` keeps those about one listing.
func (h *Handler) Threads(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.List(r.Context(), viewer(r), httpx.Query(r, "listingId"))
	answer(h, w, r, v, err)
}

// Thread is InboxFakeApi.thread: `?id=`; the whole thread, or null.
func (h *Handler) Thread(w http.ResponseWriter, r *http.Request) {
	t, err := h.S.Get(r.Context(), viewer(r), httpx.Query(r, "id"))
	if err == nil && t == nil {
		httpx.Null(w)
		return
	}
	answer(h, w, r, t, err)
}

// Open is InboxFakeApi.open: body `{listingId}`.
func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ListingID string `json:"listingId"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Open(r.Context(), viewer(r), in.ListingID)
	answer(h, w, r, v, err)
}

// Send is InboxFakeApi.send: body `{threadId, text}`.
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	var in struct{ ThreadID, Text string }
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Send(r.Context(), viewer(r), in.ThreadID, in.Text)
	answer(h, w, r, v, err)
}

// Offer is InboxFakeApi.offer: body `{listingId, amountBdt, handover}`.
func (h *Handler) Offer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ListingID string `json:"listingId"`
		AmountBdt int    `json:"amountBdt"`
		Handover  string `json:"handover"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Offer(r.Context(), viewer(r), in.ListingID, in.AmountBdt, in.Handover)
	answer(h, w, r, v, err)
}

// Decide is InboxFakeApi.decide: body `{threadId, offerId, accept}`.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ThreadID string `json:"threadId"`
		OfferID  string `json:"offerId"`
		Accept   bool   `json:"accept"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Decide(r.Context(), viewer(r), in.ThreadID, in.OfferID, in.Accept)
	answer(h, w, r, v, err)
}

// byThread handles the endpoints whose body is only `{threadId}`.
func (h *Handler) byThread(f func(*Service, context.Context, string, string) (*Thread, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			ThreadID string `json:"threadId"`
		}
		if !httpx.Decode(w, r, &in) {
			return
		}
		v, err := f(h.S, r.Context(), viewer(r), in.ThreadID)
		answer(h, w, r, v, err)
	}
}

// Rate is InboxFakeApi.rate: body `{threadId, stars, comment?}`.
func (h *Handler) Rate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ThreadID string `json:"threadId"`
		Stars    int    `json:"stars"`
		Comment  string `json:"comment"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Rate(r.Context(), viewer(r), in.ThreadID, in.Stars, in.Comment)
	answer(h, w, r, v, err)
}

// Live is InboxFakeApi.live: server-sent events, one `data: {seq, threadId, listingId}` per change
// of a thread of the signed-in reader.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	h.S.SSE.Serve(w, r, "inbox:"+viewer(r))
}

// Routes registers the inbox endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	me := func(f http.HandlerFunc) http.Handler { return a.Me(f) }
	r.Handle("GET /inbox", me(h.Threads))                                       // InboxFakeApi.threads
	r.Handle("GET /inbox/thread", me(h.Thread))                                 // InboxFakeApi.thread
	r.Handle("GET /inbox/live", a.Authed(http.HandlerFunc(h.Live)))             // InboxFakeApi.live (SSE)
	r.Handle("POST /inbox/open", me(h.Open))                                    // InboxFakeApi.open
	r.Handle("POST /inbox/send", me(h.Send))                                    // InboxFakeApi.send
	r.Handle("POST /inbox/offer", me(h.Offer))                                  // InboxFakeApi.offer
	r.Handle("POST /inbox/offer/decide", me(h.Decide))                          // InboxFakeApi.decide
	r.Handle("POST /inbox/read", me(h.byThread((*Service).Read)))               // InboxFakeApi.read
	r.Handle("POST /inbox/listing/release", me(h.byThread((*Service).Release))) // InboxFakeApi.release
	r.Handle("POST /inbox/listing/sold", me(h.byThread((*Service).Sold)))       // InboxFakeApi.sold
	r.Handle("POST /inbox/rate", me(h.Rate))                                    // InboxFakeApi.rate
}

// Messages is the moderation subject of kind "message": a reported chat message.
type Messages struct{ S *Service }

var _ moderation.Subject = Messages{}

// Find looks a message up for a report: its text and who wrote it.
func (m Messages) Find(ctx context.Context, id string) (*moderation.Found, error) {
	q := m.S.DB.Q()
	msg, err := q.GetMessage(ctx, id)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	name := "?"
	if people, err := q.ListUserNames(ctx, []string{msg.AuthorID}); err == nil && len(people) > 0 {
		name = people[0].Name
	}
	return &moderation.Found{Preview: msg.Text.String, OwnerID: msg.AuthorID, OwnerName: name}, nil
}

// Remove takes a removed message out of its thread; listening apps are told a moment after the commit.
func (m Messages) Remove(ctx context.Context, q *sqlc.Queries, id string) error {
	threadID, err := q.DeleteMessage(ctx, id)
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	t, err := q.GetThread(ctx, threadID)
	if err != nil {
		return err
	}
	time.AfterFunc(300*time.Millisecond, func() { m.S.publish(t) })
	return nil
}
