# catalogadmin

Admin → Catalog for Staff with the catalog permission (catalog manager, super admin): Books and Editions, stock, Categories, Authors, Publishers, Banners, Collections, Staff Booklists, the forced Home Season, and the ISBN lookup, low-stock and CSV import tools. 25 endpoints, all `staff:catalog`.

- **Frontend files:** `lib/features/catalog_admin/data/sources/{catalog_admin_fake_api,catalog_admin_fake_store,catalog_admin_fake_records,catalog_admin_fake_banners,catalog_admin_fake_lists,catalog_tools_fake_api,catalog_import_fake,isbn_lookup_fixtures}.dart`; rules `catalog_admin_rules.dart`, `list_rules.dart`.
- **Rules:** `rules.go` ports `CatalogAdminRules` and `ListRules` (and `Isbn`, in `internal/platform/isbn`) with their tests.
- **Tables:** it edits the catalog and Home tables (`books`, `editions`, `categories`, `authors`, `publishers`, `collections`, `booklists`, `banners`, `app_config`, `price_lows`); the owners are `catalog` and `home`. Every change drops the catalog read cache.
- **Refusals** carry the codes listed in `docs/error-codes.md` (`book_invalid`, `record_in_use`, `banner_invalid`, `list_invalid`, `stock_invalid`, ...). The `by` fields in bodies are ignored.
- **`Sweeper`** (alerts.Sweeper) runs after a Book save, a stock change or an import; it does nothing until the alerts feature (T11) plugs in.
- A price change keeps the cheaper of the two prices as the 30-day low (`price_lows`).
