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

HTTP status codes used: `200`, `201` (created), `204` (no content), `302` (auth redirects), `400` (validation), `401` (not signed in), `403` (disallowed origin), `404` (resource not found), `409` (conflict), `500` (server error), `503` (auth not configured / unavailable).

## Credentials & CORS

CORS allows only the origins in `ALLOWED_ORIGINS` and allows credentials, so
browser clients must send `credentials: "include"` for the session cookie to
travel. `POST` / `PUT` / `DELETE` requests carrying an `Origin` header that is
not allowlisted are rejected with `403`. See [Auth](#9-auth) for the session
cookie.

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

`default` ordering:
- radius mode: featured first, then `distance` ↑, then cafes with at least one image, then `updated_at` ↓
- otherwise: featured first, then cafes with at least one image, then `updated_at` ↓, then `distance` ↑

Ties are broken by `id` ↑.

`rating` sorts by the cafe's average `overall_score` over its [visible reviews](#visible-reviews); cafes without one sort last.

### Visible reviews
A review is publicly visible when its `status` is `approved` and it is either a
partnership review (no user account) or written by a user whose account is
still active. Every public review read and aggregate (review list, `/review`
aggregates, `ratings` search filter, `sort=rating`) counts only visible reviews.

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
| `ratings` | int CSV | no | — | Comma-separated rating-bucket ids (one per category type), AND-combined. A cafe matches a bucket when its average score in that category over its [visible reviews](#visible-reviews) (rounded to 2 decimals) lies within the bucket bounds. Two buckets of the same category type are rejected. |
| `tags` | string CSV | no | — | Comma-separated tag slugs, AND-combined. Unknown slugs are silently ignored. |
| `open_hour` | string | no | — | `now` (current time in Jakarta, WIB) or a `"HH:MM"` time, evaluated on today's Jakarta date. Filters to cafes open at that time per their resolved [opening hours](#opening-hours): today's ranges (same-day, or an overnight range from its open time), plus the early-hours spill of yesterday's overnight ranges (also on a day that is otherwise closed). Unknown hours never match. |
| `price_min` | int | no | — | `0`–`999999`. |
| `price_max` | int | no | — | `0`–`999999`. Must be ≥ `price_min`. |
| `is_featured` | bool | no | — | `true` / `false`. |
| `status` | enum (`active`, `closed`) | no | `active` | Cafe lifecycle status to list. `closed` covers cafes that closed or relocated (backs the `/closed-cafes` archive page). `deleted` cafes are never returned. |
| `weather` | string CSV | no | — | Either `current` alone (Bandung's weather right now, cached ≤ 30 min) or any of `clear`, `cloudy`, `rain`, OR-combined. Matches cafes whose admin-defined `weather` list shares at least one value. If `current` can't be resolved (no provider key, provider and caches all unavailable), the weather filter is skipped and `weather` in the response is `null`. |
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
- `status` must be `active` or `closed`.
- `ratings` may not include two buckets of the same category type.
- `sort=distance` requires a coordinate-based focus (either `query_coords`, or `query_type` ∈ {`cafe`, `poi`}).
- `weather` values must be `clear`, `cloudy`, `rain`, or a lone `current`.

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
    "weather": { "condition": "rain", "temp_c": 21.4, "observed_at": "2026-09-28T10:30:00+07:00" },
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
| `location_name` | string | Resolved focus name. With `query_coords`, the name of the area containing the point, falling back to its district; empty string when the point is outside every area/district or when there is no focus. |
| `formatted_location_name` | string | Human label: `"in <Area/District>"`, `"near <Cafe/POI>"`, `"near Selected Spot"` for raw coords, or empty. Localized. |
| `search_description` | string | Long-form blurb: tag description (only when filtering by a single tag and nothing else), focus description (for area/district/POI), else empty. |
| `locations` | array | Focus breadcrumb (ancestor chain, outermost first, including the focus). Empty for cafe / coordinate / global searches. |
| `page`, `size` | int | Echo of the (normalized) pagination request. |
| `weather` | object \| null | Only with `weather=current`: the reading it resolved to — `condition` (`clear` / `cloudy` / `rain`), `temp_c`, and `observed_at` (provider's last update, RFC 3339 in WIB). `null` for every other request, or when no reading was available. |
| `cafes[].description` | string | Address / short description (may be empty). |
| `cafes[].coordinates` | object \| null | `null` if the cafe has no stored coordinates. |
| `cafes[].thumbnail` | string \| null | Image URL or null. |
| `cafes[].area` | string \| null | Area name the cafe belongs to. |
| `cafes[].price_range` | string \| null | Pre-formatted: `"Rp. 25k - Rp. 60k"`, `"start from Rp. 25k"`, `"up to Rp. 60k"`, or null. Localized. |
| `cafes[].distance` | int \| null | Meters from the focus point. Only populated when `query_coords` is provided; otherwise `null`. |
| `cafes[].remark` | string \| null | Editor's note for the cafe. When `tags` is sent, excludes the selected tags and is `null` if none remain. |

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
| 400 | `invalid status` | `status` not `active` / `closed` |
| 400 | `duplicate rating category in filter` | two `ratings` buckets share a category type |
| 400 | `invalid weather` | a `weather` value outside `clear` / `cloudy` / `rain` / `current` |
| 400 | `weather=current cannot be combined with other weather values` | `current` sent alongside another value |
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

Closed / relocated cafe archive, second page:
```
GET /v1/search/cafes?status=closed&page=2&size=12
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

Match Bandung's current weather, or pick conditions manually:
```
GET /v1/search/cafes?weather=current
GET /v1/search/cafes?weather=cloudy,rain
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
    "about": "One of the coffee shops that feels like a second home...",
    "status": "active",
    "images": [
      { "url": "https://example.com/photo.jpg", "description": "Interior" }
    ],
    "instagram": "_accio.coffee",
    "menu": "https://example.com/menu.pdf",
    "hours": [
      { "date": "2026-10-24", "day": "sat", "closed": false, "special": false,
        "ranges": [{ "open": "08:00", "close": "14:00" }, { "open": "17:00", "close": "23:00" }] },
      { "date": "2026-10-25", "day": "sun", "closed": true, "special": true, "ranges": [] },
      { "date": "2026-10-26", "day": "mon", "closed": false, "special": false,
        "ranges": [{ "open": "08:00", "close": "22:00" }] },
      { "date": "2026-10-27", "day": "tue", "closed": false, "special": false,
        "ranges": [{ "open": "08:00", "close": "22:00" }] },
      { "date": "2026-10-28", "day": "wed", "closed": false, "special": false,
        "ranges": [{ "open": "08:00", "close": "22:00" }] },
      { "date": "2026-10-29", "day": "thu", "closed": false, "special": false,
        "ranges": [{ "open": "08:00", "close": "22:00" }] },
      { "date": "2026-10-30", "day": "fri", "closed": false, "special": false,
        "ranges": [{ "open": "08:00", "close": "02:00" }] }
    ],
    "regular_hours": [
      { "day": "mon", "closed": false, "ranges": [{ "open": "08:00", "close": "22:00" }] },
      { "day": "tue", "closed": false, "ranges": [{ "open": "08:00", "close": "22:00" }] },
      { "day": "wed", "closed": false, "ranges": [{ "open": "08:00", "close": "22:00" }] },
      { "day": "thu", "closed": false, "ranges": [{ "open": "08:00", "close": "22:00" }] },
      { "day": "fri", "closed": false, "ranges": [{ "open": "08:00", "close": "02:00" }] },
      { "day": "sat", "closed": false,
        "ranges": [{ "open": "08:00", "close": "14:00" }, { "open": "17:00", "close": "23:00" }] },
      { "day": "sun", "closed": false, "ranges": [{ "open": "08:00", "close": "22:00" }] }
    ],
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
    },
    "updated_at": "2026-05-03T09:55:59Z"
  }
}
```

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Location slug, same as the `:id` path param. |
| `description` | string \| null | Address / description text. `null` if not set. |
| `about` | string \| null | Editorial "about" blurb for the cafe (localized; falls back to English). `null` when empty. |
| `status` | string | Location status: `active`, `closed`, or `deleted`. |
| `images` | array | Ordered by `display_order`. Empty array if none. `description` is the image's alt text. |
| `instagram` | string \| null | Instagram handle without `@`. |
| `menu` | string \| null | Either an `http(s)` URL, or a `#image-N` gallery hash (`N` is 1-based, so `#image-1` is the first entry of `images`) meaning "open gallery image N". Passed through untouched; `null` if not set. |
| `hours` | array | The resolved schedule for the 7 days starting today (Jakarta), in date order; see [Opening hours](#opening-hours). `date` is `"YYYY-MM-DD"`, `day` is `mon`…`sun`. `closed` = closed that day (`ranges` is empty). `special` = resolved from a dated row (holiday / special event). Unknown hours: `closed = false`, `ranges = []`. |
| `regular_hours` | array | The regular week from undated rows only (dated rows ignored), always 7 entries `mon` → `sun`, same `closed` / `ranges` semantics as `hours` (no `date` / `special`). Backs the SEO JSON-LD and meta description. |
| `ranges[]` | object | `{ "open": "HH:MM", "close": "HH:MM" }`, 24-hour, sorted by `open`; several for split hours. `close` < `open` is an overnight range (closes the next day). Open 24 hours is `00:00`–`23:59`. |
| `gmaps_id` | string \| null | The cafe's own Google Maps place ID. `null` if not set. |
| `locations` | array | Ancestor chain of the area/neighbourhood the cafe sits in, outermost first (district, then area). Empty array if coordinates are unset or no matching area is found. |
| `updated_at` | string | When the cafe was last updated (backs "Last updated"), `"YYYY-MM-DDTHH:MM:SSZ"`. Written by mission control on cafe edits; independent of reviews. |
| `price.rank` | object \| null | Derived from the median of `price_range_min` and `price_range_max` matched against `price-rank` rating categories. `null` if price data is missing. `type` is the 0-based ordinal of the matched bucket (cheapest = 0). `label` is the bucket's localized description. |

#### Opening hours
Resolved from the cafe's `cafe_hour` rows with the rules in `CLAUDE.md` ("Opening hours"). For date D, most specific first: (a) dated rows covering D, a row for D's weekday beating a null-day row; (b) undated rows for D's weekday; (c) undated null-day rows. Only the first tier that has rows is used. "Today" is always Asia/Jakarta.

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

Aggregates over the cafe's [visible reviews](#visible-reviews), plus the cafe's
tags. A cafe with no visible review returns zero counts, `null` scores and
empty `ratings` (not an error). The individual reviews are served by
[`GET /v1/cafe/:id/reviews`](#101-get-v1cafeidreviews).

### Path params
| Name | Type | Notes |
|------|------|-------|
| `id` | string | Location slug. Same as `/v1/cafe/:id`. |

### Success `200`

```json
{
  "success": true,
  "data": {
    "overall_score": 4.25,
    "total_review_count": 3,
    "overall_review_count": 2,
    "wfc_score": 4.2,
    "wfc_review_count": 1,
    "tags": [
      { "name": "WFC Friendly", "slug": "wfc-friendly" },
      { "name": "Reading", "slug": "reading" }
    ],
    "ratings": {
      "vibe": {
        "display_name": "Vibe",
        "range": [
          { "name": "Hangout",     "slug": null,               "description": "More lively and suitable for hanging out", "lower_bound": 0,    "upper_bound": 1.67 },
          { "name": "All-rounder", "slug": null,               "description": "Good for all occasions",                   "lower_bound": 1.68, "upper_bound": 3.33 },
          { "name": "Comfy",       "slug": "comfortable-vibe", "description": "Comfortable and cozy environment",         "lower_bound": 3.34, "upper_bound": 5 }
        ],
        "score": 3.8,
        "description": "Comfortable, perfect for working and small chit-chats"
      }
    }
  }
}
```

With no visible review:
```json
{
  "success": true,
  "data": {
    "overall_score": null,
    "total_review_count": 0,
    "overall_review_count": 0,
    "wfc_score": null,
    "wfc_review_count": 0,
    "tags": [],
    "ratings": {}
  }
}
```

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `overall_score` | number \| null | Average `overall_score` of visible reviews, rounded to 2 decimals. `null` when none has one. |
| `total_review_count` | int | Number of visible reviews. |
| `overall_review_count` | int | Visible reviews that carry an `overall_score`. |
| `wfc_score` | number \| null | Average work-from-cafe score, rounded to 2 decimals. `null` when none has one. |
| `wfc_review_count` | int | Visible reviews that carry a `wfc_score`. |
| `tags` | array | Visible tags for this cafe (independent of reviews). `slug` is `null` for display-only tags. |
| `ratings` | object | Map of rating category type → entry. A category is present when at least one visible review rated it, whether or not any review has text. |
| `ratings[type].display_name` | string | Localized label for the category type. |
| `ratings[type].score` | number | Average score in this category over visible reviews, rounded to 2 decimals. |
| `ratings[type].description` | string | Per-cafe description override, taken from the top visible review that has one (pinned first, then most recently updated); empty string if none. |
| `ratings[type].range` | array | All defined buckets for that category, ordered by `lower_bound`. |
| `ratings[type].range[].name` | string | Bucket label (e.g. `"Comfy"`). |
| `ratings[type].range[].slug` | string \| null | SRP slug `"<bucket-slug>-<type>"`, only on the bucket containing the average `score` and only when that bucket is SRP-eligible; `null` otherwise. |
| `ratings[type].range[].description` | string | Bucket short description. |
| `ratings[type].range[].lower_bound` / `upper_bound` | number | Bucket score range. |

`is_subjective`, `content`, `visited_at` and `updated_at` were removed (BCF-30):
review text now lives on the individual reviews, the cafe blurb on
`/v1/cafe/:id` `about`, and "last updated" on `/v1/cafe/:id` `updated_at`.

### Errors
| Status | `error` | When |
|--------|---------|------|
| 404 | `cafe not found` | No cafe with the given `id` exists |
| 500 | `failed to fetch review` | Unexpected server / DB error |

### Example
```
GET /v1/cafe/accio-coffee/review
```

---

## 5. `GET /v1/cafe/:id/feed`

Returns the cafe's "latest updates" (entries created in mission control, usually from an Instagram post). Content is single-language, so `Accept-Language` is ignored. Only entries that are `visible` and not expired (`expires_at` is null or in the future) are returned, newest first (`published_at` desc, then `id` desc), capped at 10. Not cached.

### Path params
| Name | Type | Notes |
|------|------|-------|
| `id` | string | Location slug (e.g. `accio-coffee`). Case-sensitive. |

### Success `200`

```json
{
  "success": true,
  "data": [
    {
      "id": 12,
      "content": "New seasonal menu is out!",
      "images": ["https://image.bdgcafe.com/feed/12-1.jpg"],
      "author": "Admin",
      "link": "https://www.instagram.com/p/XXXXXXXX/",
      "published_at": "2026-10-12T09:30:00Z"
    }
  ]
}
```

A known cafe with no current entries returns `"data": []`.

#### Field semantics
| Field | Type | Notes |
|-------|------|-------|
| `id` | number | Feed entry ID. |
| `content` | string | Entry text. |
| `images` | string[] | Full image URLs. Empty array if none (never `null`). |
| `author` | string | Author label (defaults to `Admin`). |
| `link` | string \| null | Source link (e.g. the Instagram permalink). |
| `published_at` | string | UTC timestamp, `"YYYY-MM-DDTHH:MM:SSZ"` (mission control stores feed times in UTC). |

### Errors
| Status | `error` | When |
|--------|---------|------|
| 404 | `cafe not found` | No active cafe with the given `id` exists |
| 500 | `failed to fetch feed` | Unexpected server / DB error |

### Example
```
GET /v1/cafe/accio-coffee/feed
```

---

## 6. `GET /v1/location`

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

## 7. `GET /v1/location/:id`

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

## 8. `GET /v1/filters`

Returns the option lists that power the explore filter modal and SRP: selectable
tags, rating categories (grouped by type, each with its buckets), price
tiers, and weather conditions. The `price-rank` rating category is surfaced as `price_tiers` rather than
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
    ],
    "weather": [
      { "slug": "clear",  "name": "Clear" },
      { "slug": "cloudy", "name": "Cloudy" },
      { "slug": "rain",   "name": "Rain" }
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
| `weather[].slug` | string | Value for the search `weather` filter. Fixed set; the `current` selector is not listed (clients render it themselves). |
| `weather[].name` | string | Localized label. |

### Errors
| Status | `error` | When |
|--------|---------|------|
| 500 | `failed to fetch filters` | Unexpected server / DB error |

### Example
```
GET /v1/filters
GET /v1/filters?enrich_content=true
```

---

## 9. `GET /v1/cafe/random`

Returns one active cafe picked uniformly at random — powers the homepage
"Surprise me" button. Repeats are possible across calls. Responds with
`Cache-Control: no-store`.

### Success `200`

```json
{
  "success": true,
  "data": { "id": "anjis-dago", "name": "Kopi Anjis Dago" }
}
```

`id` is the cafe's location slug, usable with `GET /v1/cafe/:id`.

### Errors
| Status | `error` | When |
|--------|---------|------|
| 404 | `cafe not found` | No active cafe exists |
| 500 | `failed to pick a cafe` | Unexpected server / DB error |

---

## 9. Auth

Google sign-in (BCF-25). Sessions are opaque random tokens in an httpOnly
cookie; the server stores only their SHA-256 in Redis. Accounts live in the
`users` table, keyed by `(provider, provider_sub)`.

In production the API sits behind the `/api` prefix (`https://bdgcafe.com/api/v1/...`),
so all cookies are first-party.

### Cookies

| Cookie | Set by | Value | Attributes |
|--------|--------|-------|------------|
| `bdg_oauth` | `GET /v1/auth/:provider` | base64url JSON `{s: state, v: PKCE verifier, r: redirect path}` | `HttpOnly; SameSite=Lax; Path=/; Max-Age=600`; `Secure` unless `COOKIE_SECURE=false`. Cleared by the callback. |
| `bdg_session` | `GET /v1/auth/:provider/callback` on success | opaque token (43 chars, base64url of 32 random bytes) | `HttpOnly; SameSite=Lax; Path=/; Max-Age=2592000` (30 days); `Secure` unless `COOKIE_SECURE=false`. Cleared by logout. |

Sessions last a fixed 30 days from sign-in; activity does not extend them.
The session is attached to every `/v1` request that carries `bdg_session`;
an unknown/expired token, a deactivated user, or a Redis outage all degrade
to anonymous (no error), and a deactivated user's session is revoked.

### Availability

Auth needs `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `PUBLIC_BASE_URL` and
Redis. When any is missing (or Redis is unreachable), `GET /v1/auth/google`
and the callback answer `503` `auth unavailable`; `GET /v1/me` answers `401`;
the rest of the API is unaffected.

### 9.1 `GET /v1/auth/:provider`

Starts sign-in. Only `provider=google` exists; anything else is `404`
`provider not found`. Navigate the browser here (full page navigation, not
`fetch`).

| Query param | Notes |
|-------------|-------|
| `redirect` | Optional path to land on afterwards, e.g. `/en/cafe/accio-coffee?tab=review`. Must start with a single `/` (no `//`, no backslashes, no control characters); otherwise `/` is used. |

**`302`** to Google's consent screen (scopes `openid email profile`, PKCE S256)
and sets `bdg_oauth`. **`503`** `auth unavailable` when auth is not configured
or Redis is down.

### 9.2 `GET /v1/auth/:provider/callback`

Google's redirect URI (`${PUBLIC_BASE_URL}/api/v1/auth/google/callback`, or
`AUTH_CALLBACK_URL`). Not called by the FE directly. Always clears `bdg_oauth`
and ends in a `302` to `${PUBLIC_BASE_URL}${redirect}`:

| Outcome | Redirect | Session |
|---------|----------|---------|
| Success | `${PUBLIC_BASE_URL}${redirect}` | `bdg_session` set |
| Account deactivated (`users.status = 'deleted'`) | `...${redirect}?auth_error=deactivated` | none |
| Anything else (user cancelled, state mismatch / missing `bdg_oauth`, expired cookie, token exchange or profile failure, DB/Redis error) | `...${redirect}?auth_error=failed` | none |

`auth_error` is appended with `&` when `redirect` already has a query, and
before any `#fragment`. **`503`** `auth unavailable` (JSON) when Google
credentials are not configured.

#### `auth_error` values
| Value | Meaning |
|-------|---------|
| `deactivated` | The account exists but has been deactivated; sign-in refused |
| `failed` | Sign-in did not complete for any other reason; safe to retry |

### 9.3 `GET /v1/me`

The signed-in user. Responds with `Cache-Control: no-store`.

#### Success `200`
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "Radit",
    "email": "radit@example.com",
    "avatar_url": "https://lh3.googleusercontent.com/a/..."
  }
}
```

`avatar_url` is `null` when the provider has no picture. `name` falls back to
the email when the provider returns no name.

#### Errors
| Status | `error` | When |
|--------|---------|------|
| 401 | `unauthorized` | No session cookie, unknown/expired session, deactivated user, or Redis unavailable |

### 9.4 `POST /v1/auth/logout`

Revokes the current session (if any) and expires `bdg_session`. **`204`** with
no body, even when there was no session or Redis is down. Subject to the
`Origin` check (`403` `origin not allowed`).

---

## 10. User reviews

Signed-in users can write one review per cafe (BCF-30). New and edited reviews
are `pending` until approved in mission control; only
[visible reviews](#visible-reviews) are public. Partnership reviews are entered
by an admin and have no user account. None of these endpoints change the
cafe's `updated_at`.

### Review status
| Value | Meaning |
|-------|---------|
| `pending` | Submitted or edited; waiting for moderation. Not public. |
| `approved` | Public (while the author's account is active). |
| `rejected` | Refused in moderation. Not public; the author still sees it and can edit it (back to `pending`). |
| `deleted` | Deleted by its author (or superseded). Never returned. |

### Review item
| Field | Type | Notes |
|-------|------|-------|
| `id` | int | Review id. |
| `reviewer_name` | string | Display name chosen by the author (not the account name). |
| `overall_score` | number \| null | 1–5. `null` on partnership reviews without a score and on legacy rating-only reviews. |
| `wfc_score` | number \| null | 1–5 work-from-cafe score, or `null`. |
| `content` | string \| null | Review text, or `null`. Single-language. |
| `visited_at` | string \| null | Month of the visit, `"YYYY-MM"`, or `null`. |
| `created_at` / `updated_at` | string | `"YYYY-MM-DDTHH:MM:SSZ"`. `updated_at` moves on every edit. |
| `external_url_title` / `external_url_link` | string \| null | Optional external link (e.g. a partner's article), set by admins. |
| `ratings` | object | Map rating category type → score given by this review, e.g. `{"vibe": 4, "noise": 2}`. `{}` when none. |

### 10.1 `GET /v1/cafe/:id/reviews`

Visible reviews of the cafe, pinned first, then most recently updated
(`is_pinned` desc, `updated_at` desc, `id` desc), 5 per page.

| Query param | Notes |
|-------------|-------|
| `cursor` | Optional. The `next_cursor` of the previous page; omit for the first page. Opaque. |

#### Success `200`
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 41,
        "reviewer_name": "Radit",
        "overall_score": 4.5,
        "wfc_score": 4.2,
        "content": null,
        "visited_at": "2024-11",
        "created_at": "2024-11-12T08:00:00Z",
        "updated_at": "2026-05-03T09:55:59Z",
        "external_url_title": null,
        "external_url_link": null,
        "ratings": { "vibe": 3.8, "noise": 1 }
      }
    ],
    "next_cursor": "MXwyMDI2LTA1LTAzVDA5OjU1OjU5LjI0NjU2MXw0MQ"
  }
}
```

`items` is always an array (possibly empty). `next_cursor` is `null` on the
last page.

#### Errors
| Status | `error` | When |
|--------|---------|------|
| 400 | `invalid cursor` | `cursor` is malformed |
| 404 | `cafe not found` | No cafe with the given `id` exists |
| 500 | `failed to fetch reviews` | Unexpected server / DB error |

### 10.2 `GET /v1/cafe/:id/reviews/mine`

The signed-in user's review of the cafe, in any status except `deleted`.
Requires a session (`401` `unauthorized` otherwise). `Cache-Control: no-store`.

#### Success `200`
A [review item](#review-item) plus `status`:
```json
{
  "success": true,
  "data": {
    "id": 57,
    "reviewer_name": "Dina",
    "overall_score": 4,
    "wfc_score": null,
    "content": "Cozy corner, decent latte.",
    "visited_at": "2026-09",
    "created_at": "2026-10-19T03:12:40Z",
    "updated_at": "2026-10-19T03:12:40Z",
    "external_url_title": null,
    "external_url_link": null,
    "ratings": { "vibe": 4 },
    "status": "pending"
  }
}
```

#### Errors
| Status | `error` | When |
|--------|---------|------|
| 401 | `unauthorized` | Not signed in |
| 404 | `cafe not found` / `review not found` | Unknown cafe / the user has no review of it |
| 500 | `failed to fetch review` | Unexpected server / DB error |

### 10.3 `POST /v1/cafe/:id/reviews/mine` / 10.4 `PUT /v1/cafe/:id/reviews/mine`

Create (POST) or replace (PUT) the signed-in user's review. Both store the
review as `pending` (PUT also bumps `updated_at`, so an approved review leaves
the public list until re-approved) and replace all of its ratings in one
transaction. Requires a session and passes the `Origin` check.

#### Body (JSON)
| Field | Type | Required | Rules |
|-------|------|----------|-------|
| `overall_score` | number | yes | 1–5. |
| `wfc_score` | number \| null | no | 1–5 when set. |
| `reviewer_name` | string | yes | Trimmed; 1–50 characters. |
| `visited_at` | string \| null | no | `"YYYY-MM"`, not after the current month (WIB). Stored as the first of the month. `""` = unset. |
| `content` | string \| null | no | Trimmed; at most 2000 characters. Empty = `null`. |
| `ratings` | object | no | Map rating category type → score. The type must exist (see `/v1/filters`, plus `price-rank`) and the score must lie within that type's bucket span (lowest `lower_bound` to highest `upper_bound`). Omitted / `{}` removes all ratings. |

```json
{
  "overall_score": 4,
  "wfc_score": null,
  "reviewer_name": "Dina",
  "visited_at": "2026-09",
  "content": "Cozy corner, decent latte.",
  "ratings": { "vibe": 4, "noise": 2 }
}
```

#### Success
POST **`201`**, PUT **`200`**: the saved review, same shape as
[10.2](#102-get-v1cafeidreviewsmine) (`status` is `pending`).

#### Errors
| Status | `error` | When |
|--------|---------|------|
| 400 | `invalid request body` | Not valid JSON / wrong field types |
| 400 | `overall_score is required and must be between 1 and 5` | |
| 400 | `wfc_score must be between 1 and 5` | |
| 400 | `reviewer_name is required and must be at most 50 characters` | missing, blank, or too long |
| 400 | `visited_at must be YYYY-MM and not in the future` | |
| 400 | `content must be at most 2000 characters` | |
| 400 | `unknown rating type: <type>` | `ratings` key is not a rating type |
| 400 | `rating score out of range: <type> must be between <min> and <max>` | |
| 401 | `unauthorized` | Not signed in |
| 403 | `origin not allowed` | Disallowed `Origin` |
| 404 | `cafe not found` | Unknown cafe |
| 404 | `review not found` | PUT: the user has no review of this cafe |
| 409 | `you already reviewed this cafe` | POST: the user already has a non-deleted review of this cafe (use PUT) |
| 500 | `failed to save review` | Unexpected server / DB error |

### 10.5 `DELETE /v1/cafe/:id/reviews/mine`

Soft-deletes the signed-in user's review (`status` → `deleted`); the user may
then POST a new one. **`204`** with no body.

#### Errors
| Status | `error` | When |
|--------|---------|------|
| 401 | `unauthorized` | Not signed in |
| 403 | `origin not allowed` | Disallowed `Origin` |
| 404 | `cafe not found` / `review not found` | Unknown cafe / no review to delete |
| 500 | `failed to delete review` | Unexpected server / DB error |
