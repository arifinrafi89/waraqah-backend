package catalog

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Categories is BookFakeApi.categories: the categories of a section.
func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	section := httpx.Query(r, "section")
	out := []Category{}
	for _, c := range snap.Categories {
		if section != "" && c.Section == section {
			out = append(out, c)
		}
	}
	httpx.JSON(w, out)
}

// Subjects is BookFakeApi.subjects: every subject, or those with a visible book in the section.
func (h *Handler) Subjects(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	section := httpx.Query(r, "section")
	used := map[string]bool{}
	for _, b := range snap.Books {
		if !b.Hidden && b.Section == section && b.SubjectID != nil {
			used[*b.SubjectID] = true
		}
	}
	out := []Subject{}
	for _, s := range snap.Subjects {
		if section == "" || used[s.ID] {
			out = append(out, s)
		}
	}
	httpx.JSON(w, out)
}

// Author is BookFakeApi.author.
func (h *Handler) Author(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	for _, a := range snap.Authors {
		if a.ID == httpx.Query(r, "id") {
			httpx.JSON(w, a)
			return
		}
	}
	httpx.Null(w)
}

// Publisher is BookFakeApi.publisher.
func (h *Handler) Publisher(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	for _, p := range snap.Publishers {
		if p.ID == httpx.Query(r, "id") {
			httpx.JSON(w, p)
			return
		}
	}
	httpx.Null(w)
}

// Collections is CollectionFakeApi.collections.
func (h *Handler) Collections(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	f := CollectionFilter{Section: httpx.Query(r, "section"), Expert: httpx.Query(r, "expert")}
	switch httpx.Query(r, "hasExpert") {
	case "true":
		t := true
		f.HasExpert = &t
	case "false":
		t := false
		f.HasExpert = &t
	}
	list, err := h.S.Collections(r.Context(), snap, f)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// CollectionDetail is CollectionFakeApi.detail.
func (h *Handler) CollectionDetail(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	c, err := h.S.CollectionByID(r.Context(), snap, httpx.Query(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	answer(w, c)
}

// Experts is CollectionFakeApi.experts.
func (h *Handler) Experts(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.Experts(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// ExpertDetail is CollectionFakeApi.expert.
func (h *Handler) ExpertDetail(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	e, err := h.S.ExpertDetail(r.Context(), snap, httpx.Query(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	answer(w, e)
}

func viewerID(r *http.Request) string {
	if u, ok := auth.UserFrom(r.Context()); ok {
		return u.ID
	}
	return ""
}

// Booklists is BooklistFakeApi.booklists: every staff booklist and the viewer own lists.
func (h *Handler) Booklists(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	list, err := h.S.Booklists(r.Context(), snap, viewerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// BooklistDetail is BooklistFakeApi.detail. Another reader own list is not visible.
func (h *Handler) BooklistDetail(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	list, err := h.S.Booklists(r.Context(), snap, viewerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for i := range list {
		if list[i].ID == httpx.Query(r, "id") {
			httpx.JSON(w, list[i])
			return
		}
	}
	httpx.Null(w)
}

// SaveMine is BooklistFakeApi.saveMine.
func (h *Handler) SaveMine(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	var in SaveMineInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	b, err := h.S.SaveMine(r.Context(), snap, viewerID(r), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, b)
}

// DeleteMine is BooklistFakeApi.deleteMine.
func (h *Handler) DeleteMine(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID string `json:"id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := h.S.DeleteMine(r.Context(), viewerID(r), in.ID); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, map[string]string{"id": in.ID})
}
