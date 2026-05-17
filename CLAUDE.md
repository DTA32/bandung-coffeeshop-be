# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## About

BDGCafe is a personal Bandung coffee shop review and discovery site. This Go backend serves the search and discovery API. Reviews are single-author and opinionated (not crowd-sourced).

## Commands

```bash
# Run the server
go run ./cmd

# Build binary
go build -o app ./cmd

# Tests (testify + go.uber.org/mock available; no tests written yet)
go test ./...

# Static analysis and formatting
go vet ./...
go fmt ./...

# Sync dependencies
go mod tidy

# Apply DB schema (requires pg_trgm and postgis extensions)
psql -U postgres -d bandung_coffeeshop -f migrations/001_init.sql

# Seed data from master.json (reads .env for DB creds)
python3 migrations/generate_seed.py
```

Copy `.env.example` → `.env` and fill in credentials before running.

## Architecture

3-layer clean architecture with explicit dependency injection wired in `cmd/cmd.go`:

```
Handler → Service → Repository
```

- `cmd/cmd.go` — entrypoint: loads config, creates pgxpool, wires all layers, starts Gin router
- `config/config.go` — reads `DB_HOST/PORT/USER/PASSWORD/NAME` and `APP_PORT` (default 8080); exposes `DSN()`
- `handler/` — Gin HTTP layer; extracts params, calls service, responds via helpers
- `service/` — input validation and business rules; maps domain errors to handler-visible errors
- `repository/` — raw pgx queries against PostgreSQL
- `model/` — shared request/response DTOs
- `helper/response.go` — JSON envelope: `{"success": true, "data": ...}` / `{"success": false, "error": ...}`
- `constants/constants.go` — enums for location types (`cafe`, `poi`, `neighbourhood`, `area`, `district`) and rating categories

## Database

PostgreSQL with two required extensions: `pg_trgm` (trigram similarity for fuzzy name search) and `postgis` (geographic coordinates). Schema in `migrations/001_init.sql`; ERD in `docs/erd.mermaid`.

Key tables: `location`, `cafe`, `cafe_review`, `cafe_rating`, `rating_category`, `cafe_price`, `tag`, `cafe_tag`, `location_image`.

Location name search uses a GIN trigram index and `similarity()` ordering — keep queries consistent with this pattern.

## API Endpoints

- `GET /health`
- `GET /v1/quicksearch?q=<query>&type=<location_type>`
