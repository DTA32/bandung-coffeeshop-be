#!/usr/bin/env python3
"""
Two-way polygon editor helper for location rows.

Usage:
  # Export — prints a geojson.io edit URL and saves the polygon to a file
  python3 migrations/util_polygon_editor.py export <location_id>

  # Import — reads a .geojson file and updates the DB row
  python3 migrations/util_polygon_editor.py import <location_id> <file.geojson>

Workflow:
  1. Export to get the geojson.io URL.
  2. Edit the polygon in the browser.
  3. In geojson.io: Save → GeoJSON (downloads a file).
  4. Import that file back.
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


def cmd_export(location_id: str) -> None:
    with psycopg2.connect(get_dsn()) as conn:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT name, ST_AsGeoJSON(coordinates) FROM location WHERE id = %s",
                (location_id,),
            )
            row = cur.fetchone()

    if not row:
        print(f"No location found with id '{location_id}'")
        sys.exit(1)

    name, geojson_str = row
    geometry = json.loads(geojson_str)

    feature_collection = {
        "type": "FeatureCollection",
        "features": [
            {
                "type": "Feature",
                "properties": {"id": location_id, "name": name},
                "geometry": geometry,
            }
        ],
    }

    with open(f"{location_id}.geojson", "w", encoding="utf-8") as f:
        json.dump(feature_collection, f, indent=2)
    print(f"Saved to: {location_id}.geojson")

    encoded = urllib.parse.quote(json.dumps(feature_collection))
    url = f"https://geojson.io/#data=data:application/json,{encoded}"

    # geojson.io has a URL length limit — warn if the polygon is very large
    if len(url) > 8000:
        print("\nPolygon is too large for a URL. Open geojson.io and drag the saved file in instead:")
        print("  https://geojson.io")
        print(f"  Then drag & drop: {location_id}.geojson")
    else:
        print(f"\nOpen in geojson.io:\n  {url}")

    print(f"\nAfter editing, save as GeoJSON from geojson.io (Save → GeoJSON), then run:")
    print(f"  python3 migrations/polygon_editor.py import {location_id} <downloaded_file.geojson>")


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
        cmd_export(args[1])
    elif cmd == "import":
        if len(args) < 3:
            print("Usage: polygon_editor.py import <location_id> <file.geojson>")
            sys.exit(1)
        cmd_import(args[1], args[2])


if __name__ == "__main__":
    main()
