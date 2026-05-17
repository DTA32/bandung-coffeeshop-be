CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS postgis;
       
CREATE TYPE location_type_enum AS ENUM (
    'cafe',
    'poi',
    'area',
    'district'
);

CREATE TYPE location_status_enum AS ENUM (
    'active',
    'closed',
    'deleted'
);

CREATE TABLE IF NOT EXISTS location (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    gmaps_id    TEXT,
    type        location_type_enum NOT NULL,
    status      location_status_enum NOT NULL DEFAULT 'active',
    coordinates GEOMETRY(Geometry, 4326),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_location_name_trgm ON location USING GIN (name gin_trgm_ops);

CREATE TABLE IF NOT EXISTS location_image (
    id              SERIAL PRIMARY KEY,
    location_id     TEXT NOT NULL REFERENCES location(id),
    url             TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    display_order   INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TYPE rating_category_type_enum AS ENUM (
    'price-rank',
    'vibe',
    'noise',
    'wifi',
    'meals',
    'atmosphere',
    'parking'
);

CREATE TABLE IF NOT EXISTS rating_category (
    id                SERIAL PRIMARY KEY,
    name              TEXT NOT NULL,
    short_description TEXT NOT NULL DEFAULT '',
    long_description  TEXT NOT NULL DEFAULT '',
    slug              TEXT,
    type              rating_category_type_enum NOT NULL,
    lower_bound       NUMERIC(10, 2) NOT NULL,
    upper_bound       NUMERIC(10, 2) NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cafe (
    id          SERIAL PRIMARY KEY,
    location_id TEXT NOT NULL UNIQUE REFERENCES location(id),
    instagram   TEXT,
    open_hour   TIME,
    close_hour  TIME,
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cafe_review (
    id             SERIAL PRIMARY KEY,
    cafe_id        INT NOT NULL REFERENCES cafe(id),
    is_subjective  BOOLEAN NOT NULL DEFAULT FALSE,
    overall_score  NUMERIC(3, 2),
    wfc_score      NUMERIC(3, 2),
    personal_score NUMERIC(3, 2),
    content        TEXT,
    visited_at     DATE,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cafe_rating (
    id                         SERIAL PRIMARY KEY,
    cafe_id                    INT NOT NULL REFERENCES cafe(id),
    category_type              rating_category_type_enum NOT NULL,
    score                      NUMERIC(10, 2) NOT NULL,
    short_description_override TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (cafe_id, category_type)
);

CREATE TABLE IF NOT EXISTS tag (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    slug        TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cafe_tag (
    cafe_id INT NOT NULL REFERENCES cafe(id),
    tag_id  INT NOT NULL REFERENCES tag(id),
    visible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (cafe_id, tag_id)
);

CREATE TABLE IF NOT EXISTS cafe_price (
    id                SERIAL PRIMARY KEY,
    cafe_id           INT NOT NULL UNIQUE REFERENCES cafe(id),
    price_range_min   INT,
    price_range_max   INT,
    coffee_price_min  INT,
    coffee_price_max  INT,
    snack_price_min   INT,
    snack_price_max   INT,
    food_price_min    INT,
    food_price_max    INT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
