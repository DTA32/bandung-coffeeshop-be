package repository

import (
	"context"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LocationRepository struct {
	db *pgxpool.Pool
}

func NewLocationRepository(db *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{db: db}
}

func (r *LocationRepository) Quicksearch(ctx context.Context, q, locType string) ([]model.QuicksearchResult, error) {
	// Match by trigram similarity first (LIMIT 10), then resolve each match's
	// ancestors via PostGIS containment: the containing district (area + poi)
	// and the containing area (poi only). The LATERAL joins are gated by the
	// matched row's type so we only do containment lookups where relevant.
	rows, err := r.db.Query(ctx,
		`WITH matched AS (
		     SELECT id, name, type, coordinates
		     FROM location
		     WHERE name ILIKE '%' || $1 || '%'
		       AND ($2 = '' OR type = $2::location_type_enum)
		       AND status = 'active'
		     ORDER BY similarity(name, $1) DESC
		     LIMIT 10
		 )
		 SELECT m.id, m.name, m.type, d.id, d.name, a.id, a.name
		 FROM matched m
		 LEFT JOIN LATERAL (
		     SELECT d.id, d.name FROM location d
		     WHERE d.type = 'district' AND d.status = 'active' AND d.id <> m.id
		       AND ST_Within(ST_PointOnSurface(m.coordinates), d.coordinates)
		     ORDER BY d.id LIMIT 1
		 ) d ON m.type IN ('area', 'poi')
		 LEFT JOIN LATERAL (
		     SELECT a.id, a.name FROM location a
		     WHERE a.type = 'area' AND a.status = 'active' AND a.id <> m.id
		       AND ST_Within(ST_PointOnSurface(m.coordinates), a.coordinates)
		       -- Nearest-area fallback (commented out; strict containment for now).
		       -- To always resolve an area for POIs, drop the ST_Within line above
		       -- and order by distance instead:
		       -- ORDER BY ST_Distance(m.coordinates::geography, a.coordinates::geography) LIMIT 1
		     ORDER BY a.id LIMIT 1
		 ) a ON m.type = 'poi'
		 ORDER BY similarity(m.name, $1) DESC`,
		q, locType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.QuicksearchResult
	for rows.Next() {
		var res model.QuicksearchResult
		var districtID, districtName, areaID, areaName *string
		if err := rows.Scan(&res.ID, &res.Name, &res.Type, &districtID, &districtName, &areaID, &areaName); err != nil {
			return nil, err
		}
		res.Ancestors = []model.Location{}
		if districtID != nil {
			res.Ancestors = append(res.Ancestors, model.Location{ID: *districtID, Name: *districtName, Type: constants.LocationTypeDistrict})
		}
		if areaID != nil {
			res.Ancestors = append(res.Ancestors, model.Location{ID: *areaID, Name: *areaName, Type: constants.LocationTypeArea})
		}
		results = append(results, res)
	}
	if results == nil {
		results = []model.QuicksearchResult{}
	}
	return results, rows.Err()
}
