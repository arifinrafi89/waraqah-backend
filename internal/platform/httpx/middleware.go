package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/logx"
)

// Middleware wraps a handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so the first one listed is the outermost.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// statusRecorder remembers the status code and still supports flushing (SSE).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the real writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// Recover turns a panic into a 500 error shape and logs the stack.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.Error("panic", "panic", v, "stack", string(debug.Stack()))
					Error(w, r, http.StatusInternalServerError, CodeInternal, "something went wrong")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID keeps the client's X-Request-ID or makes one, echoes it, and
// puts a request-scoped logger in the context.
func RequestID(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" || len(id) > 64 {
				var b [8]byte
				_, _ = rand.Read(b[:])
				id = "req_" + hex.EncodeToString(b[:])
			}
			w.Header().Set("X-Request-ID", id)
			ctx := logx.WithRequest(r.Context(), log, "request_id", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AccessLog writes one line per request: method, route, status, latency, refusal.
// It never logs bodies, headers or query strings.
func AccessLog() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}
			attrs := []any{"method", r.Method, "route", route, "status", status,
				"latency_ms", time.Since(start).Milliseconds()}
			if code := rec.Header().Get(ErrorHeader); code != "" {
				attrs = append(attrs, "refusal", code)
			}
			logx.From(r.Context()).Info("request", attrs...)
		})
	}
}

// CORS allows only the listed origins.
func CORS(allowed []string) Middleware {
	ok := map[string]bool{}
	for _, o := range allowed {
		ok[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && ok[origin] {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
				h.Set("Access-Control-Expose-Headers", "X-Waraqah-Error, X-Request-ID")
				if r.Method == http.MethodOptions {
					h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
					h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
					h.Set("Access-Control-Max-Age", "600")
				}
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimit answers 413 when a request body is over maxBytes.
func BodyLimit(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				Error(w, r, http.StatusRequestEntityTooLarge, CodeTooLarge, "request body is too large")
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Router registers routes under the API prefix. Patterns are "METHOD /path".
type Router struct {
	Mux    *http.ServeMux
	Prefix string
}

// Handle registers h for "METHOD /path", serving it at METHOD <prefix>/path.
func (r Router) Handle(pattern string, h http.Handler) {
	method, path, found := strings.Cut(pattern, " ")
	if !found {
		panic("httpx: route pattern must be METHOD /path: " + pattern)
	}
	r.Mux.Handle(method+" "+r.Prefix+path, h)
}

// HandleFunc is Handle for a function.
func (r Router) HandleFunc(pattern string, f http.HandlerFunc) { r.Handle(pattern, f) }
