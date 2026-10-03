# sellback

Sell Back and Certified Used: a reader sells a used book to Waraqah for an instant quote, a courier picks it up, staff grade it, Waraqah pays into the reader's wallet and puts the copy on sale as Certified Used.

- **Frontend files:** `lib/features/sell_back/data/sources/{sell_back_fake_api,sell_back_fake_store,sell_back_seed,sell_back_books,certified_used_stock}.dart`; rules `sell_back_rules.dart` (`rules.go`) and `finished_it_offers.dart` (`finished.go`), ported with their tests.
- **Endpoints (me):** `GET /sell-back/books|book|mine`, `POST /sell-back`. **Staff (catalog):** `GET /sell-back/queue`, `POST /sell-back/grade`.
- **Tables:** `sell_backs` (ids `SB-<n>` from `sell_back_seq`; the book is kept as quoted), `certified_used` (ids `cu-<book>-<n>`, `sold_at` once an order takes it) (migration `0010`).
- **Statuses:** `scheduled` → `pickedUp` (courier job) → `paid` or `returned` (staff grade), the Dart `SellBackStatus` names.
- **Courier job** (`COURIER_PICKUP_DELAY`, always on): every pickup booked at least the delay ago is collected; idempotent and catches up after a sleep.
- **Certified Used:** implements `catalog.UsedStock` (the cheapest copy on `/books/used-options`), `cart.UsedStock` (the `certifiedUsed` cart kind) and `checkout.UsedCopies` (an order takes the copy off sale).
