# cart

The cart of a reader: one line per item, quantities capped per order, nothing that cannot be ordered. Every endpoint answers the whole cart.

- **Frontend files:** `lib/features/cart/data/sources/{cart_fake_api,cart_fake_store,bundle_cart_line,used_cart_line}.dart`.
- **Endpoints:** `GET /cart`, `POST /cart/add|update|remove`.
- **Table:** `cart_lines(user_id, kind, item_id, quantity)` (migration `0005`). Only what was added is stored; prices come from the catalog and the deals when the cart is read, so a price or stock change shows at once. Line ids are `<kind>-<itemId>`, as in the app.
- **Kinds:** `edition` (flash price applies), `bundle`, `certifiedUsed` (needs Sell Back, T16: `UsedStock`), `listing` (never sold through the cart; buyers make an offer). An item that cannot be added leaves the cart as it is and the answer carries `X-Waraqah-Error: cart_item_unknown`.
- **Caps:** an eBook 1, a printed Edition `min(stock, 10)` (10 for a pre-order), a bundle 5.
- Checkout (T12) reads and empties the cart with `CartIn` and `ClearIn` inside its transaction.
