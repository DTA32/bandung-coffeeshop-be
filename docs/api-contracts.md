# BDGCafe API Contracts (v1)

Base URL: `http://<host>:<APP_PORT>` (default port: `8080`)
All `/v1` endpoints return JSON in the standard envelope below.

## Response Envelope

### Success
```json
{
  "success": true,
  "data": { ... }
}
```

### Error
```json
{
  "success": false,
  "error": "human readable message"
}
```

HTTP status codes used: `200`, `400` (validation), `404` (resource not found), `500` (server error).

## Localization

Content is bilingual. Clients select a locale with the `Accept-Language` header:
`en` (English) or `id` (Indonesian). Weighted lists such as
`en-US,en;q=0.9,id;q=0.8` are tolerated. When the header is absent or
unrecognised the API falls back to Indonesian (`id`). The header applies to every
`/v1` endpoint that returns human-readable text.

---

## Shared Enums

### Location type (`type`, `query_type`)
| Value | Meaning |
|-------|---------|
| `cafe` | A coffee shop |
| `poi` | A point of interest (landmark) |
| `area` | A neighbourhood / area polygon |
| `district` | A larger administrative district |

### Quicksearch type selector (`type` on `/v1/quicksearch`)
| Value | Meaning |
|-------|---------|
| `all` | Locations and filters (default when omitted) |
| `location` | All location types only |
| `filter` | SRP filters only |
| `cafe` / `poi` / `area` / `district` | Restrict to that single location type |

### Rating category type
`price-rank`, `vibe`, `noise`, `wifi`, `meals`, `atmosphere`, `parking`

(Data-driven via the `rating_type_label` table; the list above is the current seed.)

### Sort (`sort`)
`default` (default), `updated_at`, `distance`, `rating`, `price_range`

### Order (`order`)
`asc`, `desc`

---

## 1. `GET /v1/quicksearch`

Typeahead lookup over locations (cafes, POIs, areas, districts) and SRP filters,
using trigram similarity. Locations are returned first, then filters.

### Query params
| Name | Type | Required | Notes |
|------|------|----------|-------|
| `q` | string | yes | Search term. Queries shorter than 2 characters return an empty list (no error). |
| `type` | enum (quicksearch type selector) | no | Selects the sources to search. Defaults to `all`. |

### Success `200`
```json
{
  "success": true,
  "data": [
    {
      "id": "anjis-dago",
      "name": "Kopi Anjis Dago",
      "type": "cafe"
    },
    {
      "id": "quiet-noise",
      "name": "Quiet",
      "type": "filter",
      "slug": "quiet-noise"
    }
  ]
}
```

`data` is always an array (possibly empty).

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `type` | string | One of `cafe`, `poi`, `area`, `district`, or `filter`. |
| `slug` | string | Omitted (empty) for cafes. For a filter, the SRP filter slug. For area/POI/district, the canonical `/explore` splat path; empty when the ancestor chain is incomplete (frontend falls back to id). |

### Errors
| Status | `error` | When |
|--------|---------|------|
| 400 | `invalid type` | `type` is set but not one of the selector values |
| 500 | `search failed` | Unexpected server / DB error |

### Examples
```
GET /v1/quicksearch?q=anjis
GET /v1/quicksearch?q=dago&type=area
GET /v1/quicksearch?q=quiet&type=filter
```

---

## 2. `GET /v1/search/cafes`

Search and discover cafes. Supports three search modes derived from inputs:

- **Polygon**: when `query_id` resolves to an `area` or `district` — cafes inside that polygon.
- **Radius**: when `query_id` resolves to a `cafe` / `poi`, **or** when `query_coords` is supplied — cafes within `radius_max` meters of the focus point.
- **Global**: when neither focus nor coordinates are provided.

### Query params

| Name | Type | Required | Default | Notes |
|------|------|----------|---------|-------|
| `query_id` | string | conditional | — | Location ID to focus the search on. Must be paired with `query_type`. |
| `query_type` | enum (location type) | conditional | — | Type of `query_id`. Must be paired with `query_id`. |
| `query_coords` | string `"lat,lng"` | no | — | Free coordinates, e.g. `-6.9039,107.6186`. Cannot be combined with `query_id`. Lat ∈ [-90, 90], Lng ∈ [-180, 180]. |
| `radius_max` | int (meters) | no | `3000` (coords / `cafe` focus), `2000` (`poi` focus) | Positive integer. Only meaningful in radius mode. |
| `ratings` | int CSV | no | — | Comma-separated rating-bucket ids (one per category type). Two buckets of the same category type are rejected. |
| `tags` | string CSV | no | — | Comma-separated tag slugs, AND-combined. Unknown slugs are silently ignored. |
| `open_hour` | string | no | — | `now` (current time in WIB) or a `"HH:MM"` time. Filters to cafes open at that time. |
| `price_min` | int | no | — | `0`–`999999`. |
| `price_max` | int | no | — | `0`–`999999`. Must be ≥ `price_min`. |
| `is_featured` | bool | no | — | `true` / `false`. |
| `sort` | enum (sort) | no | `default` | `distance` requires either `query_coords` or a `query_type` of `cafe` / `poi`. |
| `order` | enum (order) | no | server default | `asc` or `desc`. |
| `page` | int | no | `1` | Must be positive. |
| `size` | int | no | `8` | Must be positive. Capped at `50`. |

Locale is taken from the `Accept-Language` header (affects names, descriptions, and price labels).

#### Validation rules
- `query_id` ↔ `query_type` must both be set or both omitted.
- `query_coords` cannot coexist with `query_id`.
- `price_min` cannot exceed `price_max`.
- `open_hour` must be `now` or a valid `HH:MM`.
- `ratings` may not include two buckets of the same category type.
- `sort=distance` requires a coordinate-based focus (either `query_coords`, or `query_type` ∈ {`cafe`, `poi`}).

### Success `200`

```json
{
  "success": true,
  "data": {
    "total": 42,
    "location_name": "Dago",
    "formatted_location_name": "in Dago",
    "search_description": "Leafy uphill neighbourhood with...",
    "locations": [
      { "id": "bandung-utara", "name": "Bandung Utara", "type": "district", "thumbnail": null },
      { "id": "dago", "name": "Dago", "type": "area", "thumbnail": null }
    ],
    "page": 1,
    "size": 8,
    "cafes": [
      {
        "id": "anjis-dago",
        "name": "Kopi Anjis Dago",
        "description": "Jl. Ir. H. Djuanda No.123, Dago",
        "coordinates": { "lat": -6.8839, "lng": 107.6132 },
        "thumbnail": "https://.../thumb.jpg",
        "area": "Dago",
        "price_range": "Rp. 25k - Rp. 60k",
        "distance": 320,
        "remark": "Great pour-over"
      }
    ]
  }
}
```

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `total` | int | Total matching cafes (across all pages). |
| `location_name` | string | Resolved focus name; empty string when no focus or when `query_coords` is used. |
| `formatted_location_name` | string | Human label: `"in <Area/District>"`, `"near <Cafe/POI>"`, `"near Selected Spot"` for raw coords, or empty. Localized. |
| `search_description` | string | Long-form blurb: tag description (only when filtering by a single tag and nothing else), focus description (for area/district/POI), else empty. |
| `locations` | array | Focus breadcrumb (ancestor chain, outermost first, including the focus). Empty for cafe / coordinate / global searches. |
| `page`, `size` | int | Echo of the (normalized) pagination request. |
| `cafes[].description` | string | Address / short description (may be empty). |
| `cafes[].coordinates` | object \| null | `null` if the cafe has no stored coordinates. |
| `cafes[].thumbnail` | string \| null | Image URL or null. |
| `cafes[].area` | string \| null | Area name the cafe belongs to. |
| `cafes[].price_range` | string \| null | Pre-formatted: `"Rp. 25k - Rp. 60k"`, `"start from Rp. 25k"`, `"up to Rp. 60k"`, or null. Localized. |
| `cafes[].distance` | int \| null | Meters from the focus point. Only populated when `query_coords` is provided; otherwise `null`. |
| `cafes[].remark` | string \| null | Editor's note for the cafe. |

### Errors

| Status | `error` | Trigger |
|--------|---------|---------|
| 400 | `invalid location type` | `query_type` not in enum |
| 400 | `query_type requires query_id` / `query_id requires query_type` | only one of the pair supplied |
| 400 | `query_coords cannot be combined with query_id` | both supplied |
| 400 | `invalid query_coords` | malformed or out-of-range coords |
| 400 | `invalid radius_max` / `invalid is_featured` / `invalid page` / `invalid size` | param failed to parse |
| 400 | `invalid ratings` / `invalid price_min` / `invalid price_max` | non-integer / out-of-range value |
| 400 | `price_min cannot exceed price_max` | inverted price range |
| 400 | `invalid open_hour` | not `now` and not a valid `HH:MM` |
| 400 | `duplicate rating category in filter` | two `ratings` buckets share a category type |
| 400 | `invalid sort` / `invalid order` | not in enum |
| 400 | `sort=distance requires query_coords` | distance sort without a coord focus |
| 404 | `focus location not found` | `query_id` does not resolve |
| 404 | `rating category not found` | a `ratings` id does not resolve |
| 500 | `search failed` | Unexpected server / DB error |

### Examples

Polygon search inside an area, filtered by tags:
```
GET /v1/search/cafes?query_id=dago&query_type=area&tags=wifi-friendly,quiet&page=1&size=10
```

Radius search around user coordinates, sorted by distance:
```
GET /v1/search/cafes?query_coords=-6.9039,107.6186&radius_max=2000&sort=distance
```

Global featured list:
```
GET /v1/search/cafes?is_featured=true&sort=rating&order=desc
```

Filter by rating buckets and price, open now:
```
GET /v1/search/cafes?ratings=4,9&price_min=20000&price_max=50000&open_hour=now
```

---

## 3. `GET /v1/cafe/:id`

Returns full detail for a single cafe by its location slug.

### Path params
| Name | Type | Notes |
|------|------|-------|
| `id` | string | Location slug (e.g. `accio-coffee`). Case-sensitive. |

### Success `200`

```json
{
  "success": true,
  "data": {
    "id": "accio-coffee",
    "name": "Accio Coffee",
    "description": "Jl. Batik Kumeli No.38, Sukaluyu, Bandung",
    "status": "active",
    "images": [
      { "url": "https://example.com/photo.jpg", "description": "Interior" }
    ],
    "instagram": "_accio.coffee",
    "open_hour": "08:00",
    "close_hour": "22:00",
    "gmaps_id": "ChIJrTLr-GyuEmsRBfy61i59si0",
    "locations": [
      { "id": "bandung-tengah", "name": "Bandung Tengah", "type": "district", "thumbnail": null },
      { "id": "sukaluyu", "name": "Sukaluyu", "type": "area", "thumbnail": null }
    ],
    "price": {
      "price_range_min": 18000,
      "price_range_max": 28000,
      "coffee_price_min": 20000,
      "coffee_price_max": 28000,
      "snack_price_min": 15000,
      "snack_price_max": 17000,
      "food_price_min": 25000,
      "food_price_max": 30000,
      "rank": {
        "type": 0,
        "label": "Bandung pricing - affordable for most"
      }
    }
  }
}
```

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Location slug, same as the `:id` path param. |
| `description` | string \| null | Address / description text. `null` if not set. |
| `status` | string | Location status: `active`, `closed`, or `deleted`. |
| `images` | array | Ordered by `display_order`. Empty array if none. `description` is the image's alt text. |
| `instagram` | string \| null | Instagram handle without `@`. |
| `open_hour` / `close_hour` | string \| null | 24-hour format `"HH:MM"`. `null` if not set. |
| `gmaps_id` | string \| null | The cafe's own Google Maps place ID. `null` if not set. |
| `locations` | array | Ancestor chain of the area/neighbourhood the cafe sits in, outermost first (district, then area). Empty array if coordinates are unset or no matching area is found. |
| `price.rank` | object \| null | Derived from the median of `price_range_min` and `price_range_max` matched against `price-rank` rating categories. `null` if price data is missing. `type` is the 0-based ordinal of the matched bucket (cheapest = 0). `label` is the bucket's localized description. |

### Errors
| Status | `error` | When |
|--------|---------|------|
| 404 | `cafe not found` | No active cafe with the given `id` exists |
| 500 | (server / DB error message) | Unexpected server / DB error |

### Example
```
GET /v1/cafe/accio-coffee
```

---

## 4. `GET /v1/cafe/:id/review`

Returns the review for a single cafe. If the cafe exists but has no review yet, a zeroed response is returned (not an error).

### Path params
| Name | Type | Notes |
|------|------|-------|
| `id` | string | Location slug. Same as `/v1/cafe/:id`. |

### Success `200`

When a review exists:
```json
{
  "success": true,
  "data": {
    "is_subjective": true,
    "overall_score": 4.5,
    "wfc_score": 4.2,
    "tags": [
      { "name": "WFC Friendly", "slug": "wfc-friendly" },
      { "name": "Reading", "slug": "reading" }
    ],
    "content": "One of the coffee shops that feels like a second home...",
    "visited_at": "2024-11-12",
    "updated_at": "2026-05-03 09:55:59.246561",
    "ratings": {
      "vibe": {
        "display_name": "Vibe",
        "range": [
          { "name": "Hangout",     "slug": null,            "description": "More lively and suitable for hanging out", "lower_bound": 0,    "upper_bound": 1.67 },
          { "name": "All-rounder", "slug": null,            "description": "Good for all occasions",                   "lower_bound": 1.68, "upper_bound": 3.33 },
          { "name": "Comfy",       "slug": "comfortable-vibe", "description": "Comfortable and cozy environment",       "lower_bound": 3.34, "upper_bound": 5 }
        ],
        "score": 3.8,
        "description": "Comfortable, perfect for working and small chit-chats"
      }
    }
  }
}
```

When no review has been written yet:
```json
{
  "success": true,
  "data": {
    "is_subjective": false,
    "overall_score": null,
    "wfc_score": null,
    "tags": [],
    "content": null,
    "visited_at": null,
    "updated_at": "",
    "ratings": {}
  }
}
```

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `is_subjective` | bool | Whether the review reflects personal bias (e.g. a regular's perspective). |
| `overall_score` | number \| null | 0–5 overall score. `null` if not set. |
| `wfc_score` | number \| null | 0–5 work-from-cafe suitability score. `null` if not set. |
| `tags` | array | Visible tags for this cafe. `slug` is `null` for display-only tags. |
| `content` | string \| null | Full review text. `null` if not written. |
| `visited_at` | string \| null | Date of most recent visit, `"YYYY-MM-DD"`. `null` if unrecorded. |
| `updated_at` | string | Timestamp of last review update. Empty string when no review exists. |
| `ratings` | object | Map of rating category type → entry. Only categories the cafe has been rated in are present. |
| `ratings[type].display_name` | string | Localized label for the category type. |
| `ratings[type].score` | number | The cafe's score for this category (0–5). |
| `ratings[type].description` | string | Per-cafe description override for this rating; empty string if none. |
| `ratings[type].range` | array | All defined buckets for that category, ordered by `lower_bound`. |
| `ratings[type].range[].name` | string | Bucket label (e.g. `"Comfy"`). |
| `ratings[type].range[].slug` | string \| null | SRP slug `"<bucket-slug>-<type>"`, but only on the bucket the cafe's score falls into and only when that bucket is SRP-eligible; `null` otherwise. |
| `ratings[type].range[].description` | string | Bucket short description. |
| `ratings[type].range[].lower_bound` / `upper_bound` | number | Bucket score range. |

### Errors
| Status | `error` | When |
|--------|---------|------|
| 404 | `cafe not found` | No active cafe with the given `id` exists |
| 500 | `failed to fetch review` | Unexpected server / DB error |

### Example
```
GET /v1/cafe/accio-coffee/review
```

---

## 5. `GET /v1/location`

Lists all districts, each with its flat descendants (areas + POIs) and images.
This is the no-id fallback for the explore landing page.

### Success `200`

```json
{
  "success": true,
  "data": [
    {
      "id": "bandung-utara",
      "name": "Bandung Utara",
      "description": "",
      "type": "district",
      "ancestors": [],
      "descendants": [
        { "id": "dago", "name": "Dago", "type": "area", "thumbnail": "https://.../dago.jpg" }
      ],
      "images": [
        { "url": "https://.../bandung-utara.jpg", "description": "Skyline" }
      ],
      "show_welcome_text": false,
      "show_map": false,
      "polygon": null
    }
  ]
}
```

District summaries omit `description`/`show_welcome_text`/`show_map`/`polygon`
detail (they serialize as their zero values); fetch a single location for the
full payload.

### Errors
| Status | `error` | When |
|--------|---------|------|
| 500 | `failed to list districts` | Unexpected server / DB error |

### Example
```
GET /v1/location
```

---

## 6. `GET /v1/location/:id`

Returns full detail for a single non-cafe location (area, POI, or district).
Cafes are rejected — use `/v1/cafe/:id` instead.

### Path params
| Name | Type | Notes |
|------|------|-------|
| `id` | string | Location slug (e.g. `dago`). |

### Success `200`

```json
{
  "success": true,
  "data": {
    "id": "dago",
    "name": "Dago",
    "description": "Leafy uphill neighbourhood with...",
    "type": "area",
    "ancestors": [
      { "id": "bandung-utara", "name": "Bandung Utara", "type": "district", "thumbnail": null }
    ],
    "descendants": [
      { "id": "dago-pakar", "name": "Dago Pakar", "type": "poi", "thumbnail": null }
    ],
    "images": [
      { "url": "https://.../dago.jpg", "description": "Street view" }
    ],
    "show_welcome_text": true,
    "show_map": false,
    "polygon": { "type": "Polygon", "coordinates": [ ... ] }
  }
}
```

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `ancestors` | array | Ancestor chain, outermost first. Empty for districts. |
| `descendants` | array | Direct children (areas / POIs). |
| `images` | array | Location images (`url`, `description`). |
| `show_welcome_text` | bool | `true` only for `area` locations. |
| `show_map` | bool | `true` for districts, and for non-POI locations that have no images. |
| `polygon` | object \| null | GeoJSON geometry for the location boundary, or `null` if none. |

### Errors
| Status | `error` | When |
|--------|---------|------|
| 400 | `location is a cafe; use the cafe endpoint` | `id` resolves to a cafe |
| 404 | `location not found` | No location with the given `id` exists |
| 500 | `failed to fetch location` | Unexpected server / DB error |

### Example
```
GET /v1/location/dago
```

---

## 7. `GET /v1/filters`

Returns the option lists that power the explore filter modal and SRP: selectable
tags, rating categories (grouped by type, each with its buckets), and price
tiers. The `price-rank` rating category is surfaced as `price_tiers` rather than
a rating group.

### Query params
| Name | Type | Required | Default | Notes |
|------|------|----------|---------|-------|
| `enrich_content` | bool | no | `false` | When `true`, adds the heavier `description` / `long_description` blurbs (SRP); the filter modal omits it for a lighter payload. |

### Success `200`

```json
{
  "success": true,
  "data": {
    "tags": [
      { "name": "WFC Friendly", "slug": "wfc-friendly" }
    ],
    "rating_categories": [
      {
        "type": "vibe",
        "display_name": "Vibe",
        "options": [
          { "id": 4, "slug": "hangout-vibe",     "name": "Hangout",     "description": "More lively...",     "lower_bound": 0,    "upper_bound": 1.67 },
          { "id": 5, "slug": "all-rounder-vibe", "name": "All-rounder", "description": "Good for all...",     "lower_bound": 1.68, "upper_bound": 3.33 },
          { "id": 6, "slug": "comfortable-vibe", "name": "Comfy",       "description": "Comfortable...",      "lower_bound": 3.34, "upper_bound": 5 }
        ]
      }
    ],
    "price_tiers": [
      { "label": "Bandung", "slug": "bandung-price-rank", "min": 0,     "max": 25000 },
      { "label": "Riau",    "slug": "",                   "min": 25001, "max": 45000 },
      { "label": "Jakarta", "slug": "",                   "min": 45001, "max": null }
    ]
  }
}
```

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `tags[].slug` | string | Tag slug used as the `tags` filter value. |
| `tags[].description` | string | Only present when `enrich_content=true`. |
| `rating_categories[].type` | string | Rating category type enum value. |
| `rating_categories[].display_name` | string | Localized label for the type. |
| `rating_categories[].options[].id` | int | Bucket id; passed to the `ratings` search filter. |
| `rating_categories[].options[].slug` | string | SRP slug `"<slug>-<type>"`, or `""` when not SRP-eligible. |
| `rating_categories[].options[].long_description` | string | Only present when `enrich_content=true`. |
| `price_tiers[].slug` | string | SRP slug `"<slug>-price-rank"`, or `""` when not SRP-eligible. |
| `price_tiers[].max` | int \| null | Upper bound (Rupiah); `null` on the open-ended top tier. |
| `price_tiers[].long_description` | string | Only present when `enrich_content=true`. |

### Errors
| Status | `error` | When |
|--------|---------|------|
| 500 | `failed to fetch filters` | Unexpected server / DB error |

### Example
```
GET /v1/filters
GET /v1/filters?enrich_content=true
```
