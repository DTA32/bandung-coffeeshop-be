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

// localized builds a SQL expression returning the Indonesian column (indoCol)
// when the bound lang placeholder resolves to 'id' and is non-empty, otherwise
// the English baseline (baseCol). langArg is a positional bind placeholder such
// as "$2". The result can be NULL if baseCol is NULL; wrap in COALESCE(..., ”)
// when scanning into a non-pointer string.
func localized(langArg, indoCol, baseCol string) string {
	return fmt.Sprintf("COALESCE(NULLIF(CASE WHEN %s = '%s' THEN %s END, ''), %s)",
		langArg, constants.LangIndonesian, indoCol, baseCol)
}

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
	// Ancestors resolved by containment for breadcrumb building.
	DistrictID, DistrictName *string // containing district (area / poi focus)
	AreaID, AreaName         *string // containing area (poi focus)
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

	TagSlugs      []string
	RatingFilters []RatingFilterParam
	OpenHour      *string
	PriceMin      *int
	PriceMax      *int
	IsFeatured    *bool

	Lang  string
	Sort  string
	Order string
	Page  int
	Size  int
}

// RatingFilterParam is a resolved rating-category bucket: the category type and
// the score bounds the cafe's score must fall within.
type RatingFilterParam struct {
	Type         string
	Lower, Upper float64
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
	DistrictID, DistrictName       *string
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
	TypeLabel    string
	Score        float64
	Description  string
	RangeName    string
	RangeDesc    string
	LowerBound   float64
	UpperBound   float64
}

func (r *CafeRepository) ResolveFocus(ctx context.Context, id, queryType, lang string) (*FocusLocation, error) {
	// Resolve the focus location, plus its containing district (area / poi) and
	// area (poi only) via PostGIS containment so the service can build a
	// breadcrumb. Cafe focus is intentionally excluded from the LATERAL gates
	// (no breadcrumb for single-cafe searches).
	var f FocusLocation
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT l.id, l.name, l.type::text, %s,`, localized("$3", "l.description_indo", "l.description"))+`
		       ST_Y(ST_Centroid(l.coordinates)) AS lat,
		       ST_X(ST_Centroid(l.coordinates)) AS lng,
		       d.id, d.name, a.id, a.name
		FROM location l
		LEFT JOIN LATERAL (
		    SELECT d.id, d.name FROM location d
		    WHERE d.type = 'district' AND d.status = 'active' AND d.id <> l.id
		      AND ST_Within(ST_PointOnSurface(l.coordinates), d.coordinates)
		    ORDER BY d.id LIMIT 1
		) d ON l.type IN ('area', 'poi')
		LEFT JOIN LATERAL (
		    SELECT a.id, a.name FROM location a
		    WHERE a.type = 'area' AND a.status = 'active' AND a.id <> l.id
		      AND ST_Within(ST_PointOnSurface(l.coordinates), a.coordinates)
		      -- Nearest-area fallback (commented out; strict containment for now):
		      -- ORDER BY ST_Distance(l.coordinates::geography, a.coordinates::geography) LIMIT 1
		    ORDER BY a.id LIMIT 1
		) a ON l.type = 'poi'
		WHERE l.id = $1 AND l.type = $2 AND l.status <> 'deleted'
	`, id, queryType, lang).Scan(
		&f.ID, &f.Name, &f.Type, &f.Description, &f.CenterLat, &f.CenterLng,
		&f.DistrictID, &f.DistrictName, &f.AreaID, &f.AreaName,
	)
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
		WHERE type = $1 AND slug = $2
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

// RatingCategoriesByIDs resolves a set of rating_category bucket ids to their
// type and score bounds. Ids not found are simply absent from the result, so
// the caller can detect unknown ids by comparing lengths.
func (r *CafeRepository) RatingCategoriesByIDs(ctx context.Context, ids []int) ([]RatingCategory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, type::text, COALESCE(slug, ''), lower_bound, upper_bound
		FROM rating_category
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RatingCategory
	for rows.Next() {
		var rc RatingCategory
		if err := rows.Scan(&rc.ID, &rc.Type, &rc.Slug, &rc.LowerBound, &rc.UpperBound); err != nil {
			return nil, err
		}
		results = append(results, rc)
	}
	return results, rows.Err()
}

func (r *CafeRepository) TagBySlug(ctx context.Context, slug, lang string) (*Tag, error) {
	var t Tag
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT id, %s, COALESCE(slug, ''), %s
		FROM tag
		WHERE slug = $1
	`, localized("$2", "name_indo", "name"), localized("$2", "description_indo", "description")),
		slug, lang).Scan(&t.ID, &t.Name, &t.Slug, &t.Description)
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
	Description   string
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

	langP := addArg(p.Lang)

	sb.WriteString(fmt.Sprintf(`SELECT
		l.id,
		l.name,
		%s,
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
		(SELECT %s FROM cafe_tag ct
			JOIN tag t ON t.id = ct.tag_id
			WHERE ct.cafe_id = c.id AND ct.visible = TRUE
			ORDER BY t.id LIMIT 1) AS remark,`,
		localized(langP, "l.description_indo", "l.description"),
		localized(langP, "t.name_indo", "t.name")))

	sb.WriteString(fmt.Sprintf(`
		CASE WHEN %s IS NULL THEN NULL
		     ELSE ROUND(ST_Distance(l.coordinates::geography, %s::geography))::INT
		END AS distance_m,
		COUNT(*) OVER() AS total_count
	`, focusPointSQL, focusPointSQL))

	sb.WriteString(`FROM cafe c
		JOIN location l ON l.id = c.location_id AND l.status = 'active'
		LEFT JOIN cafe_price cp ON cp.cafe_id = c.id`)

	// One aliased join per selected rating bucket. cafe_rating is UNIQUE on
	// (cafe_id, category_type), so each join matches at most one row — no row
	// fan-out, and AND across categories is the natural intersection.
	for i, rf := range p.RatingFilters {
		alias := fmt.Sprintf("cr%d", i)
		typeP := addArg(rf.Type)
		lbP := addArg(rf.Lower)
		ubP := addArg(rf.Upper)
		sb.WriteString(fmt.Sprintf(`
		JOIN cafe_rating %s ON %s.cafe_id = c.id
			AND %s.category_type = %s
			AND %s.score >= %s AND %s.score <= %s`,
			alias, alias, alias, typeP, alias, lbP, alias, ubP))
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

	// Tags (AND): cafe must carry every selected slug. Expressed as a subquery
	// rather than joins so it can't fan out rows and break COUNT(*) OVER().
	if len(p.TagSlugs) > 0 {
		tagsP := addArg(p.TagSlugs)
		nP := addArg(len(p.TagSlugs))
		sb.WriteString(fmt.Sprintf(`
			AND c.id IN (
				SELECT ct.cafe_id FROM cafe_tag ct
				JOIN tag t ON t.id = ct.tag_id
				WHERE ct.visible = TRUE AND t.slug = ANY(%s)
				GROUP BY ct.cafe_id HAVING COUNT(DISTINCT t.slug) = %s)`, tagsP, nP))
	}

	// Price (overlap): the cafe's [min,max] band intersects the selected
	// window. Each bound is independent; NULL price bands are excluded once a
	// bound is active (NULL comparisons yield false).
	if p.PriceMin != nil {
		pMinP := addArg(*p.PriceMin)
		sb.WriteString(fmt.Sprintf(`
			AND cp.price_range_max >= %s`, pMinP))
	}
	if p.PriceMax != nil {
		pMaxP := addArg(*p.PriceMax)
		sb.WriteString(fmt.Sprintf(`
			AND cp.price_range_min <= %s`, pMaxP))
	}

	// Open hours: cafe is open at the given time. Handles overnight ranges
	// (close < open) and excludes cafes with unknown (NULL) hours.
	if p.OpenHour != nil {
		ohP := addArg(*p.OpenHour)
		sb.WriteString(fmt.Sprintf(`
			AND c.open_hour IS NOT NULL AND c.close_hour IS NOT NULL AND (
				(c.close_hour >= c.open_hour AND %s::time >= c.open_hour AND %s::time <= c.close_hour)
				OR (c.close_hour < c.open_hour AND (%s::time >= c.open_hour OR %s::time <= c.close_hour)))`,
			ohP, ohP, ohP, ohP))
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
		if p.Mode == SearchModeRadius {
			sb.WriteString(`c.is_featured DESC, distance_m ASC NULLS LAST, c.updated_at DESC`)
		} else {
			sb.WriteString(`c.is_featured DESC, c.updated_at DESC, distance_m ASC NULLS LAST`)
		}
	}

	// Sorting tiebreaker
	sb.WriteString(`, l.id ASC`)

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
			&row.Description,
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

func (r *CafeRepository) CafeByLocationID(ctx context.Context, locationID, lang string) (*CafeDetailRow, error) {
	var row CafeDetailRow
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT l.id, l.name, %s, l.status, l.gmaps_id,`, localized("$2", "l.description_indo", "l.description"))+`
		       c.instagram,
		       TO_CHAR(c.open_hour,  'HH24:MI'),
		       TO_CHAR(c.close_hour, 'HH24:MI'),
		       cp.price_range_min, cp.price_range_max,
		       cp.coffee_price_min, cp.coffee_price_max,
		       cp.snack_price_min,  cp.snack_price_max,
		       cp.food_price_min,   cp.food_price_max,
		       area.id, area.name, district.id, district.name
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
		LEFT JOIN LATERAL (
		    SELECT d.id, d.name
		    FROM location d
		    WHERE d.type = 'district' AND d.status = 'active'
		      AND ST_Within(l.coordinates, d.coordinates)
		    ORDER BY d.id LIMIT 1
		) district ON TRUE
		WHERE l.id = $1 AND l.type = 'cafe'
	`, locationID, lang).Scan(
		&row.ID, &row.Name, &row.Description, &row.Status, &row.GmapsID,
		&row.Instagram, &row.OpenHour, &row.CloseHour,
		&row.PriceRangeMin, &row.PriceRangeMax,
		&row.CoffeePriceMin, &row.CoffeePriceMax,
		&row.SnackPriceMin, &row.SnackPriceMax,
		&row.FoodPriceMin, &row.FoodPriceMax,
		&row.AreaID, &row.AreaName, &row.DistrictID, &row.DistrictName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCafeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CafeRepository) CafeImagesByLocationID(ctx context.Context, locationID, lang string) ([]CafeImageRow, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT li.url, %s
		FROM location_image li
		WHERE li.location_id = $1
		ORDER BY li.display_order ASC, li.id ASC
	`, localized("$2", "li.description_indo", "li.description")), locationID, lang)
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

func (r *CafeRepository) CafeReviewByLocationID(ctx context.Context, locationID, lang string) (*ReviewRow, error) {
	var row ReviewRow
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT cr.is_subjective, cr.overall_score, cr.wfc_score,
		       %s, cr.visited_at::text, cr.updated_at::text
		FROM cafe_review cr
		JOIN cafe c ON c.id = cr.cafe_id
		WHERE c.location_id = $1
		ORDER BY COALESCE(cr.visited_at, cr.created_at::date) DESC, cr.id DESC
		LIMIT 1
	`, localized("$2", "cr.content_indo", "cr.content")), locationID, lang).Scan(
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

func (r *CafeRepository) CafeTagsByLocationID(ctx context.Context, locationID, lang string) ([]CafeTagRow, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s, t.slug
		FROM cafe_tag ct
		JOIN tag t ON t.id = ct.tag_id
		JOIN cafe c ON c.id = ct.cafe_id
		WHERE c.location_id = $1 AND ct.visible = TRUE
		ORDER BY t.id
	`, localized("$2", "t.name_indo", "t.name")), locationID, lang)
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

func (r *CafeRepository) CafeRatingsByLocationID(ctx context.Context, locationID, lang string) ([]CafeRatingRow, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT cr.category_type::text, COALESCE(%s, ''), cr.score,
		       COALESCE(%s, ''),
		       %s, %s, rc.lower_bound, rc.upper_bound
		FROM cafe_rating cr
		JOIN cafe c ON c.id = cr.cafe_id
		JOIN rating_category rc ON rc.type = cr.category_type
		LEFT JOIN rating_type_label rtl ON rtl.type = cr.category_type
		WHERE c.location_id = $1
		ORDER BY cr.category_type, rc.lower_bound
	`, localized("$2", "rtl.label_indo", "rtl.label"),
		localized("$2", "cr.short_description_override_indo", "cr.short_description_override"),
		localized("$2", "rc.name_indo", "rc.name"),
		localized("$2", "rc.short_description_indo", "rc.short_description")), locationID, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CafeRatingRow
	for rows.Next() {
		var row CafeRatingRow
		if err := rows.Scan(
			&row.CategoryType, &row.TypeLabel, &row.Score, &row.Description,
			&row.RangeName, &row.RangeDesc,
			&row.LowerBound, &row.UpperBound,
		); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
