# BDGCafe Backend

Go REST API for [BDGCafe](https://bdgcafe.com) — a Bandung coffee shop review and discovery site.

## Stack

- **Go** with [Gin](https://github.com/gin-gonic/gin)
- **PostgreSQL** with `pg_trgm` and `postgis` extensions
- **pgx/v5** connection pool

## Getting started

### Prerequisites

- Go 1.26+
- PostgreSQL with `pg_trgm` and `postgis` extensions enabled
- Python 3 (for the data seeders)

### Setup

```bash
cp .env.example .env
# Fill in DB credentials in .env

# 1. Apply schema (extensions, enums, tables, indexes)
psql -U postgres -d bandung_coffeeshop -f migrations/001_init.sql

# 2. Seed cafes from cafe_master.json
python3 migrations/002_cafe_seeder.py

# 3. Seed reference + i18n data (tags, rating categories, translations)
psql -U postgres -d bandung_coffeeshop -f migrations/003_data_seeder.sql

# 4. Seed area/district polygons (OpenStreetMap / Nominatim)
python3 migrations/004_area_district_seeder.py

# Run
go run ./cmd
```

### Configuration

| Variable      | Default              | Description       |
|---------------|----------------------|-------------------|
| `APP_PORT`    | `8080`               | HTTP port         |
| `DB_HOST`     | `localhost`          | PostgreSQL host   |
| `DB_PORT`     | `5432`               | PostgreSQL port   |
| `DB_USER`     | `postgres`           | Database user     |
| `DB_PASSWORD` | —                    | Database password |
| `DB_NAME`     | `bandung_coffeeshop` | Database name     |

## Development

```bash
go run ./cmd           # Run server
go build -o app ./cmd/cmd.go  # Build binary
#go test ./...          # Run tests
#go test ./... -race -cover    # Tests as CI runs them
go vet ./...           # Static analysis
go fmt ./...           # Format code
go mod tidy            # Sync dependencies
```

Tests are pure unit tests (testify + testify/mock; repositories mocked via
interfaces) and need no database. Currently, this is still in feat/test branch and not merged to main yet.

### Docker

```bash
docker build -t bdgcafe .                       # Build the service image (has a /health HEALTHCHECK)
#docker build -f Dockerfile.test -t bdgcafe-test .  # Run the test suite as a CI gate
```

## Architecture

3-layer clean architecture wired in `cmd/cmd.go`:

```
Handler → Service → Repository
```

| Layer    | Package              | Responsibility                             |
|----------|----------------------|--------------------------------------------|
| HTTP     | `handler/`           | Parse params, call service, write response |
| Business | `service/`           | Input validation, domain rules             |
| Data     | `repository/`        | Raw pgx queries against PostgreSQL         |
| Shared   | `model/`             | Request/response DTOs                      |
| Util     | `helper/response.go` | JSON envelope helpers                      |

## API

Base URL: `http://localhost:8080`

All `/v1` responses use a standard envelope:

```json
{ "success": true, "data": { ... } }
{ "success": false, "error": "human readable message" }
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/v1/quicksearch` | Typeahead search over cafes, POIs, areas, districts |
| `GET` | `/v1/location` | List districts |
| `GET` | `/v1/location/:id` | Detail for an area / POI / district |
| `GET` | `/v1/search/cafes` | Cafe discovery with polygon, radius, or global mode |
| `GET` | `/v1/cafe/:id` | Full detail for a single cafe |
| `GET` | `/v1/cafe/:id/review` | Review and ratings for a single cafe |
| `GET` | `/v1/filters` | Available filter options (tags, rating categories) |

See [`docs/api-contracts.md`](docs/api-contracts.md) for full request/response schemas.

### Localization

Content is bilingual. Send `Accept-Language: en` or `Accept-Language: id` to
select the locale; it defaults to Indonesian (`id`) when the header is absent or
unrecognised.

### Quick examples

```bash
# Typeahead
GET /v1/quicksearch?q=dreezel
GET /v1/quicksearch?q=dago&type=area

# Search cafes inside an area (tags is a comma-separated list)
GET /v1/search/cafes?query_id=dago&query_type=area&tags=wifi-friendly

# Radius search from coordinates, sorted by distance
GET /v1/search/cafes?query_coords=-6.9039,107.6186&radius_max=2000&sort=distance

# Cafe detail and review
GET /v1/cafe/accio-coffee
GET /v1/cafe/accio-coffee/review
```

## License

Copyright © 2026 Muhammad Raditya.

This project is licensed under the
[Creative Commons Attribution-NonCommercial 4.0 International License (CC BY-NC 4.0)](https://creativecommons.org/licenses/by-nc/4.0/).
See [`LICENSE`](LICENSE) for the full text.

You are free to **fork, use, and adapt** this code **for personal,
non-commercial purposes**, provided you give appropriate credit and link back
to this repository. **Commercial use is not permitted.**
