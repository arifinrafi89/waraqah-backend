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

	// orders, checkout
	ErrOrderUnknown         = "order_unknown"
	ErrOrderNotCancellable  = "order_not_cancellable"
	ErrReturnNotAllowed     = "return_not_allowed"
	ErrReturnReasonInvalid  = "return_reason_invalid"
	ErrOrderStepUnavailable = "order_step_unavailable"
	ErrReturnNotWaiting     = "return_not_waiting"
	ErrCartEmpty            = "cart_empty"
	ErrAddressUnknownOrder  = "address_unknown"
	ErrGiftNameMissing      = "gift_name_missing"
	ErrGiftMessageTooLong   = "gift_message_too_long"
	ErrPaymentInvalid       = "payment_invalid"
	ErrCouponCodeTaken      = "coupon_code_taken"
	ErrCouponInvalid        = "coupon_invalid"
	ErrDonateCOD            = "donate_cod_not_allowed"
	ErrDonateTooMany        = "donate_too_many"
	ErrDonateUnknown        = "donate_recipient_unknown"
	ErrPlaceInvalid         = "place_invalid"
	ErrPlaceUnknown         = "place_unknown"
	ErrListingUnknown       = "listing_unknown"
	ErrListingNotEditable   = "listing_not_editable"
	ErrListingInvalid       = "listing_invalid"
	ErrListingPhotoInvalid  = "listing_photo_invalid"
	ErrReaderBanned         = "reader_banned"
	ErrReportInvalid        = "report_invalid"
	ErrReportTargetUnknown  = "report_target_unknown"
	ErrBlockInvalid         = "block_invalid"
	ErrListingNotWaiting    = "listing_not_waiting"
	ErrDecisionInvalid      = "decision_invalid"
	ErrDecisionReason       = "decision_reason_invalid"
	ErrReportNotOpen        = "report_not_open"
	ErrReportNoOwner        = "report_no_owner"
	ErrActionInvalid        = "action_invalid"
	ErrThreadUnknown        = "thread_unknown"
	ErrBlockedReader        = "blocked_reader"
	ErrMessageInvalid       = "message_invalid"
	ErrOfferInvalid         = "offer_invalid"
	ErrOfferPending         = "offer_pending"
	ErrOfferUnknown         = "offer_unknown"
	ErrListingUnavailable   = "listing_unavailable"
	ErrListingOwn           = "listing_own"
	ErrListingClosed        = "listing_closed"
	ErrDealUnknown          = "deal_unknown"
	ErrRatingInvalid        = "rating_invalid"
	ErrRatingNotAllowed     = "rating_not_allowed"
	ErrRequestInvalid       = "request_invalid"
	ErrRequestUnknown       = "request_unknown"

	// handled sales
	ErrSalePrepaidOnly = "sale_prepaid_only"
	ErrSaleUnknown     = "sale_unknown"
	ErrSaleStepRefused = "sale_step_refused"
	ErrDisputeInvalid  = "dispute_invalid"
	ErrSaleNotDisputed = "sale_not_disputed"
	ErrPayoutNothing   = "payout_nothing"

	// sell back
	ErrSellBackBookUnknown = "sell_back_book_unknown"
	ErrSellBackInvalid     = "sell_back_invalid"
	ErrSellBackNotWaiting  = "sell_back_not_waiting"

	// bites
	ErrBiteInvalid     = "bite_invalid"
	ErrBiteUnknown     = "bite_unknown"
	ErrBiteNotYours    = "bite_not_yours"
	ErrCommentInvalid  = "comment_invalid"
	ErrCommentUnknown  = "comment_unknown"
	ErrCommentNotYours = "comment_not_yours"

	// reviews
	ErrReviewInvalid = "review_invalid"
	ErrReviewUnknown = "review_unknown"

	// readers
	ErrReaderUnknown = "reader_unknown"
	ErrFollowRefused = "follow_refused"

	// shelves
	ErrProgressInvalid = "progress_invalid"
	ErrNotOnShelf      = "not_on_shelf"
	ErrGoalInvalid     = "goal_invalid"

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
