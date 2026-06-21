package repository

import (
	"context"
	"fmt"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuicksearchRepository struct {
	db *pgxpool.Pool
}

func NewQuicksearchRepository(db *pgxpool.Pool) *QuicksearchRepository {
	return &QuicksearchRepository{db: db}
}

// Locations matches locations (cafes, POIs, areas, districts) by name via
// trigram similarity (LIMIT 10), then resolves each match's ancestor *ids* via
// PostGIS containment: the containing district (area + poi) and the containing
// area (poi only). The LATERAL joins are gated by the matched row's type so we
// only do containment lookups where relevant. We need only the ids — they
// compose the /explore splat (Slug) below.
func (r *QuicksearchRepository) Locations(ctx context.Context, q, locType string, limit int) ([]model.QuicksearchResult, error) {
	rows, err := r.db.Query(ctx,
		`WITH matched AS (
		     SELECT id, name, type, coordinates
		     FROM location
		     WHERE name ILIKE '%' || $1 || '%'
		       AND ($2 = '' OR type = $2::location_type_enum)
		       AND status = 'active'
		     ORDER BY similarity(name, $1) DESC
		     LIMIT $3
		 )
		 SELECT m.id, m.name, m.type, d.id, a.id
		 FROM matched m
		 LEFT JOIN LATERAL (
		     SELECT d.id FROM location d
		     WHERE d.type = 'district' AND d.status = 'active' AND d.id <> m.id
		       AND ST_Within(ST_PointOnSurface(m.coordinates), d.coordinates)
		     ORDER BY d.id LIMIT 1
		 ) d ON m.type IN ('area', 'poi')
		 LEFT JOIN LATERAL (
		     SELECT a.id FROM location a
		     WHERE a.type = 'area' AND a.status = 'active' AND a.id <> m.id
		       AND ST_Within(ST_PointOnSurface(m.coordinates), a.coordinates)
		       -- Nearest-area fallback (commented out; strict containment for now).
		       -- To always resolve an area for POIs, drop the ST_Within line above
		       -- and order by distance instead:
		       -- ORDER BY ST_Distance(m.coordinates::geography, a.coordinates::geography) LIMIT 1
		     ORDER BY a.id LIMIT 1
		 ) a ON m.type = 'poi'
		 ORDER BY similarity(m.name, $1) DESC`,
		q, locType, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.QuicksearchResult
	for rows.Next() {
		var res model.QuicksearchResult
		var districtID, areaID *string
		if err := rows.Scan(&res.ID, &res.Name, &res.Type, &districtID, &areaID); err != nil {
			return nil, err
		}
		// Slug is the canonical /explore splat (ancestor ids + self), built only
		// when the full chain is present (district=1, area=2, poi=3 segments).
		// An incomplete chain leaves Slug empty so the frontend can fall back to
		// the legacy query_id/query_type route. Cafes have no explore slug.
		switch res.Type {
		case constants.LocationTypeDistrict:
			res.Slug = res.ID
		case constants.LocationTypeArea:
			if districtID != nil {
				res.Slug = *districtID + "/" + res.ID
			}
		case constants.LocationTypePOI:
			if districtID != nil && areaID != nil {
				res.Slug = *districtID + "/" + *areaID + "/" + res.ID
			}
		}
		results = append(results, res)
	}
	if results == nil {
		results = []model.QuicksearchResult{}
	}
	return results, rows.Err()
}

// Filters matches the slugged, SRP-eligible filters (tags + rating buckets,
// price-rank included) by name. It matches against both language columns
// (name/name_indo) so a query in either language hits, ranks by the better
// trigram score, and returns the localized name for display. Rating-sourced
// results fold their type label into the display name the same way the SRP
// VariantLinks do (see formatSrpLabel): appended in EN ("Quiet" -> "Quiet Noise
// Level"), prepended in ID ("Quiet" -> "Tingkat Kebisingan Quiet"). The label is
// display-only — matching stays on the bare name/name_indo. Slugs mirror
// service.srpSlug: tag slug as-is; rating/price = "<slug>-<type>" (keep in sync;
// type is TEXT since 008_rating_type_label). pg_trgm + the GIN indexes from
// migration 009 back the ILIKE/similarity.
func (r *QuicksearchRepository) Filters(ctx context.Context, q, lang string, limit int) ([]model.QuicksearchResult, error) {
	tagName := localized("$2", "name_indo", "name")            // $1 = q, $2 = lang
	ratName := localized("$2", "rc.name_indo", "rc.name")      // rating bucket name
	ratLabel := localized("$2", "rtl.label_indo", "rtl.label") // rating type label
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		WITH f AS (
		    SELECT slug AS slug, %[1]s AS name,
		           GREATEST(similarity(name, $1), similarity(name_indo, $1)) AS score
		    FROM tag
		    WHERE slug IS NOT NULL AND slug <> ''
		      AND (name ILIKE '%%' || $1 || '%%' OR name_indo ILIKE '%%' || $1 || '%%')
		    UNION ALL
		    SELECT rc.slug || '-' || rc.type AS slug,
		           CASE WHEN $2 = '%[4]s' THEN %[3]s || ' ' || %[2]s
		                ELSE %[2]s || ' ' || %[3]s END AS name,
		           GREATEST(similarity(rc.name, $1), similarity(rc.name_indo, $1)) AS score
		    FROM rating_category rc
		    LEFT JOIN rating_type_label rtl ON rtl.type = rc.type
		    WHERE rc.slug IS NOT NULL AND rc.slug <> ''
		      AND (rc.name ILIKE '%%' || $1 || '%%' OR rc.name_indo ILIKE '%%' || $1 || '%%')
			  AND rc.type <> 'price-rank' -- price-rank is surfaced as price tiers, not a rating category, so exclude it from filter search results
		)
		SELECT slug, name FROM f
		ORDER BY score DESC
		LIMIT $3
	`, tagName, ratName, ratLabel, constants.LangIndonesian), q, lang, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []model.QuicksearchResult{}
	for rows.Next() {
		var res model.QuicksearchResult
		if err := rows.Scan(&res.Slug, &res.Name); err != nil {
			return nil, err
		}
		res.ID = res.Slug
		res.Type = constants.QuicksearchTypeFilter
		results = append(results, res)
	}
	return results, rows.Err()
}
