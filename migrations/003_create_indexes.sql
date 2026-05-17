-- 1. PostGIS spatial index on location.coordinates.
--    Speeds up ST_Within (polygon mode), ST_DWithin (radius mode), and the
--    `area` LEFT JOIN subselect.
CREATE INDEX IF NOT EXISTS idx_location_coordinates_gist
    ON location USING GIST (coordinates);

-- 2. Partial GIST for the area-containment subselect, which always filters
--    type='area' AND status='active'. Smaller and the planner picks it for
--    the per-row subselect.
CREATE INDEX IF NOT EXISTS idx_location_area_coordinates_gist
    ON location USING GIST (coordinates)
    WHERE type = 'area' AND status = 'active';

-- 3. Slug lookups. Both columns are nullable by design (rows without a slug
--    are display-only and not searchable). Partial UNIQUE indexes:
--    searchable rows are guaranteed unique; NULL-slug rows coexist freely.
CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_slug
    ON tag(slug) WHERE slug IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_rating_category_type_slug
    ON rating_category(type, slug) WHERE slug IS NOT NULL;

-- 4. cafe_tag(tag_id, cafe_id). PK is (cafe_id, tag_id) so it does not help
--    when filtering by tag.id. Partial on visible since hidden rows never
--    match search/remark.
CREATE INDEX IF NOT EXISTS idx_cafe_tag_tag_id
    ON cafe_tag(tag_id, cafe_id) WHERE visible;

-- 5. Thumbnail subselect: ORDER BY display_order ASC, id ASC LIMIT 1 by location_id.
CREATE INDEX IF NOT EXISTS idx_location_image_thumbnail
    ON location_image(location_id, display_order, id);

-- 6. Latest review per cafe (used by sort=rating LATERAL).
CREATE INDEX IF NOT EXISTS idx_cafe_review_cafe_latest
    ON cafe_review(cafe_id, visited_at DESC NULLS LAST, id DESC);

-- 7. cafe sort columns. Default sort leads with is_featured DESC, updated_at DESC.
CREATE INDEX IF NOT EXISTS idx_cafe_featured_updated
    ON cafe(is_featured, updated_at DESC);

-- 8. cafe_price sorts (asc uses min, desc uses max).
CREATE INDEX IF NOT EXISTS idx_cafe_price_min ON cafe_price(price_range_min);
CREATE INDEX IF NOT EXISTS idx_cafe_price_max ON cafe_price(price_range_max);

-- 9. price-rank range lookup: WHERE type = 'price-rank' ORDER BY lower_bound.
CREATE INDEX IF NOT EXISTS idx_rating_category_type_lb
    ON rating_category (type, lower_bound);
