# auth

Sign-in, Google sign-in, e-mail sign-up and password reset with one-time codes, token refresh and logout.

- **Frontend files:** `lib/features/auth/data/sources/{auth_fake_api,auth_fixtures,auth_remote_source}.dart`, model `app_user_model.dart`.
- **Endpoints:** `POST /auth/login`, `/auth/google`, `/auth/signup/request-otp`, `/auth/signup/verify-otp`, `/auth/password/request-otp`, `/auth/password/reset`, plus the additive `/auth/refresh` and `/auth/logout`. `POST /auth/delete` belongs to `profile`.
- **Tables:** `users`, `pending_signups`, `otp_codes`, `refresh_tokens` (migration `0001`). Queries: `db/queries/accounts.sql`.
- **Platform pieces it uses:** `internal/platform/auth` (tokens, OTP, Google verifier), `email`, `ratelimit`.
- **Refusals:** `wrong_credentials`, `wrong_otp`, `phone_not_supported`, `contact_invalid`, `email_taken`, `password_invalid`, `google_token_missing`, `google_token_invalid` (see `docs/error-codes.md`).
- Every sign-in answers `{id, name, email, role}` plus `accessToken`, `refreshToken`, `expiresAt`. All `/auth/*` routes are rate limited per IP.
