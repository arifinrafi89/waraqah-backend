# Error codes

Every refusal answers `200` with body `null` and an `X-Waraqah-Error: <code>` header (BACKEND_PLAN.md §4.2, §16).
Each code lives in `internal/platform/httpx/errors.go`. Add a row here in the same PR that adds the code.

| Code | Meaning | Dart rule mirrored | Endpoints |
|---|---|---|---|
| `rate_limited` | Too many requests (sent as a 429 error body, not a refusal header). | none (new in the backend) | `/auth/*`, `/assistant/ask`, `/reports` |
| `wrong_credentials` | Wrong email or password, a banned account, or a Google account that is banned. | `AuthFakeApi.login` (accountFor) | `POST /auth/login`, `POST /auth/google` |
| `wrong_otp` | The one-time code is wrong, expired, used up, or has no pending sign-up or account behind it. | `AuthFakeApi.verifySignUpOtp`, `resetPassword` (`AuthFailure.wrongCode`) | `POST /auth/signup/verify-otp`, `POST /auth/password/reset` |
| `phone_not_supported` | Sign-up contact is a phone number; SMS costs money, so only email is supported (F8). | none (new in the backend) | `POST /auth/signup/request-otp` |
| `contact_invalid` | Sign-up contact is not a valid email address. | `SignIn` email pattern | `POST /auth/signup/request-otp` |
| `email_taken` | An account with that email already exists. | none (new in the backend) | `POST /auth/signup/request-otp`, `POST /auth/signup/verify-otp` |
| `password_invalid` | Password is empty or longer than 72 bytes (bcrypt limit). | `SignIn` (`missingPassword`) | `POST /auth/signup/request-otp`, `POST /auth/password/reset` |
| `google_token_missing` | The request carries no Google ID token (today's app sends no body until F3). | `AuthFakeApi.google` | `POST /auth/google` |
| `google_token_invalid` | The Google ID token failed the signature, issuer, audience, expiry or verified-email check. | none (new in the backend) | `POST /auth/google` |
