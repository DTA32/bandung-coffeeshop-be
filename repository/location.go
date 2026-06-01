package repository

import (
	"context"
	"errors"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrLocationNotFound = errors.New("location not found")

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

type LocationDetailRow struct {
	ID             string
	Name           string
	Description    string
	Type           string
	PolygonGeoJSON *string
}

// GetByID fetches a single non-deleted location with its geometry as GeoJSON.
func (r *LocationRepository) GetByID(ctx context.Context, id string) (*LocationDetailRow, error) {
	var row LocationDetailRow
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, type::text, ST_AsGeoJSON(coordinates)
		FROM location
		WHERE id = $1 AND status <> 'deleted'
	`, id).Scan(&row.ID, &row.Name, &row.Description, &row.Type, &row.PolygonGeoJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLocationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Ancestors returns the spatial parents of a location ordered district -> area.
// Containment is computed against the location's centroid so it works for both
// polygon (area-in-district) and point (poi-in-area) children.
func (r *LocationRepository) Ancestors(ctx context.Context, id string) ([]model.Location, error) {
	rows, err := r.db.Query(ctx, `
		WITH t AS (
			SELECT id, type, ST_Centroid(coordinates) AS c
			FROM location WHERE id = $1
		)
		SELECT DISTINCT ON (a.type) a.id, a.name, a.type::text
		FROM location a, t
		WHERE a.status = 'active' AND a.id <> t.id
		  AND ST_Within(t.c, a.coordinates)
		  AND ( a.type = 'district'
		     OR (a.type = 'area' AND t.type = 'poi') )
		ORDER BY a.type, a.id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var district, area *model.Location
	for rows.Next() {
		var ref model.Location
		if err := rows.Scan(&ref.ID, &ref.Name, &ref.Type); err != nil {
			return nil, err
		}
		switch ref.Type {
		case "district":
			r := ref
			district = &r
		case "area":
			r := ref
			area = &r
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ancestors := []model.Location{}
	if district != nil {
		ancestors = append(ancestors, *district)
	}
	if area != nil {
		ancestors = append(ancestors, *area)
	}
	return ancestors, nil
}

// Descendants returns the direct spatial children of a location: a district's
// areas, or an area's pois. Containment uses the child's centroid so it works
// for both polygon (area-in-district) and point (poi-in-area) children. Pois
// (and anything else) have no descendants.
func (r *LocationRepository) Descendants(ctx context.Context, row *LocationDetailRow) ([]model.Location, error) {
	switch row.Type {
	case "district":
		return r.queryDescendants(ctx, `
			SELECT a.id, a.name, 'area'::text
			FROM location a
			WHERE a.type = 'area' AND a.status = 'active'
			  AND ST_Within(ST_Centroid(a.coordinates), (SELECT coordinates FROM location WHERE id = $1))
			ORDER BY a.name
		`, row.ID)
	case "area":
		return r.queryDescendants(ctx, `
			SELECT p.id, p.name, 'poi'::text
			FROM location p
			WHERE p.type = 'poi' AND p.status = 'active'
			  AND ST_Within(p.coordinates, (SELECT coordinates FROM location WHERE id = $1))
			ORDER BY p.name
		`, row.ID)
	default:
		return []model.Location{}, nil
	}
}

func (r *LocationRepository) queryDescendants(ctx context.Context, sql, id string) ([]model.Location, error) {
	rows, err := r.db.Query(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	descendants := []model.Location{}
	for rows.Next() {
		var d model.Location
		if err := rows.Scan(&d.ID, &d.Name, &d.Type); err != nil {
			return nil, err
		}
		descendants = append(descendants, d)
	}
	return descendants, rows.Err()
}

// Images returns a location's images ordered by display_order (then id).
func (r *LocationRepository) Images(ctx context.Context, id string) ([]model.LocationImage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT url, description
		FROM location_image
		WHERE location_id = $1
		ORDER BY display_order ASC, id ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	images := []model.LocationImage{}
	for rows.Next() {
		var img model.LocationImage
		if err := rows.Scan(&img.URL, &img.Description); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

// Districts returns all active districts (used by the no-id fallback).
func (r *LocationRepository) Districts(ctx context.Context) ([]LocationDetailRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, type::text
		FROM location
		WHERE type = 'district' AND status = 'active'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	districts := []LocationDetailRow{}
	for rows.Next() {
		var row LocationDetailRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Type); err != nil {
			return nil, err
		}
		districts = append(districts, row)
	}
	return districts, rows.Err()
}
