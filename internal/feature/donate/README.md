# donate

Verified places that take donated books, and the orders donors place for them.

- **Frontend files:** `lib/features/donate/data/sources/{donate_fake_api,donate_admin_fake_api,donate_places_store,donate_fixtures,donation_order}.dart`, rules `recipient.dart` and `donate_place_draft.dart` (`PlaceRules`, ported in `rules.go` with their tests).
- **Endpoints:** `GET /donate/recipients`, `GET /donate/recipient` (public), `POST /donate/give` (me); staff with the orders permission: `POST /admin/donate/places/save|remove`.
- **Tables:** `donate_places` (soft remove with `removed_at`), `donate_needs(place_id, book_id, wanted, received)` (migration `0007`).
- **Give:** one transaction that locks the place, saves an order through `orders.Insert` (`is_donation`, `donate_place_id`, free delivery, the donor note on the gift card) and adds to `received`. Donors buy the cheapest printed Edition that can be ordered. Refused: cash on delivery, a quantity of less than 1 or more than still needed, an unknown place or book.
- **Save:** replaces the place and its needs (received copies start again from 0, as in the app); unknown ids and rule breaks are refused.
