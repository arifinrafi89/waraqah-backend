# orders

Orders of a reader and the staff queue.

- **Frontend files:** `lib/features/orders/data/sources/{order_fake_api,order_fake_store,order_admin_fake_api,order_reorder}.dart`.
- **Endpoints:** `GET /orders`, `GET /orders/details`, `POST /orders/cancel|return|reorder`; staff (orders permission): `GET /admin/orders`, `POST /admin/orders/advance|return`.
- **Tables:** `orders`, `order_lines`, `order_history`, `order_returns` (migration `0006`). Return photos stay base64 as the app reads them (checked as images, at most 3 kept).
- **Rules:** a cancel is only possible before shipping; it gives back points, takes back earned points, refunds what was paid to the wallet and restocks. A return is asked once, within 7 days of delivery; approving refunds the books to the wallet. Staff move an order one step at a time. The reader is notified of both.
- `orders.Insert` is shared with donate (T13); `DeliveredBooks` is for reviews and shelves (T17).
