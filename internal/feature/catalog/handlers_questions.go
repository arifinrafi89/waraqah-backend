package catalog

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// BookQuestions is BookQuestionsFakeApi.questions.
func (h *Handler) BookQuestions(w http.ResponseWriter, r *http.Request) {
	list, err := h.S.Questions(r.Context(), httpx.Query(r, "id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}

// accountName is the display name of the signed-in reader.
func (h *Handler) accountName(r *http.Request, userID string) (string, error) {
	u, err := h.S.Store.DB.Q().GetUserByID(r.Context(), userID)
	if err != nil {
		return "", err
	}
	return u.Name, nil
}

type questionBody struct {
	BookID     string `json:"bookId"`
	QuestionID string `json:"questionId"`
	Text       string `json:"text"`
	// Name and IsStaff are accepted and ignored: the poster comes from the token (BACKEND_PLAN.md 4.4).
	Name    string `json:"name"`
	IsStaff bool   `json:"isStaff"`
}

// Ask is BookQuestionsFakeApi.ask; it answers the questions of the book.
func (h *Handler) Ask(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	var b questionBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	name, err := h.accountName(r, u.ID)
	if err == nil {
		err = h.S.Ask(r.Context(), snap, u.ID, name, b.BookID, b.Text)
	}
	h.questions(w, r, b.BookID, err)
}

// Answer is BookQuestionsFakeApi.answer; it answers the questions of the book.
func (h *Handler) Answer(w http.ResponseWriter, r *http.Request) {
	snap, ok := h.snapshot(w, r)
	if !ok {
		return
	}
	var b questionBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	name, err := h.accountName(r, u.ID)
	if err == nil {
		err = h.S.Answer(r.Context(), snap, u.ID, name, u.Role.IsStaff(), b.BookID, b.QuestionID, b.Text)
	}
	h.questions(w, r, b.BookID, err)
}

func (h *Handler) questions(w http.ResponseWriter, r *http.Request, bookID string, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	list, err := h.S.Questions(r.Context(), bookID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, list)
}
