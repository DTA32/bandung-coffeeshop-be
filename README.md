# BDGCafe Backend

Go REST API for [BDGCafe](https://bdgcafe.com) — a personal Bandung coffee shop review and discovery site. Reviews are from personal experience, not crowdsourced.

## Stack

- **Go** with [Gin](https://github.com/gin-gonic/gin)
- **PostgreSQL** with `pg_trgm` and `postgis` extensions
- **pgx/v5** connection pool

## Getting started

### Prerequisites

- Go 1.21+
- PostgreSQL with `pg_trgm` and `postgis` extensions enabled

### Setup

```bash
cp .env.example .env
# Fill in DB credentials in .env

# Apply schema
psql -U postgres -d bandung_coffeeshop -f migrations/001_init.sql
psql -U postgres -d bandung_coffeeshop -f migrations/003_create_indexes.sql
psql -U postgres -d bandung_coffeeshop -f migrations/004_tag_category_seeder.sql

# Seed cafe data
python3 migrations/002_cafe_seeder.py
python3 migrations/005_area_district_seeder.py

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
go run ./cmd        # Run server
go build -o app ./cmd  # Build binary
go test ./...       # Run tests
go vet ./...        # Static analysis
go fmt ./...        # Format code
go mod tidy         # Sync dependencies
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
| `GET` | `/v1/search/cafes` | Cafe discovery with polygon, radius, or global mode |
| `GET` | `/v1/cafe/:id` | Full detail for a single cafe |
| `GET` | `/v1/cafe/:id/review` | Review and ratings for a single cafe |

See [`docs/api-contracts.md`](docs/api-contracts.md) for full request/response schemas.

### Quick examples

```bash
# Typeahead
GET /v1/quicksearch?q=dreezel
GET /v1/quicksearch?q=dago&type=area

# Search cafes inside an area
GET /v1/search/cafes?query_id=dago&query_type=area&tag=wifi-friendly

# Radius search from coordinates, sorted by distance
GET /v1/search/cafes?query_coords=-6.9039,107.6186&radius_max=2000&sort=distance

# Cafe detail and review
GET /v1/cafe/accio-coffee
GET /v1/cafe/accio-coffee/review
```
