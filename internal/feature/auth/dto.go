package auth

// AppUser is AppUserModel in the app: the fields every auth answer carries.
type AppUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Session is what sign-in answers: the user plus the tokens (BACKEND_PLAN.md section 5.2).
// The token fields are additive; the app stores them after frontend task F2.
type Session struct {
	AppUser
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
}

// Tokens is what /auth/refresh answers.
type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
}

type ok struct {
	OK bool `json:"ok"`
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type googleBody struct {
	IDToken string `json:"idToken"`
}

type signupBody struct {
	Name     string `json:"name"`
	Contact  string `json:"contact"`
	Password string `json:"password"`
}

type verifyBody struct {
	Contact string `json:"contact"`
	OTP     string `json:"otp"`
}

type contactBody struct {
	Contact string `json:"contact"`
}

type resetBody struct {
	Contact  string `json:"contact"`
	OTP      string `json:"otp"`
	Password string `json:"password"`
}

type refreshBody struct {
	RefreshToken string `json:"refreshToken"`
}
