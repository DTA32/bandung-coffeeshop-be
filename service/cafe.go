package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
)

var (
	ErrInvalidSort             = errors.New("invalid sort")
	ErrInvalidOrder            = errors.New("invalid order")
	ErrInvalidCoords           = errors.New("invalid query_coords")
	ErrCoordsConflictsWithID   = errors.New("query_coords cannot be combined with query_id")
	ErrQueryTypeWithoutID      = errors.New("query_type requires query_id")
	ErrIDWithoutType           = errors.New("query_id requires query_type")
	ErrDistanceSortNeedsCoords = errors.New("sort=distance requires query_coords")
	ErrInvalidRatingCategory   = errors.New("invalid rating_category_type")
	ErrRatingSlugWithoutType   = errors.New("rating_category_id requires rating_category_type")
	ErrInvalidOpenHour         = errors.New("invalid open_hour")
	ErrInvalidPriceRange       = errors.New("price_min cannot exceed price_max")
	ErrDuplicateRatingType     = errors.New("duplicate rating category in filter")
)

const (
	defaultPageNum   = 1
	defaultPageSize  = 8
	maxPageSize      = 50
	defaultRadiusMax = 3000 // in meters
)

var validSorts = map[string]struct{}{
	constants.SortDefault:    {},
	constants.SortUpdatedAt:  {},
	constants.SortDistance:   {},
	constants.SortRating:     {},
	constants.SortPriceRange: {},
}

var validOrders = map[string]struct{}{
	constants.OrderAsc:  {},
	constants.OrderDesc: {},
}

// Backend-generated, localized labels. Keyed by locale code with an English
// fallback applied via normLang.
var priceRankLabels = map[string]map[int]string{
	constants.LangEnglish: {
		0: "Bandung pricing - affordable for most",
		1: "Riau pricing - slightly higher but still affordable",
		2: "Jakarta pricing",
	},
	constants.LangIndonesian: {
		0: "Harga Bandung - terjangkau untuk kebanyakan orang",
		1: "Harga Riau - sedikit lebih tinggi tapi masih terjangkau",
		2: "Harga Jakarta",
	},
}

var (
	locLabelIn           = map[string]string{constants.LangEnglish: "in ", constants.LangIndonesian: "di "}
	locLabelNear         = map[string]string{constants.LangEnglish: "near ", constants.LangIndonesian: "dekat "}
	locLabelSelectedSpot = map[string]string{constants.LangEnglish: "near Selected Spot", constants.LangIndonesian: "dekat Lokasi Terpilih"}
	priceStartFrom       = map[string]string{constants.LangEnglish: "start from ", constants.LangIndonesian: "mulai dari "}
	priceUpTo            = map[string]string{constants.LangEnglish: "up to ", constants.LangIndonesian: "hingga "}
)

// normLang normalizes an arbitrary locale code to one of the supported codes,
// defaulting to Indonesian.
func normLang(lang string) string {
	if lang == constants.LangEnglish {
		return constants.LangEnglish
	}
	return constants.LangIndonesian
}

type CafeService struct {
	repo *repository.CafeRepository
}

func NewCafeService(repo *repository.CafeRepository) *CafeService {
	return &CafeService{repo: repo}
}

func (s *CafeService) Search(ctx context.Context, req model.CafeSearchRequest) (*model.CafeSearchResponse, error) {
	if err := s.validate(&req); err != nil {
		return nil, err
	}

	var (
		focus  *repository.FocusLocation
		tagRow *repository.Tag
		params repository.CafeSearchParams
		err    error
	)

	if req.QueryID != "" {
		focus, err = s.repo.ResolveFocus(ctx, req.QueryID, req.QueryType, req.Lang)
		if err != nil {
			return nil, err
		}
	}

	// Resolve rating buckets by id → (type, score bounds). A cafe has a single
	// score per category, so two buckets of the same type can never both match;
	// reject that as a bad request rather than silently returning nothing.
	if len(req.RatingIDs) > 0 {
		cats, err := s.repo.RatingCategoriesByIDs(ctx, req.RatingIDs)
		if err != nil {
			return nil, err
		}
		if len(cats) != len(req.RatingIDs) {
			return nil, repository.ErrRatingCategoryNotFound
		}
		seen := make(map[string]struct{}, len(cats))
		for _, rc := range cats {
			if _, dup := seen[rc.Type]; dup {
				return nil, ErrDuplicateRatingType
			}
			seen[rc.Type] = struct{}{}
			params.RatingFilters = append(params.RatingFilters, repository.RatingFilterParam{
				Type:  rc.Type,
				Lower: rc.LowerBound,
				Upper: rc.UpperBound,
			})
		}
	}

	// Tags (AND): resolve each slug; unknown slugs from a multi-select are
	// ignored rather than failing the whole search.
	for _, slug := range req.Tags {
		t, terr := s.repo.TagBySlug(ctx, slug, req.Lang)
		if terr != nil {
			if errors.Is(terr, repository.ErrTagNotFound) {
				continue
			}
			return nil, terr
		}
		params.TagSlugs = append(params.TagSlugs, t.Slug)
		tagRow = t
	}
	// The tag-only search description only applies to a single resolved tag.
	if len(params.TagSlugs) != 1 {
		tagRow = nil
	}

	// Open hours: resolve "now" to the current time in WIB (UTC+7, no DST).
	if req.OpenHour != "" {
		hhmm := req.OpenHour
		if req.OpenHour == "now" {
			hhmm = time.Now().In(time.FixedZone("WIB", 7*3600)).Format("15:04")
		}
		params.OpenHour = &hhmm
	}
	params.PriceMin = req.PriceMin
	params.PriceMax = req.PriceMax

	switch {
	case focus != nil && (focus.Type == constants.LocationTypeArea || focus.Type == constants.LocationTypeDistrict):
		params.Mode = repository.SearchModePolygon
		params.PolygonLocID = focus.ID
		params.FocusLat = &focus.CenterLat
		params.FocusLng = &focus.CenterLng
	case focus != nil && (focus.Type == constants.LocationTypeCafe || focus.Type == constants.LocationTypePOI):
		params.Mode = repository.SearchModeRadius
		params.FocusLat = &focus.CenterLat
		params.FocusLng = &focus.CenterLng
		params.RadiusMax = req.RadiusMax
		params.ExcludeIDs = []string{focus.ID}
	case req.QueryCoords != nil:
		params.Mode = repository.SearchModeRadius
		lat := req.QueryCoords.Lat
		lng := req.QueryCoords.Lng
		params.FocusLat = &lat
		params.FocusLng = &lng
		params.RadiusMax = req.RadiusMax
	default:
		params.Mode = repository.SearchModeGlobal
	}

	params.IsFeatured = req.IsFeatured
	params.Lang = req.Lang
	params.Sort = req.Sort
	params.Order = req.Order
	params.Page = req.Page
	params.Size = req.Size

	rows, total, err := s.repo.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	cafes := make([]model.CafeDetail, 0, len(rows))
	emitDistance := req.QueryCoords != nil
	for _, r := range rows {
		var coords *model.Coordinates
		if r.Lat != nil && r.Lng != nil {
			coords = &model.Coordinates{Lat: *r.Lat, Lng: *r.Lng}
		}
		var distance *int
		if emitDistance {
			distance = r.DistanceM
		}
		cafes = append(cafes, model.CafeDetail{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Coordinates: coords,
			Thumbnail:   r.Thumbnail,
			Area:        r.Area,
			PriceRange:  formatPriceRange(req.Lang, r.PriceRangeMin, r.PriceRangeMax),
			Distance:    distance,
			Remark:      r.Remark,
		})
	}

	locationName, formattedName := formatLocationLabel(req.Lang, focus, req.QueryCoords)
	searchDescription := s.buildSearchDescription(&req, focus, tagRow)

	return &model.CafeSearchResponse{
		Total:                 total,
		LocationName:          locationName,
		FormattedLocationName: formattedName,
		SearchDescription:     searchDescription,
		Locations:             buildBreadcrumb(focus),
		Cafes:                 cafes,
		Page:                  req.Page,
		Size:                  req.Size,
	}, nil
}

// buildBreadcrumb returns the focus location's ancestor chain, outermost to
// innermost and including the focus itself, for district / area / poi focus.
// Cafe / coordinate / global searches return an empty (non-nil) slice.
func buildBreadcrumb(focus *repository.FocusLocation) []model.Location {
	crumbs := []model.Location{}
	if focus == nil {
		return crumbs
	}
	switch focus.Type {
	case constants.LocationTypeArea, constants.LocationTypePOI:
		if focus.DistrictID != nil {
			crumbs = append(crumbs, model.Location{ID: *focus.DistrictID, Name: *focus.DistrictName, Type: constants.LocationTypeDistrict})
		}
		if focus.Type == constants.LocationTypePOI && focus.AreaID != nil {
			crumbs = append(crumbs, model.Location{ID: *focus.AreaID, Name: *focus.AreaName, Type: constants.LocationTypeArea})
		}
		crumbs = append(crumbs, model.Location{ID: focus.ID, Name: focus.Name, Type: focus.Type})
	case constants.LocationTypeDistrict:
		crumbs = append(crumbs, model.Location{ID: focus.ID, Name: focus.Name, Type: focus.Type})
	}
	return crumbs
}

func (s *CafeService) validate(req *model.CafeSearchRequest) error {
	if req.QueryType != "" {
		if _, ok := validLocationTypes[req.QueryType]; !ok {
			return ErrInvalidLocationType
		}
	}
	if req.QueryType != "" && req.QueryID == "" {
		return ErrQueryTypeWithoutID
	}
	if req.QueryID != "" && req.QueryType == "" {
		return ErrIDWithoutType
	}
	if req.QueryCoords != nil && req.QueryID != "" {
		return ErrCoordsConflictsWithID
	}

	if req.RadiusMax == nil {
		if req.QueryCoords != nil || req.QueryType == constants.LocationTypeCafe {
			defaultValue := defaultRadiusMax
			req.RadiusMax = &defaultValue
		} else if req.QueryType == constants.LocationTypePOI {
			defaultValue := 2000
			req.RadiusMax = &defaultValue
		}
	}

	if req.OpenHour != "" && req.OpenHour != "now" {
		if _, err := time.Parse("15:04", req.OpenHour); err != nil {
			return ErrInvalidOpenHour
		}
	}
	if req.PriceMin != nil && req.PriceMax != nil && *req.PriceMin > *req.PriceMax {
		return ErrInvalidPriceRange
	}

	if req.Sort == "" {
		req.Sort = constants.SortDefault
	}
	if _, ok := validSorts[req.Sort]; !ok {
		return ErrInvalidSort
	}
	if req.Order != "" {
		if _, ok := validOrders[req.Order]; !ok {
			return ErrInvalidOrder
		}
	}
	if req.Sort == constants.SortDistance && req.QueryCoords == nil {
		if req.QueryType != constants.LocationTypeCafe && req.QueryType != constants.LocationTypePOI {
			return ErrDistanceSortNeedsCoords
		}
	}

	if req.Page <= 0 {
		req.Page = defaultPageNum
	}
	if req.Size <= 0 {
		req.Size = defaultPageSize
	}
	if req.Size > maxPageSize {
		req.Size = maxPageSize
	}
	return nil
}

func formatLocationLabel(lang string, focus *repository.FocusLocation, coords *model.Coordinates) (string, string) {
	l := normLang(lang)
	if focus != nil {
		switch focus.Type {
		case constants.LocationTypeArea, constants.LocationTypeDistrict:
			return focus.Name, locLabelIn[l] + focus.Name
		case constants.LocationTypeCafe, constants.LocationTypePOI:
			return focus.Name, locLabelNear[l] + focus.Name
		}
	}
	if coords != nil {
		return "", locLabelSelectedSpot[l]
	}
	return "", ""
}

func (s *CafeService) buildSearchDescription(req *model.CafeSearchRequest, focus *repository.FocusLocation, tag *repository.Tag) string {
	tagOnly := tag != nil &&
		focus == nil &&
		req.QueryCoords == nil &&
		len(req.RatingIDs) == 0 &&
		req.OpenHour == "" &&
		req.PriceMin == nil &&
		req.PriceMax == nil &&
		req.IsFeatured == nil
	if tagOnly {
		return tag.Description
	}
	if focus != nil {
		switch focus.Type {
		case constants.LocationTypeArea, constants.LocationTypeDistrict, constants.LocationTypePOI:
			return focus.Description
		}
	}
	return ""
}

func formatPriceRange(lang string, min, max *int) *string {
	l := normLang(lang)
	switch {
	case min != nil && max != nil:
		s := fmt.Sprintf("Rp. %s - Rp. %s", formatThousand(*min), formatThousand(*max))
		return &s
	case min != nil:
		s := fmt.Sprintf("%sRp. %s", priceStartFrom[l], formatThousand(*min))
		return &s
	case max != nil:
		s := fmt.Sprintf("%sRp. %s", priceUpTo[l], formatThousand(*max))
		return &s
	default:
		return nil
	}
}

func formatThousand(v int) string {
	if v%1000 == 0 {
		return strconv.Itoa(v/1000) + "k"
	}
	return strconv.Itoa(v)
}

func (s *CafeService) GetByID(ctx context.Context, locationID, lang string) (*model.CafeDetailResponse, error) {
	row, err := s.repo.CafeByLocationID(ctx, locationID, lang)
	if err != nil {
		return nil, err
	}

	images, err := s.repo.CafeImagesByLocationID(ctx, locationID, lang)
	if err != nil {
		return nil, err
	}

	priceRank, err := s.repo.CafePriceRankByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}

	imgs := make([]model.LocationImage, 0, len(images))
	for _, img := range images {
		imgs = append(imgs, model.LocationImage{URL: img.URL, Description: img.Alt})
	}

	var rank *model.CafeRank
	if priceRank != nil {
		rank = &model.CafeRank{Type: *priceRank, Label: priceRankLabels[normLang(lang)][*priceRank]}
	}

	var desc *string
	if row.Description != "" {
		desc = &row.Description
	}

	// Ancestor chain, outermost to innermost: district then area.
	loc := []model.Location{}
	if row.DistrictID != nil {
		loc = append(loc, model.Location{ID: *row.DistrictID, Name: *row.DistrictName, Type: constants.LocationTypeDistrict})
	}
	if row.AreaID != nil {
		loc = append(loc, model.Location{ID: *row.AreaID, Name: *row.AreaName, Type: constants.LocationTypeArea})
	}

	return &model.CafeDetailResponse{
		ID:          row.ID,
		Name:        row.Name,
		Description: desc,
		Status:      row.Status,
		Images:      imgs,
		Instagram:   row.Instagram,
		OpenHour:    row.OpenHour,
		CloseHour:   row.CloseHour,
		GmapsID:     row.GmapsID,
		Locations:   loc,
		Price: model.CafePrice{
			PriceRangeMin:  row.PriceRangeMin,
			PriceRangeMax:  row.PriceRangeMax,
			CoffeePriceMin: row.CoffeePriceMin,
			CoffeePriceMax: row.CoffeePriceMax,
			SnackPriceMin:  row.SnackPriceMin,
			SnackPriceMax:  row.SnackPriceMax,
			FoodPriceMin:   row.FoodPriceMin,
			FoodPriceMax:   row.FoodPriceMax,
			Rank:           rank,
		},
	}, nil
}

func (s *CafeService) GetReviewByID(ctx context.Context, locationID, lang string) (*model.CafeReviewResponse, error) {
	exists, err := s.repo.CafeExistsByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrCafeNotFound
	}

	reviewRow, err := s.repo.CafeReviewByLocationID(ctx, locationID, lang)
	if err != nil {
		return nil, err
	}
	if reviewRow == nil {
		return &model.CafeReviewResponse{
			Tags:    []model.CafeTag{},
			Ratings: model.RatingsResponse{},
		}, nil
	}

	tagRows, err := s.repo.CafeTagsByLocationID(ctx, locationID, lang)
	if err != nil {
		return nil, err
	}

	ratingRows, err := s.repo.CafeRatingsByLocationID(ctx, locationID, lang)
	if err != nil {
		return nil, err
	}

	tags := make([]model.CafeTag, 0, len(tagRows))
	for _, t := range tagRows {
		tags = append(tags, model.CafeTag{Name: t.Name, Slug: t.Slug})
	}

	ratings := make(model.RatingsResponse)
	for _, r := range ratingRows {
		entry, ok := ratings[r.CategoryType]
		if !ok {
			entry = model.RatingEntry{
				DisplayName: r.TypeLabel,
				Score:       r.Score,
				Description: r.Description,
				Range:       []model.RatingRange{},
			}
		}
		ratingRange := model.RatingRange{
			Name:        r.RangeName,
			Description: r.RangeDesc,
			LowerBound:  r.LowerBound,
			UpperBound:  r.UpperBound,
		}
		if r.Score >= r.LowerBound && r.Score <= r.UpperBound && r.Slug != nil {
			slug := fmt.Sprintf("%s-%s", *r.Slug, r.CategoryType)
			ratingRange.Slug = &slug
		}
		entry.Range = append(entry.Range, ratingRange)
		ratings[r.CategoryType] = entry
	}

	return &model.CafeReviewResponse{
		IsSubjective: reviewRow.IsSubjective,
		OverallScore: reviewRow.OverallScore,
		WFCScore:     reviewRow.WFCScore,
		Tags:         tags,
		Content:      reviewRow.Content,
		VisitedAt:    reviewRow.VisitedAt,
		UpdatedAt:    reviewRow.UpdatedAt,
		Ratings:      ratings,
	}, nil
}

func ParseCoords(s string) (*model.Coordinates, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return nil, ErrInvalidCoords
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return nil, ErrInvalidCoords
	}
	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return nil, ErrInvalidCoords
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return nil, ErrInvalidCoords
	}
	return &model.Coordinates{Lat: lat, Lng: lng}, nil
}
