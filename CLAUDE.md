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
```

Copy `.env.example` → `.env` and fill in credentials before running.

## Architecture

3-layer clean architecture with explicit dependency injection wired in `cmd/cmd.go`:

```
Handler → Service → Repository
```

There are four domains — `location`, `cafe`, `filter`, and `quicksearch` — each with a handler/service/repository/model file.

- `cmd/cmd.go` — entrypoint: loads config, configures CORS (all origins), creates pgxpool, wires all layers, registers routes, starts Gin router
- `config/config.go` — reads `DB_HOST/PORT/USER/PASSWORD/NAME` and `APP_PORT` (default 8080); exposes `DSN()`
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

Key tables: `location`, `location_image`, `cafe`, `cafe_review`, `tag`, `cafe_tag`, `cafe_price`, `rating_type_label`, `rating_category`, `cafe_rating`.

Location name search uses a GIN trigram index and `similarity()` ordering — keep queries consistent with this pattern. Area/district containment uses PostGIS polygons (`ST_PointOnSurface` / `ST_Contains`).

## API Endpoints

See `docs/api-contracts.md` for full request/response schemas.

- `GET /health`
- `GET /v1/quicksearch?q=<query>&type=<all|location|filter|cafe|poi|area|district>` — typeahead
- `GET /v1/location` — list districts
- `GET /v1/location/:id` — location (area/POI/district) detail
- `GET /v1/search/cafes` — cafe discovery (polygon / radius / global modes; tag, rating, price, open-hour, featured filters; sort + pagination)
- `GET /v1/cafe/:id` — full cafe detail
- `GET /v1/cafe/:id/review` — cafe review and ratings
- `GET /v1/filters?enrich_content=<bool>` — available filter options (tags, rating categories)
