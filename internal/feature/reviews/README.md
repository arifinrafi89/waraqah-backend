# reviews

Readers' reviews of Books: one per reader per Book, Verified Purchase, and the Book's rating kept as their average.

- **Frontend files:** `lib/features/reviews/data/sources/{review_fake_api,review_fake_store,review_record,review_seed}.dart`; rules `review_rules.dart` (`rules.go`, ported with its test).
- **Endpoints:** `GET /reviews?bookId=` (public, optional token), `POST /reviews/save|delete` (me). Each answers `{average, count, mine, reviews}`, newest first.
- **Tables:** `reviews` (primary key `(book_id, user_id)`, `seed_verified` for seeded readers) (migration `0011`).
- **Verified:** a seeded reader's flag, or a delivered order (not a donation) with the Book, through `orders.DeliveredBooks`.
- **Rating:** saving or deleting recomputes `books.rating` in the same transaction (`catalog.Store.SetRating`) and drops the catalog cache; a Book whose last review goes keeps its rating, as the fake does.
- **Moderation:** `Reviews` is registered as the `review` subject; removing one keeps the rating right.
