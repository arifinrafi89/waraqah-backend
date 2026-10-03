package auth

import (
	"errors"
	"net/http"

	pauth "github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Handler is the HTTP side of the auth endpoints.
type Handler struct{ S *Service }

// NewHandler builds the handler around a service.
func NewHandler(s *Service) *Handler { return &Handler{S: s} }

// fail answers a service error: a Refusal as 200 null with its code, anything else as a 500.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ref Refusal
	if errors.As(err, &ref) {
		httpx.Refuse(w, r, string(ref))
		return
	}
	h.S.Log.Error("auth endpoint failed", "error", err)
	httpx.Error(w, r, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong")
}

// Login is AuthFakeApi.login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var b loginBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	res, err := h.S.Login(r.Context(), b.Email, b.Password)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, res)
}

// Google is AuthFakeApi.google. Today's app sends no body, which counts as a missing token.
func (h *Handler) Google(w http.ResponseWriter, r *http.Request) {
	var b googleBody
	if r.ContentLength != 0 && !httpx.Decode(w, r, &b) {
		return
	}
	res, err := h.S.LoginGoogle(r.Context(), b.IDToken)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, res)
}

// RequestSignUpOtp is AuthFakeApi.requestSignUpOtp.
func (h *Handler) RequestSignUpOtp(w http.ResponseWriter, r *http.Request) {
	var b signupBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	if err := h.S.RequestSignUpOTP(r.Context(), b.Name, b.Contact, b.Password); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, ok{true})
}

// VerifySignUpOtp is AuthFakeApi.verifySignUpOtp.
func (h *Handler) VerifySignUpOtp(w http.ResponseWriter, r *http.Request) {
	var b verifyBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	res, err := h.S.VerifySignUp(r.Context(), b.Contact, b.OTP)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, res)
}

// RequestPasswordReset is AuthFakeApi.requestPasswordReset. It always answers ok.
func (h *Handler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var b contactBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	if err := h.S.RequestPasswordReset(r.Context(), b.Contact); err != nil {
		h.S.Log.Error("password reset request failed", "error", err) // never reveal it to the caller
	}
	httpx.JSON(w, ok{true})
}

// ResetPassword is AuthFakeApi.resetPassword.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var b resetBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	if err := h.S.ResetPassword(r.Context(), b.Contact, b.OTP, b.Password); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, ok{true})
}

// Refresh is the additive POST /auth/refresh (BACKEND_PLAN.md section 4.6). A bad token is a 401.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var b refreshBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	res, err := h.S.Rotate(r.Context(), b.RefreshToken)
	if errors.Is(err, pauth.ErrInvalidRefresh) {
		httpx.Error(w, r, http.StatusUnauthorized, httpx.CodeUnauthorized, "sign in again")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, res)
}

// Logout is the additive POST /auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var b refreshBody
	if !httpx.Decode(w, r, &b) {
		return
	}
	if err := h.S.Logout(r.Context(), b.RefreshToken); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, ok{true})
}
