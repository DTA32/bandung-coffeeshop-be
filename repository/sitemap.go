package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SitemapRepository struct {
	db *pgxpool.Pool
}

func NewSitemapRepository(db *pgxpool.Pool) *SitemapRepository {
	return &SitemapRepository{db: db}
}

type SitemapCafeRow struct {
	ID        string
	UpdatedAt string
	Image     *string
}

// SitemapLocationRow is an active district / area / poi with its search focus
// point and the ancestors ResolveFocus would give it.
type SitemapLocationRow struct {
	ID                   string
	Type                 string
	CenterLat, CenterLng float64
	DistrictID, AreaID   *string
}

// Cafes lists every active or closed cafe (both keep a public detail page) with
// its updated_at, formatted like the cafe detail's, and its first image (the
// detail's images[0]), nil when it has none.
func (r *SitemapRepository) Cafes(ctx context.Context) ([]SitemapCafeRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT l.id, TO_CHAR(c.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       (SELECT li.url FROM location_image li
		        WHERE li.location_id = l.id
		        ORDER BY li.display_order ASC, li.id ASC LIMIT 1)
		FROM cafe c
		JOIN location l ON l.id = c.location_id
		WHERE l.status IN ('active', 'closed')
		ORDER BY l.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cafes := []SitemapCafeRow{}
	for rows.Next() {
		var row SitemapCafeRow
		if err := rows.Scan(&row.ID, &row.UpdatedAt, &row.Image); err != nil {
			return nil, err
		}
		cafes = append(cafes, row)
	}
	return cafes, rows.Err()
}

// Locations lists every active district, area and poi. The centroid and the
// district (area / poi) and area (poi) ancestors are resolved exactly as
// ResolveFocus does, so a search focused on the row matches the SRP's.
func (r *SitemapRepository) Locations(ctx context.Context) ([]SitemapLocationRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT l.id, l.type::text,
		       ST_Y(ST_Centroid(l.coordinates)), ST_X(ST_Centroid(l.coordinates)),
		       d.id, a.id
		FROM location l
		LEFT JOIN LATERAL (
		    SELECT d.id FROM location d
		    WHERE d.type = 'district' AND d.status = 'active' AND d.id <> l.id
		      AND ST_Within(ST_PointOnSurface(l.coordinates), d.coordinates)
		    ORDER BY d.id LIMIT 1
		) d ON l.type IN ('area', 'poi')
		LEFT JOIN LATERAL (
		    SELECT a.id FROM location a
		    WHERE a.type = 'area' AND a.status = 'active' AND a.id <> l.id
		      AND ST_Within(ST_PointOnSurface(l.coordinates), a.coordinates)
		    ORDER BY a.id LIMIT 1
		) a ON l.type = 'poi'
		WHERE l.type IN ('district', 'area', 'poi') AND l.status = 'active'
		ORDER BY l.type, l.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locations := []SitemapLocationRow{}
	for rows.Next() {
		var row SitemapLocationRow
		if err := rows.Scan(&row.ID, &row.Type, &row.CenterLat, &row.CenterLng, &row.DistrictID, &row.AreaID); err != nil {
			return nil, err
		}
		locations = append(locations, row)
	}
	return locations, rows.Err()
}
