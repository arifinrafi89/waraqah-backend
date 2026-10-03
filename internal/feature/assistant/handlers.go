package assistant

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the assistant endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

// Greeting is AssistantFakeApi.greeting: `?lang=bn`, the opening line.
func (h *Handler) Greeting(w http.ResponseWriter, r *http.Request) {
	lang := httpx.Query(r, "lang")
	if lang == "" {
		lang = "en"
	}
	httpx.JSON(w, h.S.Greeting(lang))
}

// Ask is AssistantFakeApi.ask: body `{prompt, history: [text], lang}`.
func (h *Handler) Ask(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Prompt  string   `json:"prompt"`
		History []string `json:"history"`
		Lang    string   `json:"lang"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.Lang == "" {
		in.Lang = "en"
	}
	v, err := h.S.Ask(r.Context(), in.Prompt, in.History, in.Lang)
	if err != nil {
		h.S.Log.Error("assistant failed", "error", err)
		httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
		return
	}
	httpx.JSON(w, v)
}

// Routes registers the assistant endpoints. ask is rate-limited per reader (AI_RATE_PER_MIN).
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware, perUser httpx.Middleware) {
	r.Handle("GET /assistant/greeting", a.Public(http.HandlerFunc(h.Greeting))) // AssistantFakeApi.greeting
	r.Handle("POST /assistant/ask", a.Me(perUser(http.HandlerFunc(h.Ask))))     // AssistantFakeApi.ask
}
