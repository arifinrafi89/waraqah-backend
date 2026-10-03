package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/logx"
)

// User is the signed-in person a handler sees. The role comes from the database, not the
// token, so a role change or a ban takes effect on the next request.
type User struct {
	ID   string
	Role Role
}

type userKey struct{}

// UserFrom returns the signed-in user of the request, if any.
func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey{}).(User)
	return u, ok
}

// WithUser puts a user in the context (for tests of handlers that need one).
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

// Middleware has one wrapper per auth level (BACKEND_PLAN.md section 5.4).
type Middleware struct {
	jwt   *JWT
	users UserSource
}

// NewMiddleware builds the auth levels from the token signer and the user source.
func NewMiddleware(j *JWT, users UserSource) *Middleware {
	return &Middleware{jwt: j, users: users}
}

type tokenState int

const (
	noToken tokenState = iota
	badToken
	goodToken
)

// identify reads the Authorization header. A token is good only when it parses and its
// user is still active (not deleted, not banned).
func (m *Middleware) identify(r *http.Request) (User, tokenState) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return User{}, noToken
	}
	scheme, token, found := strings.Cut(h, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return User{}, badToken
	}
	claims, err := m.jwt.Parse(strings.TrimSpace(token))
	if err != nil {
		return User{}, badToken
	}
	u, err := m.users.Lookup(r.Context(), claims.UserID)
	if err != nil {
		return User{}, badToken
	}
	return u, goodToken
}

func (m *Middleware) with(r *http.Request, u User) *http.Request {
	ctx := WithUser(r.Context(), u)
	logx.Annotate(ctx, "user_id", u.ID, "role", string(u.Role))
	return r.WithContext(ctx)
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	httpx.Error(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "sign in required")
}

// Public reads the token if there is one, so per-viewer fields are right, and treats a
// missing or bad token as a guest. It never answers 401.
func (m *Middleware) Public(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, st := m.identify(r); st == goodToken {
			r = m.with(r, u)
		}
		next.ServeHTTP(w, r)
	})
}

// Me is for user-scoped endpoints. A GET without a token reaches the handler with no user
// (the handler answers the empty value). Everything else, and any bad token, is a 401.
func (m *Middleware) Me(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, st := m.identify(r)
		switch {
		case st == goodToken:
			r = m.with(r, u)
		case st == noToken && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		default:
			unauthorized(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Authed always needs a good token. Use it for live (SSE) streams.
func (m *Middleware) Authed(next http.Handler) http.Handler {
	return m.require(func(User) bool { return true }, next)
}

// Staff needs any staff role: no token is 401, a reader is 403.
func (m *Middleware) Staff(next http.Handler) http.Handler {
	return m.require(func(u User) bool { return u.Role.IsStaff() }, next)
}

// StaffCan needs a staff role that holds the permission (super admin holds all).
func (m *Middleware) StaffCan(p Perm, next http.Handler) http.Handler {
	return m.require(func(u User) bool { return u.Role.Can(p) }, next)
}

func (m *Middleware) require(allowed func(User) bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, st := m.identify(r)
		if st != goodToken {
			unauthorized(w, r)
			return
		}
		if !allowed(u) {
			httpx.Error(w, r, http.StatusForbidden, httpx.CodeForbidden, "not allowed for your role")
			return
		}
		next.ServeHTTP(w, m.with(r, u))
	})
}
