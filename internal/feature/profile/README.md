# profile

The reader profile (name, phone, photo), settings (muted notification groups, privacy switches), saved addresses, the Bangladesh geography list, and account deletion.

- **Frontend files:** `lib/features/profile/data/sources/{profile_fake_api,profile_fake_store,address_fake_store}.dart`, `geo/*.dart`; rules `profile_rules.dart`, `address_rules.dart`.
- **Endpoints:** `GET /profile`, `POST /profile/save`, `GET /profile/prefs`, `POST /profile/prefs/save`, `GET /addresses`, `POST /addresses/save|default|delete`, `GET /geo` (public), `POST /auth/delete`.
- **Tables:** `profile_prefs`, `addresses` (a partial unique index keeps one default per reader), and `users.photo_data` (migration `0002`). Queries: `db/queries/profile.sql`.
- **Rules:** `rules.go` and `address_rules.go` are ports of `ProfileRules` and `AddressRules`, with their tests.
- **Geo:** `seed/geo.json` is embedded in the binary (`seed/embed.go`) and served as is.
- **Account deletion** anonymises the account, signs it out everywhere and runs `Service.Hooks`, where later features hide the listings and Bites of the reader.
- A guest sees the empty profile, no addresses and the default settings.
