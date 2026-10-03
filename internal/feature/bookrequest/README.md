# bookrequest

What readers ask for ("I am looking for this book"), matched against the copies sellers offer.

- **Frontend files:** `lib/features/book_request/data/sources/{book_request_fake_api,book_request_fake_store,book_request_demand}.dart`, `request_rules.dart` (`rules.go`, ported with its test).
- **Endpoints:** me `POST /requests`, `GET /requests/mine`, `POST /requests/close`, `GET /requests/wanted`; staff `GET /requests/demand`.
- **Table:** `book_requests` (migration `0009`).
- **Matching:** the same catalog book, or the same words in the title (either title may contain the other). `matchCount` counts live copies of other readers, `notifiedSellers` the sellers with a copy not sold yet; creating a request sends them a `bookWanted` notification.
- **Wanted:** other readers' open requests for books the reader is selling. **Demand:** open requests per title, most asked first.
- The fake API sends nothing when a listing is approved, and the app has no notification kind for it, so there is no approve hook.
