# checkout

Turns the cart into an order, all or nothing, and holds the coupons.

- **Frontend files:** `lib/features/checkout/data/sources/{checkout_fake_api,coupon_fake_store,coupon_admin_fake_api}.dart`, `lib/features/checkout/domain/` (totals).
- **Endpoints:** `GET /coupons/check`, `POST /orders/place`, `GET /admin/coupons`, `POST /admin/coupons/create` (staff with the orders permission).
- **Tables:** `coupons` (migration `0006`).
- **Place:** one transaction. It locks the reader, resolves the address, prices the cart (`Compute`, a port of the Dart totals), spends points and wallet, takes the order number from the sequence, saves the order, takes the copies out of stock, counts the sale and empties the cart. Any failure leaves nothing behind (`TestPlaceIsAllOrNothing`). The catalog cache is dropped after the commit.
- **Refusals:** `cart_empty`, `address_unknown`, `payment_invalid`, `gift_name_missing`, `gift_message_too_long`, `coupon_code_taken`, `coupon_invalid`. An unknown or expired coupon code at checkout just gives no discount.
- Coupons work until their end date; codes are stored in capitals.
