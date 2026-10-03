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
| `profile_invalid` | The name is not 2 to 60 characters, or the phone is not a Bangladesh mobile number. | `ProfileRules.check` | `POST /profile/save` |
| `photo_invalid` | The profile photo is not a JPEG, PNG or WebP, or is over `MAX_IMAGE_MB`. | none (new in the backend) | `POST /profile/save` |
| `address_invalid` | The address breaks `AddressRules` (a blank label, recipient or line, a bad mobile, or no division, district and upazila). | `AddressRules.check` | `POST /addresses/save` |
| `address_unknown` | The address id is not one of the reader's. | `AddressFakeStore` (`save`, `delete`, `makeDefault`) | `POST /addresses/save`, `/addresses/default`, `/addresses/delete` |
| `notification_unknown` | The notification id is not one of the reader's. | `NotificationFakeStore.markRead` | `POST /notifications/read` |
| `booklist_not_yours` | The booklist id is not one of the reader own lists (unknown, or a staff list). | `BooklistFakeApi.saveMine`, `deleteMine` | `POST /booklists/mine/save`, `/booklists/mine/delete` |
| `booklist_invalid` | The booklist has no name or a blank one, repeats a book, or names a book the catalog does not have. | `BooklistFakeApi._saveMine` | `POST /booklists/mine/save` |
| `book_unknown` | The book id is not in the catalog. | none (new in the backend) | `POST /books/questions/ask`, `/books/questions/answer` |
| `question_invalid` | The question or answer is empty or longer than 500 characters. | none (new in the backend) | `POST /books/questions/ask`, `/books/questions/answer` |
