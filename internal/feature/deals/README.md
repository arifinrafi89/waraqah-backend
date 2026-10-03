# deals

`GET /deals` (public): the flash sale, the bundles and the pre-orders running now. Frontend: `lib/features/deals/data/sources/{deals_fake_api,deals_fake_store}.dart`.

- **Tables:** `flash_sale_items`, `bundles`, `preorders` (migration `0005`). The flash sale runs all day, every day: it ends at the next midnight in Dhaka; a pre-order shows a release date 40 days out, as the fake API did.
- The cart asks `FlashPrice` and `BundleByID` for the prices that apply.
