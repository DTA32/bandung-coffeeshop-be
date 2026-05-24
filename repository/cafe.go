package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrFocusNotFound          = errors.New("focus location not found")
	ErrRatingCategoryNotFound = errors.New("rating category not found")
	ErrTagNotFound            = errors.New("tag not found")
	ErrCafeNotFound           = errors.New("cafe not found")
)

const (
	SearchModeGlobal  = "global"
	SearchModePolygon = "polygon"
	SearchModeRadius  = "radius"
)

type CafeRepository struct {
	db *pgxpool.Pool
}

func NewCafeRepository(db *pgxpool.Pool) *CafeRepository {
	return &CafeRepository{db: db}
}

type FocusLocation struct {
	ID          string
	Name        string
	Type        string
	Description string
	CenterLat   float64
	CenterLng   float64
}

type RatingCategory struct {
	ID         int
	Type       string
	Slug       string
	LowerBound float64
	UpperBound float64
}

type Tag struct {
	ID          int
	Name        string
	Slug        string
	Description string
}

type CafeSearchParams struct {
	Mode         string
	PolygonLocID string
	FocusLat     *float64
	FocusLng     *float64
	RadiusMax    *int
	ExcludeIDs   []string

	TagSlug            string
	RatingCategoryType string
	RatingLowerBound   *float64
	RatingUpperBound   *float64
	IsFeatured         *bool

	Sort  string
	Order string
	Page  int
	Size  int
}

type CafeDetailRow struct {
	ID, Name, Status, Description  string
	GmapsID                        *string
	Instagram, OpenHour, CloseHour *string
	PriceRangeMin, PriceRangeMax   *int
	CoffeePriceMin, CoffeePriceMax *int
	SnackPriceMin, SnackPriceMax   *int
	FoodPriceMin, FoodPriceMax     *int
	AreaID, AreaName               *string
}

type CafeImageRow struct{ URL, Alt string }

type PriceRankRow struct {
	RankType  int
	RankLabel string
}

type ReviewRow struct {
	IsSubjective           bool
	OverallScore, WFCScore *float64
	Content                *string
	VisitedAt              *string
	UpdatedAt              string
}

type CafeTagRow struct {
	Name string
	Slug *string
}

type CafeRatingRow struct {
	CategoryType string
	Score        float64
	Description  string
	RangeName    string
	RangeDesc    string
	LowerBound   float64
	UpperBound   float64
}

func (r *CafeRepository) ResolveFocus(ctx context.Context, id string) (*FocusLocation, error) {
	var f FocusLocation
	err := r.db.QueryRow(ctx, `
		SELECT id, name, type::text, description,
		       ST_Y(ST_Centroid(coordinates)) AS lat,
		       ST_X(ST_Centroid(coordinates)) AS lng
		FROM location
		WHERE id = $1 AND status <> 'deleted'
	`, id).Scan(&f.ID, &f.Name, &f.Type, &f.Description, &f.CenterLat, &f.CenterLng)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFocusNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *CafeRepository) RatingCategoryBySlug(ctx context.Context, categoryType, slug string) (*RatingCategory, error) {
	var rc RatingCategory
	var lb, ub float64
	err := r.db.QueryRow(ctx, `
		SELECT id, type::text, slug, lower_bound, upper_bound
		FROM rating_category
		WHERE type = $1::rating_category_type_enum AND slug = $2
	`, categoryType, slug).Scan(&rc.ID, &rc.Type, &rc.Slug, &lb, &ub)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRatingCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	rc.LowerBound = lb
	rc.UpperBound = ub
	return &rc, nil
}

func (r *CafeRepository) TagBySlug(ctx context.Context, slug string) (*Tag, error) {
	var t Tag
	err := r.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(slug, ''), description
		FROM tag
		WHERE slug = $1
	`, slug).Scan(&t.ID, &t.Name, &t.Slug, &t.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTagNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type CafeSearchRow struct {
	ID            string
	Name          string
	Lat           *float64
	Lng           *float64
	Thumbnail     *string
	Area          *string
	PriceRangeMin *int
	PriceRangeMax *int
	Remark        *string
	DistanceM     *int
}

func (r *CafeRepository) Search(ctx context.Context, p CafeSearchParams) ([]CafeSearchRow, int, error) {
	var sb strings.Builder
	args := make([]any, 0, 12)
	addArg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	var focusPointSQL string
	if p.FocusLat != nil && p.FocusLng != nil {
		lngP := addArg(*p.FocusLng)
		latP := addArg(*p.FocusLat)
		focusPointSQL = fmt.Sprintf("ST_SetSRID(ST_MakePoint(%s, %s), 4326)", lngP, latP)
	} else {
		focusPointSQL = "NULL::geometry"
	}

	sb.WriteString(`SELECT
		l.id,
		l.name,
		ST_Y(l.coordinates) AS lat,
		ST_X(l.coordinates) AS lng,
		(SELECT li.url FROM location_image li
			WHERE li.location_id = l.id
			ORDER BY li.display_order ASC, li.id ASC LIMIT 1) AS thumbnail,
		(SELECT a.name FROM location a
			WHERE a.type = 'area' AND a.status = 'active'
				AND ST_Within(l.coordinates, a.coordinates)
			ORDER BY a.id LIMIT 1) AS area,
		cp.price_range_min,
		cp.price_range_max,
		(SELECT t.name FROM cafe_tag ct
			JOIN tag t ON t.id = ct.tag_id
			WHERE ct.cafe_id = c.id AND ct.visible = TRUE
			ORDER BY t.id LIMIT 1) AS remark,`)

	sb.WriteString(fmt.Sprintf(`
		CASE WHEN %s IS NULL THEN NULL
		     ELSE ROUND(ST_Distance(l.coordinates::geography, %s::geography))::INT
		END AS distance_m,
		COUNT(*) OVER() AS total_count
	`, focusPointSQL, focusPointSQL))

	sb.WriteString(`FROM cafe c
		JOIN location l ON l.id = c.location_id AND l.status = 'active'
		LEFT JOIN cafe_price cp ON cp.cafe_id = c.id`)

	if p.TagSlug != "" {
		slugP := addArg(p.TagSlug)
		sb.WriteString(fmt.Sprintf(`
		JOIN cafe_tag ctf ON ctf.cafe_id = c.id AND ctf.visible = TRUE
		JOIN tag tf ON tf.id = ctf.tag_id AND tf.slug = %s`, slugP))
	}

	if p.RatingCategoryType != "" && p.RatingLowerBound != nil && p.RatingUpperBound != nil {
		typeP := addArg(p.RatingCategoryType)
		lbP := addArg(*p.RatingLowerBound)
		ubP := addArg(*p.RatingUpperBound)
		sb.WriteString(fmt.Sprintf(`
		JOIN cafe_rating cr ON cr.cafe_id = c.id
			AND cr.category_type = %s::rating_category_type_enum
			AND cr.score >= %s AND cr.score <= %s`, typeP, lbP, ubP))
	}

	if p.Sort == constants.SortRating {
		sb.WriteString(`
		LEFT JOIN LATERAL (
			SELECT overall_score FROM cafe_review
			WHERE cafe_id = c.id
			ORDER BY COALESCE(visited_at, created_at::date) DESC, id DESC
			LIMIT 1
		) lr ON TRUE`)
	}

	sb.WriteString(`
		WHERE 1=1`)

	switch p.Mode {
	case SearchModePolygon:
		idP := addArg(p.PolygonLocID)
		sb.WriteString(fmt.Sprintf(`
			AND ST_Within(l.coordinates, (SELECT coordinates FROM location WHERE id = %s))`, idP))
	case SearchModeRadius:
		if p.RadiusMax != nil {
			rP := addArg(*p.RadiusMax)
			sb.WriteString(fmt.Sprintf(`
			AND ST_DWithin(l.coordinates::geography, %s::geography, %s)`, focusPointSQL, rP))
		}
	}

	if p.IsFeatured != nil {
		fP := addArg(*p.IsFeatured)
		sb.WriteString(fmt.Sprintf(`
			AND c.is_featured = %s`, fP))
	}

	if len(p.ExcludeIDs) > 0 {
		idPs := make([]string, len(p.ExcludeIDs))
		for i, id := range p.ExcludeIDs {
			idPs[i] = addArg(id)
		}
		sb.WriteString(fmt.Sprintf(`
			AND l.id NOT IN (%s)`, strings.Join(idPs, ", ")))
	}

	sb.WriteString(`
		ORDER BY `)
	switch p.Sort {
	case constants.SortUpdatedAt:
		if p.Order == constants.OrderAsc {
			sb.WriteString(`c.updated_at ASC`)
		} else {
			sb.WriteString(`c.updated_at DESC`)
		}
	case constants.SortDistance:
		if p.Order == constants.OrderDesc {
			sb.WriteString(`distance_m DESC NULLS LAST`)
		} else {
			sb.WriteString(`distance_m ASC NULLS LAST`)
		}
	case constants.SortRating:
		if p.Order == constants.OrderAsc {
			sb.WriteString(`lr.overall_score ASC NULLS LAST`)
		} else {
			sb.WriteString(`lr.overall_score DESC NULLS LAST`)
		}
	case constants.SortPriceRange:
		if p.Order == constants.OrderDesc {
			sb.WriteString(`cp.price_range_max DESC NULLS LAST`)
		} else {
			sb.WriteString(`cp.price_range_min ASC NULLS LAST`)
		}
	default:
		sb.WriteString(`c.is_featured DESC, c.updated_at DESC, distance_m ASC NULLS LAST`)
	}

	sizeP := addArg(p.Size)
	offsetP := addArg((p.Page - 1) * p.Size)
	sb.WriteString(fmt.Sprintf(` LIMIT %s OFFSET %s`, sizeP, offsetP))

	rows, err := r.db.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var (
		results []CafeSearchRow
		total   int
	)
	for rows.Next() {
		var (
			row  CafeSearchRow
			tcnt int64
		)
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Lat,
			&row.Lng,
			&row.Thumbnail,
			&row.Area,
			&row.PriceRangeMin,
			&row.PriceRangeMax,
			&row.Remark,
			&row.DistanceM,
			&tcnt,
		); err != nil {
			return nil, 0, err
		}
		total = int(tcnt)
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *CafeRepository) CafeByLocationID(ctx context.Context, locationID string) (*CafeDetailRow, error) {
	var row CafeDetailRow
	err := r.db.QueryRow(ctx, `
		SELECT l.id, l.name, l.description, l.status, l.gmaps_id,
		       c.instagram,
		       TO_CHAR(c.open_hour,  'HH24:MI'),
		       TO_CHAR(c.close_hour, 'HH24:MI'),
		       cp.price_range_min, cp.price_range_max,
		       cp.coffee_price_min, cp.coffee_price_max,
		       cp.snack_price_min,  cp.snack_price_max,
		       cp.food_price_min,   cp.food_price_max,
		       area.id, area.name
		FROM location l
		JOIN cafe c ON c.location_id = l.id
		LEFT JOIN cafe_price cp ON cp.cafe_id = c.id
		LEFT JOIN LATERAL (
		    SELECT a.id, a.name, a.gmaps_id
		    FROM location a
		    WHERE a.type = 'area' AND a.status = 'active'
		      AND ST_Within(l.coordinates, a.coordinates)
		    ORDER BY a.id LIMIT 1
		) area ON TRUE
		WHERE l.id = $1 AND l.type = 'cafe'
	`, locationID).Scan(
		&row.ID, &row.Name, &row.Description, &row.Status, &row.GmapsID,
		&row.Instagram, &row.OpenHour, &row.CloseHour,
		&row.PriceRangeMin, &row.PriceRangeMax,
		&row.CoffeePriceMin, &row.CoffeePriceMax,
		&row.SnackPriceMin, &row.SnackPriceMax,
		&row.FoodPriceMin, &row.FoodPriceMax,
		&row.AreaID, &row.AreaName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCafeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CafeRepository) CafeImagesByLocationID(ctx context.Context, locationID string) ([]CafeImageRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT li.url, li.description
		FROM location_image li
		WHERE li.location_id = $1
		ORDER BY li.display_order ASC, li.id ASC
	`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CafeImageRow
	for rows.Next() {
		var row CafeImageRow
		if err := rows.Scan(&row.URL, &row.Alt); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func (r *CafeRepository) CafePriceRankByLocationID(ctx context.Context, locationID string) (*int, error) {
	var priceRank *int
	err := r.db.QueryRow(ctx, `
		WITH ranked AS (
		    SELECT id, lower_bound, upper_bound,
		           (ROW_NUMBER() OVER (ORDER BY lower_bound) - 1)::int AS rank_type
		    FROM rating_category
		    WHERE type = 'price-rank'
		),
		median AS (
		    SELECT (cp.price_range_min + cp.price_range_max) / 2.0 AS mid
		    FROM cafe_price cp
		    JOIN cafe c ON c.id = cp.cafe_id
		    WHERE c.location_id = $1
		)
		SELECT r.rank_type
		FROM ranked r, median m
		WHERE m.mid >= r.lower_bound AND m.mid <= r.upper_bound
		LIMIT 1
	`, locationID).Scan(&priceRank)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return priceRank, nil
}

func (r *CafeRepository) CafeExistsByLocationID(ctx context.Context, locationID string) (bool, error) {
	var dummy int
	err := r.db.QueryRow(ctx, `
		SELECT 1 FROM location WHERE id = $1 AND type = 'cafe'
	`, locationID).Scan(&dummy)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *CafeRepository) CafeReviewByLocationID(ctx context.Context, locationID string) (*ReviewRow, error) {
	var row ReviewRow
	err := r.db.QueryRow(ctx, `
		SELECT cr.is_subjective, cr.overall_score, cr.wfc_score,
		       cr.content, cr.visited_at::text, cr.updated_at::text
		FROM cafe_review cr
		JOIN cafe c ON c.id = cr.cafe_id
		WHERE c.location_id = $1
		ORDER BY COALESCE(cr.visited_at, cr.created_at::date) DESC, cr.id DESC
		LIMIT 1
	`, locationID).Scan(
		&row.IsSubjective, &row.OverallScore, &row.WFCScore,
		&row.Content, &row.VisitedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CafeRepository) CafeTagsByLocationID(ctx context.Context, locationID string) ([]CafeTagRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.name, t.slug
		FROM cafe_tag ct
		JOIN tag t ON t.id = ct.tag_id
		JOIN cafe c ON c.id = ct.cafe_id
		WHERE c.location_id = $1 AND ct.visible = TRUE
		ORDER BY t.id
	`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CafeTagRow
	for rows.Next() {
		var row CafeTagRow
		if err := rows.Scan(&row.Name, &row.Slug); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func (r *CafeRepository) CafeRatingsByLocationID(ctx context.Context, locationID string) ([]CafeRatingRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cr.category_type::text, cr.score,
		       COALESCE(cr.short_description_override, ''),
		       rc.name, rc.short_description, rc.lower_bound, rc.upper_bound
		FROM cafe_rating cr
		JOIN cafe c ON c.id = cr.cafe_id
		JOIN rating_category rc ON rc.type = cr.category_type
		WHERE c.location_id = $1
		ORDER BY cr.category_type, rc.lower_bound
	`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CafeRatingRow
	for rows.Next() {
		var row CafeRatingRow
		if err := rows.Scan(
			&row.CategoryType, &row.Score, &row.Description,
			&row.RangeName, &row.RangeDesc,
			&row.LowerBound, &row.UpperBound,
		); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
