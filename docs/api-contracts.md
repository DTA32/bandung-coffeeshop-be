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

---

## Shared Enums

### Location type (`type`, `query_type`)
| Value | Meaning |
|-------|---------|
| `cafe` | A coffee shop |
| `poi` | A point of interest (landmark) |
| `area` | A neighbourhood / area polygon |
| `district` | A larger administrative district |

### Rating category type (`rating_category_type`)
`price-rank`, `vibe`, `noise`, `wifi`, `meals`, `atmosphere`, `parking`

### Sort (`sort`)
`default` (default), `updated_at`, `distance`, `rating`, `price_range`

### Order (`order`)
`asc`, `desc`

---

## 1. `GET /v1/quicksearch`

Typeahead lookup over locations (cafes, POIs, areas, districts) using trigram similarity.

### Query params
| Name | Type | Required | Notes |
|------|------|----------|-------|
| `q` | string | yes | Search term. Queries shorter than 2 characters return an empty list (no error). |
| `type` | enum (location type) | no | When set, restricts results to that type. |

### Success `200`
```json
{
  "success": true,
  "data": [
    {
      "id": "loc_abc123",
      "name": "Kopi Anjis",
      "type": "cafe"
    }
  ]
}
```

`data` is always an array (possibly empty).

### Errors
| Status | `error` | When |
|--------|---------|------|
| 400 | `invalid location type` | `type` is set but not one of the enum values |
| 500 | `search failed` | Unexpected server / DB error |

### Examples
```
GET /v1/quicksearch?q=anjis
GET /v1/quicksearch?q=dago&type=area
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
| `radius_max` | int (meters) | no | `5000` when in radius mode | Positive integer. Only meaningful in radius mode. |
| `rating_category_type` | enum (rating category) | conditional | — | Required when `rating_category_id` is set. |
| `rating_category_id` | string (slug) | no | — | Slug of a rating bucket inside `rating_category_type`. |
| `tag` | string (slug) | no | — | Filter cafes that carry this tag. |
| `is_featured` | bool | no | — | `true` / `false`. |
| `sort` | enum (sort) | no | `default` | `distance` requires either `query_coords` or a `query_type` of `cafe` / `poi`. |
| `order` | enum (order) | no | server default | `asc` or `desc`. |
| `page` | int | no | `1` | Must be positive. |
| `size` | int | no | `8` | Must be positive. Capped at `50`. |

#### Validation rules
- `query_id` ↔ `query_type` must both be set or both omitted.
- `query_coords` cannot coexist with `query_id`.
- `rating_category_id` requires `rating_category_type`.
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
    "page": 1,
    "size": 8,
    "cafes": [
      {
        "id": "cafe_xyz",
        "name": "Kopi Anjis Dago",
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
| `formatted_location_name` | string | Human label: `"in <Area/District>"`, `"near <Cafe/POI>"`, `"near Selected Spot"` for raw coords, or empty. |
| `search_description` | string | Long-form blurb: tag description (when filtering by tag only), focus description (for area/district/POI), else empty. |
| `page`, `size` | int | Echo of the (normalized) pagination request. |
| `cafes[].coordinates` | object \| null | `null` if the cafe has no stored coordinates. |
| `cafes[].thumbnail` | string \| null | Image URL or null. |
| `cafes[].area` | string \| null | Area name the cafe belongs to. |
| `cafes[].price_range` | string \| null | Pre-formatted: `"Rp. 25k - Rp. 60k"`, `"start from Rp. 25k"`, `"up to Rp. 60k"`, or null. |
| `cafes[].distance` | int \| null | Meters from the focus point. Only populated when `query_coords` is provided; otherwise `null`. |
| `cafes[].remark` | string \| null | Editor's note for the cafe. |

### Errors

| Status | `error` | Trigger |
|--------|---------|---------|
| 400 | `invalid location type` | `query_type` not in enum |
| 400 | `invalid sort` | `sort` not in enum |
| 400 | `invalid order` | `order` not in enum |
| 400 | `invalid query_coords` | malformed or out-of-range coords |
| 400 | `query_coords cannot be combined with query_id` | both supplied |
| 400 | `query_type requires query_id` | `query_type` set without `query_id` |
| 400 | `query_id requires query_type` | `query_id` set without `query_type` |
| 400 | `sort=distance requires query_coords` | distance sort without a coord focus |
| 400 | `invalid rating_category_type` | unknown rating category |
| 400 | `rating_category_id requires rating_category_type` | slug without type |
| 400 | `invalid radius_max` / `invalid is_featured` / `invalid page` / `invalid size` | param failed to parse |
| 404 | `focus location not found` | `query_id` does not resolve |
| 404 | `rating category not found` | unknown `rating_category_id` slug |
| 404 | `tag not found` | unknown `tag` slug |
| 500 | `search failed` | Unexpected server / DB error |

> Exact 404 messages come from the repository layer; treat the listed strings as representative.

### Examples

Polygon search inside an area, filtered by tag:
```
GET /v1/search/cafes?query_id=loc_dago&query_type=area&tag=wifi-friendly&page=1&size=10
```

Radius search around user coordinates, sorted by distance:
```
GET /v1/search/cafes?query_coords=-6.9039,107.6186&radius_max=2000&sort=distance
```

Global featured list:
```
GET /v1/search/cafes?is_featured=true&sort=rating&order=desc
```

Filter by rating bucket:
```
GET /v1/search/cafes?rating_category_type=vibe&rating_category_id=cozy
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
      { "url": "https://example.com/photo.jpg", "alt": "Interior" }
    ],
    "instagram": "_accio.coffee",
    "open_hour": "08:00",
    "close_hour": "22:00",
    "gmaps_id": "ChIJrTLr-GyuEmsRBfy61i59si0",
    "location": {
      "id": "sukaluyu",
      "name": "Sukaluyu"
    },
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
| `images` | array | Ordered by `display_order`. Empty array if none. `alt` is the image description. |
| `instagram` | string \| null | Instagram handle without `@`. |
| `open_hour` / `close_hour` | string \| null | 24-hour format `"HH:MM"`. `null` if not set. |
| `gmaps_id` | string \| null | The cafe's own Google Maps place ID. `null` if not set. |
| `location` | object \| null | The area/neighbourhood the cafe is geographically inside. `null` if coordinates are unset or no matching area is found. |
| `price.rank` | object \| null | Derived from the median of `price_range_min` and `price_range_max` matched against `price-rank` rating categories. `null` if price data is missing. `type` is the 0-based ordinal of the matched bucket (cheapest = 0). `label` is the bucket's long description. |

### Errors
| Status | `error` | When |
|--------|---------|------|
| 404 | `cafe not found` | No active cafe with the given `id` exists |
| 500 | `failed to fetch cafe` | Unexpected server / DB error |

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
        "range": [
          { "name": "Hangout",    "description": "Better for casual meetups", "lower_bound": 0,    "upper_bound": 1.67 },
          { "name": "All-rounder","description": "Works for both hangout and work", "lower_bound": 1.67, "upper_bound": 3.33 },
          { "name": "Comfy",      "description": "Great for focused, extended work", "lower_bound": 3.33, "upper_bound": 5 }
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
| `ratings[type].range` | array | All defined buckets for that category, ordered by `lower_bound`. |
| `ratings[type].score` | number | The cafe's score for this category (0–5). |
| `ratings[type].description` | string | Per-cafe description override for this rating; empty string if none. |
| `ratings[type].range[].name` | string | Bucket label (e.g. `"Comfy"`). |
| `ratings[type].range[].description` | string | Bucket short description. |

#### Rating category types
`price-rank`, `vibe`, `noise`, `wifi`, `meals`, `atmosphere`, `parking`

### Errors
| Status | `error` | When |
|--------|---------|------|
| 404 | `cafe not found` | No active cafe with the given `id` exists |
| 500 | `failed to fetch review` | Unexpected server / DB error |

### Example
```
GET /v1/cafe/accio-coffee/review
```
