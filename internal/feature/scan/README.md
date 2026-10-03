# scan

`GET /scan/lookup?isbn=`: the Book with an Edition of that ISBN (`{bookId, title, author, isbn, coverSeed, newPriceBdt}`), or `null`. Public. Hidden books are not offered. Frontend: `lib/features/scan/data/sources/scan_fake_api.dart`.
