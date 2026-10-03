# wishlist

The books a reader saved (newest first, saving twice keeps one copy moved to the top) and the share link friends open without an account.

- **Frontend file:** `lib/features/wishlist/data/sources/wishlist_fake_api.dart`.
- **Endpoints:** `GET /wishlist`, `POST /wishlist/save|remove|share`, `GET /wishlist/shared` (public).
- **Tables:** `wishlist_items`, `wishlist_shares` (migration `0005`). `share` makes or reuses the link **of the signed-in reader**; `ownerName` is only the name friends see. The shared list is live: it shows the owner books as they are now.
