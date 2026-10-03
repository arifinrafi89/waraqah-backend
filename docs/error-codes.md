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
| `photo_invalid` | The profile photo, a return photo or a dispute photo is not a JPEG, PNG or WebP, or is over `MAX_IMAGE_MB`. | none (new in the backend) | `POST /profile/save`, `POST /orders/return`, `POST /sales/dispute` |
| `address_invalid` | The address breaks `AddressRules` (a blank label, recipient or line, a bad mobile, or no division, district and upazila). | `AddressRules.check` | `POST /addresses/save` |
| `address_unknown` | The address id is not one of the reader's. | `AddressFakeStore` (`save`, `delete`, `makeDefault`) | `POST /addresses/save`, `/addresses/default`, `/addresses/delete` |
| `notification_unknown` | The notification id is not one of the reader's. | `NotificationFakeStore.markRead` | `POST /notifications/read` |
| `booklist_not_yours` | The booklist id is not one of the reader own lists (unknown, or a staff list). | `BooklistFakeApi.saveMine`, `deleteMine` | `POST /booklists/mine/save`, `/booklists/mine/delete` |
| `booklist_invalid` | The booklist has no name or a blank one, repeats a book, or names a book the catalog does not have. | `BooklistFakeApi._saveMine` | `POST /booklists/mine/save` |
| `book_unknown` | The book id is not in the catalog. | none (new in the backend) | `POST /books/questions/ask`, `/books/questions/answer` |
| `question_invalid` | The question or answer is empty or longer than 500 characters. | none (new in the backend) | `POST /books/questions/ask`, `/books/questions/answer` |
| `book_invalid` | The Book draft breaks `CatalogAdminRules` (blank title, no editions, bad price, ISBN, class or exam), names an unknown Author, Category or Publisher, or uses an unknown enum name. | `CatalogAdminFakeStore.saveBook` | `POST /admin/catalog/books/save` |
| `record_invalid` | A Category, Author or Publisher breaks `CatalogAdminRules.record`, a Category has no Section, or its Section changes while Books use it. | `CatalogAdminFakeRecords.save` | `POST /admin/catalog/{categories,authors,publishers}/save` |
| `record_unknown` | The record id is not in the catalog. | `CatalogAdminFakeRecords` | `POST /admin/catalog/{categories,authors,publishers}/save`, `/delete` |
| `record_in_use` | A Book still uses the Category, Author or Publisher. | `CatalogAdminFakeRecords.delete` | `POST /admin/catalog/{categories,authors,publishers}/delete` |
| `banner_invalid` | The Banner breaks `CatalogAdminRules.banner`, or has an unknown target kind or Season. | `CatalogAdminFakeBanners.save` | `POST /admin/catalog/banners/save`, `/move` |
| `banner_unknown` | The Banner id is not one of Home's Banners. | `CatalogAdminFakeBanners` | `POST /admin/catalog/banners/save`, `/delete`, `/move` |
| `list_invalid` | The Collection or Booklist breaks `ListRules`, names an unknown Book or Expert, or a Booklist has no Staff kind. | `CatalogAdminFakeLists` | `POST /admin/catalog/collections/save`, `/booklists/save` |
| `list_unknown` | The Collection or Staff Booklist id does not exist (a Reader own list counts as unknown). | `CatalogAdminFakeLists` | `POST /admin/catalog/collections|booklists/save`, `/delete` |
| `stock_invalid` | Stock is negative, the Edition is an eBook, or the Edition is unknown. | `CatalogToolsFakeApi._setStock` | `POST /admin/catalog/editions/stock` |
| `season_unknown` | The Season name is not one of ramadan, boiMela, admission, backToSchool. | `CatalogAdminFakeApi.season` | `POST /admin/catalog/season/save` |
| `cart_item_unknown` | The item cannot go in the cart: unknown, not orderable (out of stock and not a pre-order), or a kind the cart does not sell (reader listings are bought with an offer). The answer is still the cart, with this code in the header. | `CartFakeStore.add` | `POST /cart/add`, `POST /orders/place` (a Certified Used copy sold meanwhile) |
| `wishlist_name_missing` | Sharing the wishlist needs the name friends see. | `WishlistFakeApi.share` | `POST /wishlist/share` |
| `alert_invalid` | The alert names an unknown Edition or kind. The answer is still the list of alerts, with this code in the header. | `AlertFakeStore.set` | `POST /alerts/set` |
| `order_unknown` | The order number is not one of the reader orders (or unknown, for Staff). | `OrderFakeStore.find` | `GET /orders/details`, `POST /orders/cancel`, `/orders/return`, `/orders/reorder`, `POST /admin/orders/advance`, `/admin/orders/return` |
| `order_not_cancellable` | The order has shipped, was delivered or is already cancelled. | `OrderFakeStore.cancel` | `POST /orders/cancel` |
| `return_not_allowed` | A return can only be asked for once, within 7 days of delivery, on printed books. | `OrderFakeStore.requestReturn` | `POST /orders/return` |
| `return_reason_invalid` | The return reason is not damaged, wrongBook or other. | `ReturnReason` | `POST /orders/return` |
| `order_step_unavailable` | The status asked for is not the order next step (someone moved it first, or it is delivered or cancelled). | `OrderFakeStore.advance` | `POST /admin/orders/advance` |
| `return_not_waiting` | The order has no return waiting for a decision. | `OrderFakeStore.decideReturn` | `POST /admin/orders/return` |
| `cart_empty` | The cart has nothing to order. | `CheckoutFakeApi.placeOrder` | `POST /orders/place` |
| `address_unknown` | The address id is not one of the reader (also used by profile addresses). | `CheckoutFakeApi.placeOrder` | `POST /orders/place` |
| `gift_name_missing` | A gift needs the name of the person it is for. | `CheckoutFakeApi.placeOrder` | `POST /orders/place` |
| `gift_message_too_long` | The gift card message is over 150 characters. | `Gift.maxMessageLength` | `POST /orders/place` |
| `payment_invalid` | The payment method is not bkash, nagad, cashOnDelivery or card. | `PaymentMethod` | `POST /orders/place`, `POST /donate/give` |
| `coupon_code_taken` | A coupon with that code already exists. | `CouponFakeStore.add` | `POST /admin/coupons/create` |
| `coupon_invalid` | The coupon breaks `CreateCoupon`: a code of 3 to 20 letters or digits, a percent from 1 to 90 or an amount of at least 1 taka, no negative minimum or cap, an end date in the future. | `CreateCoupon` | `POST /admin/coupons/create` |
| `donate_cod_not_allowed` | A donation is paid in advance; cash on delivery is not allowed. | `DonateFakeApi.give` | `POST /donate/give` |
| `donate_too_many` | The quantity is under 1 or more than the place still needs of that book. | `DonateFakeApi.give` | `POST /donate/give` |
| `donate_recipient_unknown` | The place, or the book it asked for, does not exist. | `DonateFakeApi.give` | `POST /donate/give` |
| `place_invalid` | The place breaks `PlaceRules` (name 3 to 80, a district, an area, a story of 10 to 300, at least one need of 1 to 100 copies) or names a book the catalog does not have. | `PlaceRules.check` | `POST /admin/donate/places/save` |
| `place_unknown` | The place id is not a verified place. | `DonatePlacesStore` | `POST /admin/donate/places/save`, `POST /admin/donate/places/remove` |
| `listing_unknown` | The listing does not exist or is not the reader own. | `P2pListingWriter.save` | `POST /p2p/listings/save` |
| `listing_not_editable` | Only a draft, a listing sent back or a rejected one can be changed. | `ListingRules.canEdit` | `POST /p2p/listings/save` |
| `listing_invalid` | The listing breaks `ListingRules`: a title of up to 120 characters, a note of up to 500, a price of up to 50,000 and, to send it for review, a price and front and back photos (and a damage photo when the damage flag is ticked). | `ListingRules.check` | `POST /p2p/listings/save` |
| `listing_photo_invalid` | A photo is not a JPEG, PNG or WebP image, or is too large. | `cloudinary.DecodeImage` | `POST /p2p/listings/save` |
| `reader_banned` | A banned reader cannot use the marketplace. | `ModerationFakeStore.isBanned` | `POST /p2p/listings/save` |
| `report_invalid` | The reason or note breaks `ReportRules`: "something else" needs a note, and a note is at most 500 characters. | `ReportRules.check` | `POST /reports` |
| `report_target_unknown` | The thing reported does not exist, or is the reader own. | `ReportFakeStore.report` | `POST /reports` |
| `block_invalid` | The reader to block is unknown or the reader themselves. | `ReportFakeStore.block` | `POST /blocks/add` |
| `listing_not_waiting` | The listing is not waiting for a decision. | `ModerationFakeStore.decide` | `POST /moderation/listings/decide` |
| `decision_invalid` | The decision is not approve, requestChanges or reject. | `ListingDecision` | `POST /moderation/listings/decide` |
| `decision_reason_invalid` | Asking for changes and rejecting need a reason of up to 300 characters. | `ModerationRules.checkDecision` | `POST /moderation/listings/decide` |
| `report_not_open` | The report does not exist or was already handled. | `ModerationFakeStore.act` | `POST /moderation/reports/act` |
| `report_no_owner` | There is nobody to warn or ban for that report. | `ModerationFakeStore.act` | `POST /moderation/reports/act` |
| `action_invalid` | The action is not remove, dismiss, warn or ban. | `ReportAction` | `POST /moderation/reports/act` |
| `thread_unknown` | The thread does not exist or the reader is not in it. | `InboxFakeStore.threads` | `GET /inbox/thread` (null), `POST /inbox/send`, `/inbox/offer/decide`, `/inbox/read`, `/inbox/listing/release`, `/inbox/listing/sold`, `/inbox/rate` |
| `blocked_reader` | One of the two readers blocked the other: no messages, offers or new deals either way. | `InboxFakeStore.isBlocked` | `POST /inbox/open`, `/inbox/send`, `/inbox/offer`, `/inbox/offer/decide` |
| `message_invalid` | A message is empty or over 1,000 characters. | `OfferRules.maxMessageLength` | `POST /inbox/send` |
| `offer_invalid` | The amount breaks `OfferRules` (at least 1 taka, at most the asking price, exactly the asking price when it is not negotiable) or the handover is not meetup or courier. | `OfferRules.check` | `POST /inbox/offer` |
| `offer_pending` | The thread already has an offer waiting for the seller. | `FakeThread.pendingOffer` | `POST /inbox/offer` |
| `offer_unknown` | The thread has no waiting offer with that id. | `InboxFakeSelling.decide` | `POST /inbox/offer/decide` |
| `listing_unavailable` | The listing is not on sale (live) any more. | `InboxFakeBuying.offer` | `POST /inbox/offer`, `POST /inbox/offer/decide` |
| `listing_own` | Sellers do not message or make offers on their own listing. | `InboxFakeBuying.openFor` | `POST /inbox/open`, `POST /inbox/offer` |
| `listing_closed` | A sold or unlisted book cannot start a new conversation. | `InboxFakeBuying.openFor` | `POST /inbox/open` |
| `deal_unknown` | The listing is not reserved for the buyer of this thread, or the reader is not its seller. | `InboxFakeSelling._dealThread` | `POST /inbox/listing/release`, `POST /inbox/listing/sold` |
| `rating_invalid` | Stars outside 1 to 5, or a comment over 300 characters. | `RatingRules.check` | `POST /inbox/rate` |
| `rating_not_allowed` | Rating is possible once each, after the sale to the buyer of the thread. | `InboxFakeRating.rate` | `POST /inbox/rate` |
| `request_invalid` | The book request breaks `RequestRules`: a title of 2 to 120 characters, a maximum price above 0, a note of up to 300 characters. | `RequestRules.check` | `POST /requests` |
| `request_unknown` | The request does not exist or is not the reader own. | `BookRequestFakeStore.close` | `POST /requests/close` |
| `sale_prepaid_only` | A handled sale needs money paid up front (bKash, Nagad or card); cash on delivery cannot be held by Waraqah. | `PaymentMethod.isPrepaid` in `HandledSaleFakeStore.buy` | `POST /sales/buy` |
| `sale_unknown` | The sale does not exist or the reader is neither its buyer nor its seller. | `HandledSaleFakeApi.sale` | `POST /sales/step`, `POST /sales/dispute` |
| `sale_step_refused` | Not the reader's move: only the seller sends a paid sale, only the buyer cancels it before that or confirms or disputes it once sent. | `HandledSaleFakeSteps.step`, `HandledSaleFakeMoney.dispute` | `POST /sales/step`, `POST /sales/dispute` |
| `dispute_invalid` | The reason is not a `DisputeReason`, the note is over 300 characters or there are more than 3 photos. | `SaleMath.maxDisputeNote`, `SaleMath.maxDisputePhotos` | `POST /sales/dispute` |
| `sale_not_disputed` | The sale is unknown or not waiting for a moderator. | `HandledSaleFakeMoney.settle` | `POST /sales/disputes/settle` |
| `payout_nothing` | Nothing earned is left to pay out. | `HandledSaleFakeMoney.payout` | `POST /sales/payout` |
| `sell_back_book_unknown` | The book is not in the catalog or has no printed Edition, so Waraqah does not buy it back. | `SellBackBooks.find` | `POST /sell-back` |
| `sell_back_invalid` | The condition is not a `BookCondition`, the flags are negative or the pickup address is under 5 characters. | `SellBackRules.minAddress` | `POST /sell-back`, `POST /sell-back/grade` |
| `sell_back_not_waiting` | The Sell Back is unknown or not picked up and waiting to be graded. | `SellBackFakeStore.grade` | `POST /sell-back/grade` |
