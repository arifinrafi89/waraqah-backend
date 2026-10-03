# catalog

The storefront: books and editions, search, suggestions, details, look inside, series, price lows, used options, records (categories, subjects, authors, publishers), collections, experts, booklists and the questions readers ask about a book.

- **Frontend files:** `lib/features/catalog/data/sources/{book_fake_api,book_suggest_fake_api,book_questions_fake_api,collection_fake_api,booklist_fake_api,phonetic_key,book_search_match,book_sort,book_edition_filter}.dart`, fixtures `*_fixtures.dart`, `seed/`; rules `delivery_estimate.dart`.
- **Endpoints (25):** `GET /books`, `/books/detail|details|look-inside|price-lows|series|used-options|suggest|did-you-mean|questions`, `POST /books/questions/ask|answer`, `GET /categories|subjects|authors/detail|publishers/detail|series/detail|collections|collections/detail|experts|experts/detail|booklists|booklists/detail`, `POST /booklists/mine/save|delete`.
- **Tables:** `books`, `editions`, `categories`, `authors`, `publishers`, `subjects`, `book_details`, `look_inside`, `series`, `book_questions`, `book_answers`, `experts`, `collections`, `booklists`, `price_lows`, `sales_by_month` (migration `0003`). Queries: `db/queries/catalog.sql`.
- **Search runs in memory.** `Store.Snapshot` keeps the whole catalog (a few hundred books) and `snapshot.go`, `search.go`, `suggest.go` are ports of the Dart ranking, phonetic key, suggestions and "did you mean". Call `Store.Invalidate()` after any change to the catalog tables.
- **Rules ported with their tests:** `phonetic.go` (`PhoneticKey`, `Levenshtein`), `search.go` (`BookSearchMatch`, `BookEditionFilter`, `BookSort`), `delivery.go` (`DeliveryEstimate`).
- **Other features use `catalog.Books`** (`Find`, `FindEdition`, `Visible`), implemented by `Store`.
- Hidden books show only with `includeHidden=true` **and** a catalog staff token. The name and staff flag of a question or answer come from the token, never the body.
- Certified Used copies come from `UsedStock` (a stand-in until Sell Back, T16, plugs in).
