package repository

import (
	"context"
	"errors"
	"fmt"

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

type LocationDetailRow struct {
	ID             string
	Name           string
	Description    string
	Type           string
	PolygonGeoJSON *string
}

// GetByID fetches a single non-deleted location with its geometry as GeoJSON.
func (r *LocationRepository) GetByID(ctx context.Context, id, lang string) (*LocationDetailRow, error) {
	var row LocationDetailRow
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT id, name, %s, type::text, ST_AsGeoJSON(coordinates)
		FROM location
		WHERE id = $1 AND status <> 'deleted'
	`, localized("$2", "description_indo", "description")), id, lang).Scan(&row.ID, &row.Name, &row.Description, &row.Type, &row.PolygonGeoJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLocationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Ancestors returns the spatial parents of a location ordered district -> area.
// Containment is computed against a point guaranteed to lie on the location's
// surface (ST_PointOnSurface) so it works for both polygon (area-in-district)
// and point (poi-in-area) children; a centroid can fall outside a concave or
// multipolygon child and resolve the wrong parent.
func (r *LocationRepository) Ancestors(ctx context.Context, id string) ([]model.Location, error) {
	rows, err := r.db.Query(ctx, `
		WITH t AS (
			SELECT id, type, ST_PointOnSurface(coordinates) AS c
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
// areas, or an area's pois. Containment uses a point guaranteed to lie on the
// child's surface (ST_PointOnSurface) so it works for both polygon
// (area-in-district) and point (poi-in-area) children; a centroid can fall
// outside a concave or multipolygon child and be wrongly excluded. Pois (and
// anything else) have no descendants.
func (r *LocationRepository) Descendants(ctx context.Context, row *LocationDetailRow) ([]model.Location, error) {
	switch row.Type {
	case "district":
		return r.queryDescendants(ctx, `
			SELECT a.id, a.name, 'area'::text,
			       (SELECT li.url FROM location_image li
			        WHERE li.location_id = a.id
			        ORDER BY li.display_order ASC, li.id ASC LIMIT 1) AS thumbnail
			FROM location a
			WHERE a.type = 'area' AND a.status = 'active'
			  AND ST_Within(ST_PointOnSurface(a.coordinates), (SELECT coordinates FROM location WHERE id = $1))
			ORDER BY a.name
		`, row.ID)
	case "area":
		return r.queryDescendants(ctx, `
			SELECT p.id, p.name, 'poi'::text,
			       (SELECT li.url FROM location_image li
			        WHERE li.location_id = p.id
			        ORDER BY li.display_order ASC, li.id ASC LIMIT 1) AS thumbnail
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
		if err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Thumbnail); err != nil {
			return nil, err
		}
		descendants = append(descendants, d)
	}
	return descendants, rows.Err()
}

// Images returns a location's images ordered by display_order (then id).
func (r *LocationRepository) Images(ctx context.Context, id, lang string) ([]model.LocationImage, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT url, %s
		FROM location_image
		WHERE location_id = $1
		ORDER BY display_order ASC, id ASC
	`, localized("$2", "description_indo", "description")), id, lang)
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
