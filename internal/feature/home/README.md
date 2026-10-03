# home

Home: the Banners carousel, the Season hero card and the Ayah of the day. All public.

- **Frontend files:** `lib/features/home/data/sources/{home_fake_api,ayah_fake_api,season_picker,season_fixtures,banner_fixtures,ayah_fixtures}.dart`.
- **Endpoints:** `GET /home/banners`, `GET /home/season` (both take an optional `?date=yyyy-mm-dd`), `GET /islamic/ayah-of-the-day`.
- **Season:** `season.go` ports `SeasonPicker` and `SeasonFixtures` (Ramadan first, then the month; dates in the app timezone). Staff can force one (`app_config` key `season_override`, written by `catalogadmin`).
- **Tables:** `banners` (explicit `position`), `app_config`, `ayahs` (migration `0004`).
