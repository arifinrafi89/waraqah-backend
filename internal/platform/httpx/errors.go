package httpx

// Refusal codes: sent in the X-Waraqah-Error header of a `200 null` answer.
// Features add their codes here and in docs/error-codes.md (AGENTS.md rule 10).
const (
	// ErrRateLimited is sent as a 429 error body code, not as a refusal header.
	ErrRateLimited = "rate_limited"

	// auth
	ErrWrongCredentials   = "wrong_credentials"
	ErrWrongOTP           = "wrong_otp"
	ErrPhoneNotSupported  = "phone_not_supported"
	ErrContactInvalid     = "contact_invalid"
	ErrEmailTaken         = "email_taken"
	ErrPasswordInvalid    = "password_invalid"
	ErrGoogleTokenMissing = "google_token_missing"
	ErrGoogleTokenInvalid = "google_token_invalid"

	// profile
	ErrProfileInvalid = "profile_invalid"
	ErrPhotoInvalid   = "photo_invalid"
	ErrAddressInvalid = "address_invalid"
	ErrAddressUnknown = "address_unknown"

	// catalog
	ErrBooklistNotYours = "booklist_not_yours"
	ErrBooklistInvalid  = "booklist_invalid"
	ErrBookUnknown      = "book_unknown"
	ErrQuestionInvalid  = "question_invalid"

	// catalog admin
	ErrBookInvalid   = "book_invalid"
	ErrRecordInvalid = "record_invalid"
	ErrRecordUnknown = "record_unknown"
	ErrRecordInUse   = "record_in_use"
	ErrBannerInvalid = "banner_invalid"
	ErrBannerUnknown = "banner_unknown"
	ErrListInvalid   = "list_invalid"
	ErrListUnknown   = "list_unknown"
	ErrStockInvalid  = "stock_invalid"
	ErrSeasonUnknown = "season_unknown"

	// cart, wishlist, alerts
	ErrCartItemUnknown     = "cart_item_unknown"
	ErrWishlistNameMissing = "wishlist_name_missing"
	ErrAlertInvalid        = "alert_invalid"

	// notifications
	ErrNotificationUnknown = "notification_unknown"
)

// Error body codes for non-200 answers (BACKEND_PLAN.md §4.2).
const (
	CodeUnauthorized = "unauthorized"
	CodeForbidden    = "forbidden"
	CodeNotFound     = "not_found"
	CodeBadRequest   = "bad_request"
	CodeTooLarge     = "too_large"
	CodeInternal     = "internal"
)
