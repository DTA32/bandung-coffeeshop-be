#!/usr/bin/env python3
"""
Seeds area and district locations from OpenStreetMap polygon data.
- Areas: fetches polygon boundaries from Nominatim using OSM relation IDs or search.
- Districts: uses hardcoded approximate bounding polygons (informal Bandung regions).

Run from anywhere: python3 migrations/005_area_district_seeder.py
"""

import json
import re
import time
import urllib.request
import urllib.parse
from pathlib import Path

import psycopg2

REPO_ROOT = Path(__file__).parent.parent
ENV_FILE = REPO_ROOT / ".env"

NOMINATIM_BASE = "https://nominatim.openstreetmap.org"
USER_AGENT = "BDGCafe/1.0 (bandung2003@gmail.com)"


def load_env(path: Path) -> dict[str, str]:
    env = {}
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, value = line.partition("=")
        env[key.strip()] = value.strip()
    return env


def make_location_id(key: str) -> str:
    s = key.lower().replace(" ", "-")
    s = re.sub(r"[^a-z0-9-]", "", s)
    s = re.sub(r"-+", "-", s)
    return s.strip("-")


def nominatim_get(url: str) -> dict | None:
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())
    except Exception as e:
        print(f"  HTTP error: {e}")
        return None


def fetch_by_relation_id(osm_id: int) -> dict | None:
    url = f"{NOMINATIM_BASE}/lookup?osm_ids=R{osm_id}&format=geojson&polygon_geojson=1"
    data = nominatim_get(url)
    time.sleep(3)
    if not data or not data.get("features"):
        return None
    return data["features"][0]["geometry"]


def fetch_by_search(query: str) -> dict | None:
    encoded = urllib.parse.quote(query)
    url = f"{NOMINATIM_BASE}/search?q={encoded}&format=geojson&polygon_geojson=1&limit=1"
    data = nominatim_get(url)
    time.sleep(3)
    if not data or not data.get("features"):
        return None
    geom = data["features"][0]["geometry"]
    if geom["type"] not in ("Polygon", "MultiPolygon"):
        print(f"  Non-polygon result ({geom['type']}) — skipped")
        return None
    return geom


def make_rect_polygon(min_lng: float, min_lat: float, max_lng: float, max_lat: float) -> dict:
    return {
        "type": "Polygon",
        "coordinates": [[
            [min_lng, min_lat],
            [max_lng, min_lat],
            [max_lng, max_lat],
            [min_lng, max_lat],
            [min_lng, min_lat],
        ]],
    }


# Nine areas have confirmed OSM relation IDs from Nominatim exploration.
# Three fall back to kelurahan-level search (Riau → Cihapit, Dipatiukur → Lebakgede).
AREAS: list[tuple[str, dict]] = [
    ("Dago",        {"osm_id": 13290175}),
    ("Buahbatu",    {"osm_id": 13290241}),
    ("Pajajaran",   {"osm_id": 13290193}),
    ("Cigadung",    {"osm_id": 13290104}),
    ("Sukaluyu",    {"osm_id": 13290105}),
    ("Antapani",    {"osm_id": 13290118}),
    ("Arcamanik",   {"osm_id": 13290122}),
    ("Ujungberung", {"osm_id": 13290128}),
    ("Ciumbuleuit", {"osm_id": 13290182}),
    ("Riau",        {"search": "Cihapit, Cibeunying Kidul, Bandung"}),
    ("Dipatiukur",  {"search": "Lebakgede, Coblong, Bandung"}),
    ("Lembang",     {"search": "Lembang, Kabupaten Bandung Barat, Indonesia"}),
]

# Informal Bandung districts have no official OSM boundary.
# Approximate bounding boxes: (name, min_lng, min_lat, max_lng, max_lat).
DISTRICTS: list[tuple[str, float, float, float, float]] = [
    ("Bandung Utara",   107.58, -6.90, 107.65, -6.82),
    ("Bandung Tengah",  107.59, -6.93, 107.64, -6.90),
    ("Bandung Selatan", 107.60, -6.97, 107.68, -6.93),
    ("Bandung Timur",   107.63, -6.94, 107.73, -6.88),
    ("Bandung Barat",   107.54, -6.95, 107.61, -6.88),
]


def main():
    env = load_env(ENV_FILE)
    dsn = (
        f"host={env.get('DB_HOST', 'localhost')} "
        f"port={env.get('DB_PORT', '5432')} "
        f"user={env.get('DB_USER', 'postgres')} "
        f"password={env.get('DB_PASSWORD', '')} "
        f"dbname={env.get('DB_NAME', 'bandung_coffeeshop')} "
        f"sslmode=disable"
    )

    rows = []

    print("Fetching area polygons from Nominatim (1 req/sec)...")
    for name, config in AREAS:
        loc_id = make_location_id(name)
        print(f"  {name} ({loc_id})...", end=" ", flush=True)
        if "osm_id" in config:
            geom = fetch_by_relation_id(config["osm_id"])
        else:
            geom = fetch_by_search(config["search"])
        if geom:
            print(f"OK ({geom['type']})")
            rows.append({"id": loc_id, "name": name, "type": "area", "geojson": json.dumps(geom)})
        else:
            print("FAILED — skipped")

    print("\nBuilding district polygons (hardcoded approximate bounds)...")
    for name, min_lng, min_lat, max_lng, max_lat in DISTRICTS:
        loc_id = make_location_id(name)
        geom = make_rect_polygon(min_lng, min_lat, max_lng, max_lat)
        print(f"  {name} ({loc_id}) — OK")
        rows.append({"id": loc_id, "name": name, "type": "district", "geojson": json.dumps(geom)})

    if not rows:
        print("\nNo rows to insert.")
        return

    print(f"\nInserting {len(rows)} locations...")
    with psycopg2.connect(dsn) as conn:
        with conn.cursor() as cur:
            cur.executemany(
                """
                INSERT INTO location (id, name, type, status, coordinates)
                VALUES (
                    %(id)s, %(name)s, %(type)s, 'active',
                    ST_GeomFromGeoJSON(%(geojson)s)
                )
                ON CONFLICT (id) DO NOTHING
                """,
                rows,
            )
        conn.commit()

    print(f"Done. Seeded {len(rows)} area/district locations.")


if __name__ == "__main__":
    main()
