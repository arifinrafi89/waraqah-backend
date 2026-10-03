package bites

import (
	"errors"
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the Bites endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("bites endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
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

func viewer(r *http.Request) string {
	u, _ := auth.UserFrom(r.Context())
	return u.ID
}

// Feed is BiteFakeApi.feed: `?feed=forYou|following&bookId=&authorId=`, newest first, at most 30.
func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	q := Query{Following: httpx.Query(r, "feed") == "following", BookID: httpx.Query(r, "bookId"), AuthorID: httpx.Query(r, "authorId")}
	v, err := h.S.Feed(r.Context(), viewer(r), q)
	answer(h, w, r, &v, err)
}

// Detail is BiteFakeApi.detail: `?id=`, one Bite with its comments, or null.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	v, err := h.S.Detail(r.Context(), viewer(r), httpx.Query(r, "id"))
	answer(h, w, r, v, err)
}

// Post is BiteFakeApi.post: body `{text, bookId?, spoiler}`.
func (h *Handler) Post(w http.ResponseWriter, r *http.Request) {
	var in Draft
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Post(r.Context(), viewer(r), in)
	answer(h, w, r, v, err)
}

// Edit is BiteFakeApi.edit: body `{id, text, bookId?, spoiler}`, own only.
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	var in Draft
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Edit(r.Context(), viewer(r), in)
	answer(h, w, r, v, err)
}

// Delete is BiteFakeApi.delete: body `{id}`, own only; answers `{id}`.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	var in struct{ ID string }
	if !httpx.Decode(w, r, &in) {
		return
	}
	err := h.S.Delete(r.Context(), viewer(r), in.ID)
	answer(h, w, r, &map[string]string{"id": in.ID}, err)
}

// Like is BiteFakeApi.like: body `{id, liked}`.
func (h *Handler) Like(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID    string `json:"id"`
		Liked bool   `json:"liked"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Like(r.Context(), viewer(r), in.ID, in.Liked)
	answer(h, w, r, v, err)
}

// CommentPost is BiteFakeApi.comment: body `{biteId, text, parentId?}`; answers the detail.
func (h *Handler) CommentPost(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BiteID   string  `json:"biteId"`
		Text     string  `json:"text"`
		ParentID *string `json:"parentId"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.Comment(r.Context(), viewer(r), in.BiteID, in.Text, in.ParentID)
	answer(h, w, r, v, err)
}

// DeleteComment is BiteFakeApi.deleteComment: body `{id}`, own only; answers the detail.
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	var in struct{ ID string }
	if !httpx.Decode(w, r, &in) {
		return
	}
	v, err := h.S.DeleteComment(r.Context(), viewer(r), in.ID)
	answer(h, w, r, v, err)
}

// Routes registers the Bites endpoints.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	public := func(f http.HandlerFunc) http.Handler { return a.Public(f) }
	me := func(f http.HandlerFunc) http.Handler { return a.Me(f) }
	r.Handle("GET /bites", public(h.Feed))                       // BiteFakeApi.feed
	r.Handle("GET /bites/detail", public(h.Detail))              // BiteFakeApi.detail
	r.Handle("POST /bites/post", me(h.Post))                     // BiteFakeApi.post
	r.Handle("POST /bites/edit", me(h.Edit))                     // BiteFakeApi.edit
	r.Handle("POST /bites/delete", me(h.Delete))                 // BiteFakeApi.delete
	r.Handle("POST /bites/like", me(h.Like))                     // BiteFakeApi.like
	r.Handle("POST /bites/comments/post", me(h.CommentPost))     // BiteFakeApi.comment
	r.Handle("POST /bites/comments/delete", me(h.DeleteComment)) // BiteFakeApi.deleteComment
}
