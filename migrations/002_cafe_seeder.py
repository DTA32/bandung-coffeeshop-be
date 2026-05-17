#!/usr/bin/env python3
"""
Seeds the database with cafe data from master.json.
Reads DB connection from .env at the repo root.
Run from anywhere: python3 migrations/seeder.py
"""

import json
import re
import time
import urllib.request
import base64
import struct
import csv

from pathlib import Path

import psycopg2

REPO_ROOT = Path(__file__).parent.parent
INPUT = REPO_ROOT / "cafe_master.json"
CSV_INPUT = REPO_ROOT / "cafe_master.csv"
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


def make_location_id(key: str) -> str:
    s = key.lower()
    s = s.replace(" ", "-")
    s = re.sub(r"[^a-z0-9-]", "", s)
    # collapse multiple dashes into one
    s = re.sub(r"-+", "-", s)
    s = s.strip("-")
    return s

def decode_place_id(url: str) -> dict:
    """
    Decode a Google Maps URL and extract the Place ID (ChIJ... format).
 
    Args:
        url: Full Google Maps URL containing !1s0x...!8m segment
 
    Returns:
        dict with keys: hex_string, bytes_array, place_id
        or raises ValueError if the URL is invalid/unsupported
    """
    match = re.search(r"!1s(0x[0-9a-fA-F]+:0x[0-9a-fA-F]+)", url)
    if not match:
        raise ValueError(
            "Could not find the hex segment (!1s...!8m) in the URL.\n"
            "Make sure you paste the full Google Maps place URL."
        )

    raw = match.group(1)
    part1, part2 = raw.split(":")

    hex1 = part1.replace("0x", "").zfill(16)
    hex2 = part2.replace("0x", "").zfill(16)

    # Convert each hex part to little-endian 8-byte sequence.
    # The two halves are separated by 0x11, which is a protobuf field tag:
    # (field_number=2 << 3) | wire_type=1 (fixed64) = 0x11
    bytes1 = struct.pack("<Q", int(hex1, 16))
    bytes2 = struct.pack("<Q", int(hex2, 16))
    all_bytes = bytes1 + b"\x11" + bytes2

    # Base64url encode (no padding), then prepend ChIJ
    b64 = base64.urlsafe_b64encode(all_bytes).rstrip(b"=").decode()
    place_id = "ChIJ" + b64

    return {
        "hex_string": raw,
        "bytes_array": list(all_bytes),
        "place_id": place_id,
    }


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

    with open(INPUT, encoding="utf-8") as f:
        data: dict = json.load(f)
    
    with open(CSV_INPUT, encoding="utf-8") as f:
        # url is in 3rd column
        csv_data = list(csv.reader(f))

    locations = []
    cafes = []
    reviews = []

    for i, (key, val) in enumerate(data.items()):
        if not key:
            continue
        loc_id = make_location_id(key)
        if not loc_id:
            continue
        
        cid = val.get("cid", "")
        url = csv_data[i][2]
        decoded = decode_place_id(url)
        if decoded:
            gmaps_id = decoded["place_id"]
        else:
            gmaps_id = cid

        locations.append({
            "id": loc_id,
            "name": val.get("name", ""),
            "description": val.get("address", ""),
            "gmaps_id": gmaps_id,
            "type": "cafe",
            "status": val.get("status", "active"),
            "lat": val.get("lat", 0),
            "lng": val.get("lng", 0),
        })
        cafes.append(loc_id)

        review = val.get("review", "")
        if review:
            reviews.append({"location_id": loc_id, "content": review})
    
    dry_run = False
    
    if dry_run:
        print("Dry run mode - no database changes will be made.")
        print(f"Locations: {locations}")
        print(f"Cafes: {cafes}")
        print(f"Reviews: {reviews}")
        return

    with psycopg2.connect(dsn) as conn:
        with conn.cursor() as cur:
            cur.executemany(
                """
                INSERT INTO location (id, name, description, gmaps_id, type, status, coordinates)
                VALUES (
                    %(id)s, %(name)s, %(description)s, %(gmaps_id)s, %(type)s,
                    %(status)s, ST_SetSRID(ST_MakePoint(%(lng)s, %(lat)s), 4326)
                )
                """,
                locations,
            )

            cur.executemany(
                "INSERT INTO cafe (location_id) VALUES (%s) ON CONFLICT DO NOTHING",
                [(lid,) for lid in cafes],
            )

            cur.executemany(
                """
                INSERT INTO cafe_review (cafe_id, content, is_subjective)
                SELECT id, %(content)s, FALSE
                FROM cafe WHERE location_id = %(location_id)s
                """,
                reviews,
            )

        conn.commit()

    print(
        f"Seeded {len(locations)} locations, {len(cafes)} cafes,"
        f" {len(reviews)} reviews."
    )


if __name__ == "__main__":
    main()
