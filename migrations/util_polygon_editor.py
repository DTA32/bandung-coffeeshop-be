#!/usr/bin/env python3
"""
Two-way polygon editor helper for location rows.

Usage:
  # Export — saves the polygon(s) to a file and (if small enough) prints a
  # geojson.io edit URL. Pass several ids to bundle them into one GeoJSON,
  # handy for checking area coverage when building a district.
  python3 migrations/util_polygon_editor.py export <location_id> [<location_id> ...]

  # Import — reads a .geojson file and updates the DB row
  python3 migrations/util_polygon_editor.py import <location_id> <file.geojson>

Workflow:
  1. Export to get the geojson.io URL (or the saved file for multiple ids).
  2. Edit the polygon in the browser.
  3. In geojson.io: Save → GeoJSON (downloads a file).
  4. Import that file back (one location at a time).
"""

import json
import sys
import urllib.parse
from pathlib import Path

import psycopg2

REPO_ROOT = Path(__file__).parent.parent
ENV_FILE = REPO_ROOT / ".env"


def load_env(path: Path) -> dict[str, str]:
    env = {}
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, value = line.partition("=")
        env[key.strip()] = value.strip()
    return env


def get_dsn() -> str:
    env = load_env(ENV_FILE)
    return (
        f"host={env.get('DB_HOST', 'localhost')} "
        f"port={env.get('DB_PORT', '5432')} "
        f"user={env.get('DB_USER', 'postgres')} "
        f"password={env.get('DB_PASSWORD', '')} "
        f"dbname={env.get('DB_NAME', 'bandung_coffeeshop')} "
        f"sslmode=disable"
    )


def cmd_export(location_ids: list[str]) -> None:
    with psycopg2.connect(get_dsn()) as conn:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT id, name, ST_AsGeoJSON(coordinates) "
                "FROM location WHERE id = ANY(%s) AND coordinates IS NOT NULL",
                (location_ids,),
            )
            rows = cur.fetchall()

    if not rows:
        print(f"No locations with a polygon found for: {', '.join(location_ids)}")
        sys.exit(1)

    found = {row[0] for row in rows}
    missing = [lid for lid in location_ids if lid not in found]

    feature_collection = {
        "type": "FeatureCollection",
        "features": [
            {
                "type": "Feature",
                "properties": {"id": loc_id, "name": name},
                "geometry": json.loads(geojson_str),
            }
            for loc_id, name, geojson_str in rows
        ],
    }

    # Single id keeps the "<id>.geojson" name; multiple ids bundle into one file.
    out_file = f"{location_ids[0]}.geojson" if len(location_ids) == 1 else "multi_export.geojson"
    with open(out_file, "w", encoding="utf-8") as f:
        json.dump(feature_collection, f, indent=2)
    print(f"Saved {len(rows)} location(s) to: {out_file}")
    if missing:
        print(f"Not found or no polygon (skipped): {', '.join(missing)}")

    encoded = urllib.parse.quote(json.dumps(feature_collection))
    url = f"https://geojson.io/#data=data:application/json,{encoded}"

    # geojson.io has a URL length limit — warn if the polygon(s) are too large
    if len(url) > 8000:
        print("\nPolygon data is too large for a URL. Open geojson.io and drag the saved file in instead:")
        print("  https://geojson.io")
        print(f"  Then drag & drop: {out_file}")
    else:
        print(f"\nOpen in geojson.io:\n  {url}")

    print("\nAfter editing, save as GeoJSON from geojson.io (Save → GeoJSON), then import each id with:")
    print("  python3 migrations/util_polygon_editor.py import <location_id> <downloaded_file.geojson>")


def cmd_import(location_id: str, geojson_file: str) -> None:
    path = Path(geojson_file)
    if not path.exists():
        print(f"File not found: {geojson_file}")
        sys.exit(1)

    data = json.loads(path.read_text())

    # Accept either a FeatureCollection, a Feature, or a bare Geometry
    if data.get("type") == "FeatureCollection":
        features = data.get("features", [])
        if not features:
            print("FeatureCollection has no features.")
            sys.exit(1)
        geometry = features[0]["geometry"]
    elif data.get("type") == "Feature":
        geometry = data["geometry"]
    elif data.get("type") in ("Polygon", "MultiPolygon"):
        geometry = data
    else:
        print(f"Unrecognised GeoJSON type: {data.get('type')}")
        sys.exit(1)

    geojson_str = json.dumps(geometry)

    with psycopg2.connect(get_dsn()) as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                UPDATE location
                SET coordinates = ST_GeomFromGeoJSON(%s), updated_at = NOW()
                WHERE id = %s
                RETURNING id, name
                """,
                (geojson_str, location_id),
            )
            updated = cur.fetchone()
        conn.commit()

    if not updated:
        print(f"No location found with id '{location_id}' — doing insertion instead.")
        with psycopg2.connect(get_dsn()) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    INSERT INTO location (id, name, type, status, coordinates)
                    VALUES (%s, %s, 'area', 'active', ST_GeomFromGeoJSON(%s))
                    RETURNING id, name
                    """,
                    (location_id, location_id, geojson_str),
                )
                updated = cur.fetchone()
            conn.commit()
        
        print(f"Inserted new location '{updated[1]}' ({updated[0]}) with the provided polygon.")
        sys.exit(0)

    print(f"Updated polygon for '{updated[1]}' ({updated[0]})")


def main() -> None:
    args = sys.argv[1:]
    if len(args) < 2 or args[0] not in ("export", "import"):
        print(__doc__)
        sys.exit(1)

    cmd = args[0]
    if cmd == "export":
        cmd_export(args[1:])
    elif cmd == "import":
        if len(args) < 3:
            print("Usage: polygon_editor.py import <location_id> <file.geojson>")
            sys.exit(1)
        cmd_import(args[1], args[2])


if __name__ == "__main__":
    main()
