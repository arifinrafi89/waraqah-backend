# handledsale

Used books bought through Waraqah: the buyer pays in the app, the seller sends the book, and Waraqah holds the money until the buyer confirms or a moderator settles a dispute.

- **Frontend files:** `lib/features/handled_sale/data/sources/{handled_sale_fake_api,handled_sale_fake_store,handled_sale_fake_steps,handled_sale_fake_money,fake_sale,sale_changes,handled_sale_seed,sale_live_source}.dart`; rules `sale_math.dart` (`math.go`, ported with its test).
- **Endpoints (me):** `POST /sales/buy|step|dispute|payout`, `GET /sales/detail|mine|earnings`, `GET /sales/live` (SSE). **Staff (moderate):** `GET /sales/disputes`, `POST /sales/disputes/settle`.
- **Tables:** `handled_sales` (ids `HS-<n>` from `handled_sale_seq`; the fee is fixed at sale time), `payouts` (migration `0010`).
- **Money:** `buy` reserves the listing through `p2p.Status` (prepaid methods only); `cancel` before sending and a refund settlement credit the buyer's wallet (`saleRefund`, price + delivery); earnings are `SaleMath.sellerGets` of completed and released sales, `payout` sends what is left.
- **Disputes:** photos are checked as images and kept as base64 (the app decodes them, contract v1). `settle` writes the audit log through `moderation.Record` with the staff member of the token (`by` is ignored).
- **Live:** every move publishes `{seq, saleId}` on `sales:<buyer>` and `sales:<seller>`, and on `sales:moderators` for disputes; moderators' streams include that topic.
- **Demo bot** (`DEMO_MODE`, `DEMO_BOT_DELAY`): a seeded demo seller (`p-...`) sends a paid book once the sale is old enough; it is a job (`SendDemoSales`), idempotent, and also scheduled right after a purchase.
