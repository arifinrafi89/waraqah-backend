# Contract changes

Every change to the v1 API contract is listed here, newest first (BACKEND_PLAN.md §4.7).

- **v1.2** (book covers): every Book gains `coverUrl` (`String?`), a picture of the book from Open Library (https://openlibrary.org, used with credit in a non-commercial project); `null` keeps the generated cover. Listing answers gain `coverUrl` too: the cover of the listing's catalog book, or of the same title, when the seller has no photo. Both are additive, so the app at v1 is unaffected.
- **v1.1** (T19, backend half of F7): listing answers (`/p2p/listings`, `/p2p/listing`, `/p2p/listings/for-book`, `/p2p/listings/mine`, `/p2p/listings/save` and `/p2p/seller`) and the moderation queue (`/moderation/listings`) gain `photoUrls: {slot: url}`, the Cloudinary thumbnail (`c_fill,w_300,q_auto,f_auto`) of every slot that has an uploaded photo. `photos` keeps the slot names, so the app at v1 is unaffected (the field is additive); the frontend half of F7 shows the real photos.
- **v1** = the fake API at frontend PR #144 (`d286a04`). Re-exported against frontend PR #145 (`71906e4`, the app's switch to the real API): no shape changed, only dates, the Ayah of the day and generated ids.
