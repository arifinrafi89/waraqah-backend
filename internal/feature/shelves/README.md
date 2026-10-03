# shelves

The reader's shelves (Want to Read, Reading, Finished), reading progress, reading days and the yearly goal.

- **Frontend files:** `lib/features/shelves/data/sources/{shelf_fake_api,shelf_fake_store,shelf_seed,reading_log}.dart`; rules `progress_rules.dart` (`rules.go`, ported with its test).
- **Endpoints (me):** `GET /shelves`, `POST /shelves/move|progress`, `GET /reading/stats`, `POST /reading/goal`.
- **Tables:** `shelf_entries` (`position` breaks ties of Books added together), `shelf_order_sync`, `reading_days` (Dhaka dates), `reading_goals` (migration `0011`).
- **Delivered orders:** each Book of a delivered order the reader kept (not a donation or gift) goes on Want to Read once, through `orders.DeliveredBooks`; taking it off is respected.
- **Rules:** a shelf name the app does not know takes the Book off (as the fake does); progress by pages sets the percentage, 100% finishes the Book, and moving forward counts today as a reading day. Stats are per year in Dhaka time: goal, Books finished per month, streak and the three Categories read most.
