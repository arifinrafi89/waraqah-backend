# readers

A reader's public page and who follows whom.

- **Frontend files:** `lib/features/readers/data/sources/{reader_fake_api,follow_fake_store}.dart`, model `reader_model.dart`.
- **Endpoints:** `GET /readers/detail?id=` (public, optional token), `POST /readers/follow` (me) answering the page.
- **Tables:** `follows` (migration `0011`); names, areas and member-since come from `users`, privacy from `profile_prefs`.
- **Privacy:** with `profileVisible` off, or for a reader blocked either way, the page shows only the name and the follow state; with `activityVisible` off the Bites are not counted. The reader always sees their own page.
- **Follow:** not yourself, not someone unknown (`reader_unknown`) or blocked (`follow_refused`); the followed reader is told the first time. `Following` feeds the Bites "following" tab.
