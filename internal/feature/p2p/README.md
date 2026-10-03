# p2p (used marketplace)

Listings readers sell, their seller pages, and the status changes the inbox and handled sales make.

- **Frontend files:** `lib/features/p2p/data/sources/{p2p_fake_api,p2p_fake_store,p2p_listing_writer,p2p_catalog_link,p2p_people,p2p_ratings,p2p_seller_json}.dart`; rules `listing_rules.dart` (`rules.go`) and `fair_price.dart` (`fair_price.go`), ported with their tests.
- **Endpoints:** public `GET /p2p/listings` (`available`, `limit`), `/p2p/listing`, `/p2p/listings/for-book`, `/p2p/seller`; me `GET /p2p/listings/mine`, `POST /p2p/listings/save`.
- **Tables:** `listings` (newest first by `position`; `buyer_id` is who it is reserved for or sold to), `listing_photos` (one row per slot with the Cloudinary `public_id`), `ratings`, and `users.sold_before` for the books a reader sold before the records of the app (migration `0008`).
- **Save:** a new listing or an edit of the own one that is a draft, sent back or rejected (`listing_not_editable` otherwise). A draft needs only a title; sending needs a price and front and back photos. New photos (`photoData`, base64) are checked as images, uploaded to Cloudinary, and the assets of removed slots are deleted. `photos` in the answer lists the slots kept (contract v1). `isMine` and `isMyDeal` come from the token.
- **Lists:** on sale or reserved, newest first; listings of deleted accounts and of readers the viewer blocked, or who blocked the viewer, are left out.
- **Interfaces:** `Status` (`Find`, `SetStatus`: reserve, sell, release, for the inbox and handled sales), `Bans` (implemented by moderation), `Moderate`/`TakeDown`/`Queue` for the Moderation Center, `OnAccountDeleted` (a profile delete hook).
