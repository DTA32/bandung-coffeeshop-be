# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## About

BDGCafe is a personal Bandung coffee shop review and discovery site. This Go backend serves the search and discovery API. Reviews are single-author and opinionated (not crowd-sourced).

## Commands

```bash
# Run the server
go run ./cmd

# Build binary
go build -o app ./cmd/cmd.go

# Tests — testify + testify/mock, white-box (same-package _test.go),
# repositories mocked via consumer interfaces; no DB required.
# update: currently still in feat/test branch, not merged to main yet
#go test ./...
#go test ./... -race -cover   # how CI runs it

# Static analysis and formatting
go vet ./...
go fmt ./...

# Sync dependencies
go mod tidy

# Docker: build/run the service (multi-stage, includes a /health HEALTHCHECK)
docker build -t bdgcafe .
# Docker: run the test suite as a CI gate (build fails if go vet / go test fails)
#docker build -f Dockerfile.test -t bdgcafe-test .
```

### Database setup

```bash
# 1. Schema — extensions (pg_trgm, postgis), enums, tables, and indexes
psql -U postgres -d bandung_coffeeshop -f migrations/001_init.sql

# 2. Seed cafes from cafe_master.json (reads .env for DB creds)
python3 migrations/002_cafe_seeder.py

# 3. Reference + i18n data: tags, rating categories, Indonesian translations,
#    rating descriptions, rating_type_label rows
psql -U postgres -d bandung_coffeeshop -f migrations/003_data_seeder.sql

# 4. Seed area/district polygons from OpenStreetMap (Nominatim) + hardcoded regions
python3 migrations/004_area_district_seeder.py

# 5. cafe.weather column (clear / cloudy / rain) for "Match the weather"
psql -U postgres -d bandung_coffeeshop -f migrations/005_cafe_weather.sql

# 6–8. Freshness pipeline (BCF-11 / BCF-12): cafe.phone + cafe.menu, collector
#      tables (source_target, source_snapshot, change_proposal, scrape_run) and cafe_feed
psql -U postgres -d bandung_coffeeshop -f migrations/006_cafe_contact.sql
psql -U postgres -d bandung_coffeeshop -f migrations/007_collector.sql
psql -U postgres -d bandung_coffeeshop -f migrations/008_cafe_feed.sql

# 9. users table (Google sign-in accounts; sessions live in Redis)
psql -U postgres -d bandung_coffeeshop -f migrations/009_users.sql

# 10. One-off: align cafe.updated_at with each cafe's latest review (BCF-35)
psql -U postgres -d bandung_coffeeshop -f migrations/010_cafe_updated_at_backfill.sql

# 11. User reviews, additive part (BCF-29). Needs Radit's users row (sign in
#     once first); every existing review becomes his, approved + pinned.
psql -U postgres -d bandung_coffeeshop -v radit_email=<radit's google email> \
     -f migrations/011_user_reviews.sql

# 12. User reviews cleanup (BCF-29): drops is_subjective / personal_score /
#     content_indo, moves cafe_rating to UNIQUE (review_id, category_type)
psql -U postgres -d bandung_coffeeshop -f migrations/012_user_reviews_cleanup.sql
```

Production run order for the user-reviews release: run 010 + 011 **before**
deploying the new BE, then deploy the BE and `bandung-coffeeshop-admin`, then
run 012 right away. The BE works with and without 012 (it never reads the
dropped columns). Until 012 runs, cafe_rating is still UNIQUE on
`(cafe_id, category_type)`, so a second reviewer's ratings on an already-rated
cafe fail to save; and after 011 the old admin can no longer create reviews
(`reviewer_name` is NOT NULL).

This repo owns the schema for every BDGCafe service. `bandung-coffeeshop-admin`
(mission control, `mission-control.bdgcafe.com`) and `bandung-coffeeshop-scraper`
(daily Google Maps / Instagram / Linktree CronJobs) connect to the same database and
only consume it. See `docs/freshness-pipeline-plan.md`.

Copy `.env.example` → `.env` and fill in credentials before running.

## Architecture

3-layer clean architecture with explicit dependency injection wired in `cmd/cmd.go`:

```
Handler → Service → Repository
```

There are five domains — `location`, `cafe`, `filter`, `quicksearch`, and `auth` — each with a handler/service/repository/model file. `weather` is a service/repository pair with no handler of its own: it backs `weather=current` on the cafe search. `auth` spans `repository/user.go` (Postgres), `repository/session.go` (Redis) and `repository/google.go` (OAuth + userinfo) behind `service/auth.go` / `handler/auth.go`.

- `cmd/cmd.go` — entrypoint: loads config, configures CORS (`ALLOWED_ORIGINS`, credentials allowed), creates pgxpool, wires all layers, registers routes and the `/v1` middleware, starts Gin router
- `config/config.go` — reads `DB_HOST/PORT/USER/PASSWORD/NAME`, `APP_PORT` (default 8080), `REDIS_HOST/PORT/DB`, `WEATHERAPI_KEY`, and auth: `GOOGLE_CLIENT_ID/SECRET`, `PUBLIC_BASE_URL`, `AUTH_CALLBACK_URL` (default `${PUBLIC_BASE_URL}/api/v1/auth/google/callback`), `ALLOWED_ORIGINS` (comma-separated; default `PUBLIC_BASE_URL`, else `http://localhost:3000`), `COOKIE_SECURE` (default true); exposes `DSN()`, `RedisAddr()`, `GoogleAuthEnabled()`
- `cache/redis.go` — optional Redis client (disabled when `REDIS_HOST` is empty). As a cache every error is treated as a miss; it also backs sessions (`Pipelined` for the multi-key writes)
- `middleware/auth.go` — `LoadUser` on the whole `/v1` group: reads the `bdg_session` cookie (no cookie → no Redis/DB hit), resolves the user, drops sessions of deactivated users, degrades to anonymous on any error. `RequireUser` → 401. `CurrentUser(c)` reads the user from the Gin context — use it in handlers that need the signed-in user.
- `middleware/origin.go` — `RequireAllowedOrigin`: 403 for POST/PUT/DELETE whose `Origin` is present and not in `ALLOWED_ORIGINS`
- `service/auth.go` — Google sign-in (state + PKCE round-tripped in the `bdg_oauth` cookie), user upsert, session issue/lookup/revoke. Sessions: `session:<sha256(token)>` → user id and `user_sessions:<id>` set, both a fixed 30-day TTL; only the hash is stored. Auth routes answer 503 when Google creds or Redis are missing
- `service/weather.go` — Bandung's current weather from weatherapi.com, cached 30 min in Redis (plus an in-process copy and a 24 h last-known fallback); folds provider condition codes into `clear` / `cloudy` / `rain`
- `handler/` — Gin HTTP layer; parses/validates params, calls service, maps domain errors to HTTP status, responds via helpers. Each handler defines a consumer interface over its service (the seam used for unit tests).
- `service/` — input validation and business rules; maps domain errors to handler-visible errors. Defines consumer interfaces over its repository.
- `repository/` — raw pgx queries against PostgreSQL; owns the `Err*NotFound` sentinel errors
- `model/` — shared request/response DTOs
- `helper/response.go` — JSON envelope: `{"success": true, "data": ...}` / `{"success": false, "error": ...}` via `helper.Success` / `helper.Error`
- `helper/lang.go` — resolves request locale from the `Accept-Language` header (see i18n below)
- `constants/constants.go` — enums for location types (`cafe`, `poi`, `area`, `district`), quicksearch type selectors (`all`, `location`, `filter`), sort/order, and languages
- `docs/api-contracts.md` — full request/response schemas for every endpoint; keep it in sync when changing the API
- `docs/erd.mermaid` — database ERD

## Internationalization

Content is bilingual (English / Indonesian). Clients select a locale via the `Accept-Language` header; `helper.Lang` parses it (tolerating `q=` weighted lists), supports `en` and `id`, and falls back to `constants.DefaultLang` (`id`) when absent or unrecognised. The resolved `lang` is threaded down to the repository, which selects the matching `*_indo` column.

## Testing

Unit tests are white-box (same-package `_test.go`) using `testify` assertions and `testify/mock`. Each layer is tested against a hand-written mock of the consumer interface it depends on (`mocks_test.go` per package); compile-time `var _ iface = (*mock)(nil)` checks keep mocks in sync. No database is needed — repository SQL is deferred to integration testing. The suite runs under `-race -cover` in `Dockerfile.test` as a CI gate.
Currently, this is still in feat/test branch, not merged to main yet.

## Database

PostgreSQL with two required extensions: `pg_trgm` (trigram similarity for fuzzy name search) and `postgis` (geographic coordinates). Schema in `migrations/001_init.sql`; ERD in `docs/erd.mermaid`.

Key tables: `location`, `location_image`, `cafe`, `cafe_review`, `tag`, `cafe_tag`, `cafe_price`, `rating_type_label`, `rating_category`, `cafe_rating`, `cafe_feed`, `users` (accounts keyed by `(provider, provider_sub)`; sessions are in Redis, not the DB). Collector tables (written by the scraper, reviewed in mission control): `source_target`, `source_snapshot`, `change_proposal`, `scrape_run`.

Location name search uses a GIN trigram index and `similarity()` ordering — keep queries consistent with this pattern. Area/district containment uses PostGIS polygons (`ST_PointOnSurface` / `ST_Contains`).

## API Endpoints

See `docs/api-contracts.md` for full request/response schemas.

- `GET /health`
- `GET /v1/quicksearch?q=<query>&type=<all|location|filter|cafe|poi|area|district>` — typeahead
- `GET /v1/location` — list districts
- `GET /v1/location/:id` — location (area/POI/district) detail
- `GET /v1/search/cafes` — cafe discovery (polygon / radius / global modes; tag, rating, price, open-hour, featured, status (`active`/`closed`), weather filters; sort + pagination)
- `GET /v1/cafe/random` — one random active cafe ("Surprise me")
- `GET /v1/cafe/:id` — full cafe detail
- `GET /v1/cafe/:id/review` — cafe review and ratings
- `GET /v1/cafe/:id/feed` — latest updates feed (up to 10 visible, unexpired `cafe_feed` entries)
- `GET /v1/filters?enrich_content=<bool>` — available filter options (tags, rating categories, price tiers, weather)
- `GET /v1/auth/:provider?redirect=<path>` — start sign-in (`google` only); 302 to Google
- `GET /v1/auth/:provider/callback` — OAuth callback; sets `bdg_session`, 302 to `PUBLIC_BASE_URL + redirect` (`?auth_error=deactivated|failed` on failure)
- `GET /v1/me` — signed-in user (401 when anonymous)
- `POST /v1/auth/logout` — revoke session, 204
