# Deploying Waraqah's backend

The free-tier setup of BACKEND_PLAN.md §19: the API on Render, Postgres on Neon, photos on Cloudinary, OTP email through Resend (or Brevo), optional Gemini. Everything is configured with environment variables (`.env.example` lists them all).

## 1. Accounts and secrets

| Service | What to create | Variables |
|---|---|---|
| Neon | A project (region near Singapore). Copy the **pooled** and the **direct** connection strings. | `DATABASE_URL` (pooled), `DATABASE_URL_DIRECT` (direct) |
| Cloudinary | A free account; the dashboard shows the cloud name, API key and secret. | `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`, `CLOUDINARY_API_SECRET` |
| Resend | An API key and a sender address on a verified domain (or Brevo: `EMAIL_PROVIDER=brevo`). | `EMAIL_API_KEY`, `EMAIL_FROM` |
| Google Cloud | OAuth client ids of the Android, iOS and web apps (for Google sign-in). | `GOOGLE_OAUTH_CLIENT_IDS` (comma separated) |
| Google AI Studio | Optional: a Gemini API key and model id. Without them the assistant answers with its rules. | `GEMINI_API_KEY`, `GEMINI_MODEL` |

## 2. The API on Render

1. Render → **New → Blueprint** → pick this repository. `render.yaml` creates the free web service `waraqah-api` (Go build with the commit and build time stamped for `/version`, health check `/readyz`, deploy of `main` only after CI passes, `JWT_SECRET` generated).
2. Fill in the values the blueprint marks as not synced (the table above, `PUBLIC_BASE_URL`, `CORS_ALLOWED_ORIGINS` with the web app's origin, and `SEED_DEMO_PASSWORD`).
3. Deploy. `RUN_MIGRATIONS_ON_START=true` migrates the database before the server listens. In production the server refuses to start with `OTP_DEV_CODE` set, `EMAIL_PROVIDER=log`, `CLOUDINARY_FAKE=true` or a short `JWT_SECRET`.

## 3. Seed the demo data once

From the Render shell of the service (or locally with the production variables):

```bash
SEED_ALLOW_PRODUCTION=true ./bin/seed
```

Seeding is idempotent; it creates the demo accounts (`reader@`, `admin@`, `moderator@`, `catalog@`, `support@waraqah.test`, password `SEED_DEMO_PASSWORD`) and the catalog, marketplace and community data the app shows on its fake API. Roles change with `./bin/admin set-role <email> <role>`.

## 4. Check it

```bash
make smoke API_BASE_URL=https://<service>.onrender.com/v1 SEED_DEMO_PASSWORD=<the demo password>
```

It calls the health endpoints, the public reads, one endpoint per feature as `reader@` and the staff screens as `admin@`.

## 5. Point the app at it

```bash
flutter run --dart-define=API_BASE_URL=https://<service>.onrender.com/v1
flutter build web --dart-define=API_BASE_URL=https://<service>.onrender.com/v1   # optional web app
```

## Free-tier budgets

- **Render:** 750 hours a month for the service; it sleeps after 15 minutes idle and wakes in about a minute (open the app a minute before a demo). The jobs (demo bot, courier pickups) catch up when it wakes.
- **Neon:** 0.5 GB. The seeded database is about 13 MB; photos are not stored in it (Cloudinary), except return and dispute photos (base64, at most 3 each, `MAX_IMAGE_MB` apiece). `DB_MAX_CONNS=5` keeps CU-hours low.
- **Cloudinary:** 25 credits a month; listing photos are shown as 300 px thumbnails and removed slots are deleted.
- **Email and Gemini:** OTP sends are limited per contact (`OTP_RESEND_SECONDS`) and per IP (`AUTH_RATE_PER_MIN`); the assistant per reader (`AI_RATE_PER_MIN`).
