# dashboard

The staff dashboard and the search log behind its top searches.

- **Frontend files:** `lib/features/admin/data/sources/{dashboard_fake_api,dashboard_fake_store,search_log}.dart`, model `admin_dashboard_model.dart`.
- **Endpoint:** `GET /admin/dashboard` (any staff role): today's orders and sales (Dhaka day, not cancelled), orders to ship, listings waiting, open reports, open disputes, top five searches and the five most requested books; plus the additive `returnsWaiting`, `sellBackWaiting` and `lowStock` the app ignores today.
- **Sources:** every number comes from its feature through a read interface (orders, p2p, moderation, handled sales, Sell Back, catalog admin, book requests); the dashboard reads no other feature's tables.
- **Search log** (`search.Log`): `/books?q=` counts reader searches per term and Dhaka day; a live search that grows or shrinks from the same searcher's last term within two minutes counts once, terms under three characters and staff `includeHidden` lists are not counted.
- **Tables:** `search_log` (migration `0012`), seeded with the demo counts of `search_log.dart`.
