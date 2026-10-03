// Package ratelimit is an in-memory token bucket keyed by IP or user (BACKEND_PLAN.md section 17).
package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter allows `limit` events per `per` for each key, with a burst of `limit`.
type Limiter struct {
	limit float64
	per   time.Duration
	now   func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucket
	swept   time.Time
}

// New makes a limiter: limit events per window. A limit of 0 or less allows everything.
func New(limit int, per time.Duration) *Limiter {
	return &Limiter{limit: float64(limit), per: per, now: time.Now, buckets: map[string]*bucket{}}
}

// WithClock sets the time source (tests).
func (l *Limiter) WithClock(now func() time.Time) *Limiter { l.now = now; return l }

// Allow takes one token for key and reports whether the event may go ahead.
func (l *Limiter) Allow(key string) bool {
	if l.limit <= 0 {
		return true
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.limit, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.limit / l.per.Seconds()
	if b.tokens > l.limit {
		b.tokens = l.limit
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweep drops buckets that have been idle long enough to be full again, so memory stays small.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.swept) < l.per {
		return
	}
	l.swept = now
	for k, b := range l.buckets {
		if now.Sub(b.last) > l.per {
			delete(l.buckets, k)
		}
	}
}

// IPKey is the client address. Behind a proxy it is the last X-Forwarded-For entry, which is
// the one the proxy itself saw (earlier ones are client supplied).
func IPKey(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
			return "ip:" + ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}

// UserKey is the signed-in user id, or the client address for a guest.
func UserKey(r *http.Request) string {
	if u, ok := auth.UserFrom(r.Context()); ok {
		return "user:" + u.ID
	}
	return IPKey(r)
}

// Middleware answers 429 in the error shape when key(r) is over the limit.
func (l *Limiter) Middleware(key func(*http.Request) string) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow(key(r)) {
				w.Header().Set("Retry-After", "60")
				httpx.Error(w, r, http.StatusTooManyRequests, httpx.ErrRateLimited, "too many requests, slow down")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ByIP limits per client address (the /auth/* routes).
func (l *Limiter) ByIP() httpx.Middleware { return l.Middleware(IPKey) }

// ByUser limits per signed-in user (assistant, reports). It must run inside the auth middleware.
func (l *Limiter) ByUser() httpx.Middleware { return l.Middleware(UserKey) }
