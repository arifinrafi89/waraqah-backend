package auth

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Routes registers the auth endpoints. limit is the per-IP rate limit for /auth/*;
// the Public wrapper reads an optional token (none of these need one).
func (h *Handler) Routes(r httpx.Router, limit httpx.Middleware) {
	r.Handle("POST /auth/login", limit(http.HandlerFunc(h.Login)))                               // AuthFakeApi.login
	r.Handle("POST /auth/google", limit(http.HandlerFunc(h.Google)))                             // AuthFakeApi.google
	r.Handle("POST /auth/signup/request-otp", limit(http.HandlerFunc(h.RequestSignUpOtp)))       // AuthFakeApi.requestSignUpOtp
	r.Handle("POST /auth/signup/verify-otp", limit(http.HandlerFunc(h.VerifySignUpOtp)))         // AuthFakeApi.verifySignUpOtp
	r.Handle("POST /auth/password/request-otp", limit(http.HandlerFunc(h.RequestPasswordReset))) // AuthFakeApi.requestPasswordReset
	r.Handle("POST /auth/password/reset", limit(http.HandlerFunc(h.ResetPassword)))              // AuthFakeApi.resetPassword
	r.Handle("POST /auth/refresh", limit(http.HandlerFunc(h.Refresh)))                           // additive (BACKEND_PLAN.md 4.6)
	r.Handle("POST /auth/logout", limit(http.HandlerFunc(h.Logout)))                             // additive (BACKEND_PLAN.md 4.6)
}
