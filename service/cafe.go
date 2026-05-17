package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
)

const (
	defaultPageNum   = 1
	defaultPageSize  = 8
	maxPageSize      = 50
	defaultRadiusMax = 5000 // in meters
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

var priceRankLabels = map[int]string{
	0: "Bandung pricing - affordable for most",
	1: "Riau pricing - slightly higher but still affordable",
	2: "Jakarta pricing",
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
		focus     *repository.FocusLocation
		ratingCat *repository.RatingCategory
		tagRow    *repository.Tag
		params    repository.CafeSearchParams
		err       error
	)

	if req.QueryID != "" {
		focus, err = s.repo.ResolveFocus(ctx, req.QueryID)
		if err != nil {
			return nil, err
		}
	}

	if req.RatingCategorySlug != "" {
		ratingCat, err = s.repo.RatingCategoryBySlug(ctx, req.RatingCategoryType, req.RatingCategorySlug)
		if err != nil {
			return nil, err
		}
		params.RatingCategoryType = ratingCat.Type
		params.RatingLowerBound = &ratingCat.LowerBound
		params.RatingUpperBound = &ratingCat.UpperBound
	}

	if req.Tag != "" {
		tagRow, err = s.repo.TagBySlug(ctx, req.Tag)
		if err != nil {
			return nil, err
		}
		params.TagSlug = tagRow.Slug
	}

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
			Coordinates: coords,
			Thumbnail:   r.Thumbnail,
			Area:        r.Area,
			PriceRange:  formatPriceRange(r.PriceRangeMin, r.PriceRangeMax),
			Distance:    distance,
			Remark:      r.Remark,
		})
	}

	locationName, formattedName := formatLocationLabel(focus, req.QueryCoords)
	searchDescription := s.buildSearchDescription(&req, focus, tagRow)

	return &model.CafeSearchResponse{
		Total:                 total,
		LocationName:          locationName,
		FormattedLocationName: formattedName,
		SearchDescription:     searchDescription,
		Cafes:                 cafes,
		Page:                  req.Page,
		Size:                  req.Size,
	}, nil
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

	switch req.QueryType {
	case constants.LocationTypeCafe, constants.LocationTypePOI:
		if req.RadiusMax == nil {
			defaultValue := defaultRadiusMax
			req.RadiusMax = &defaultValue
		}
	}
	if req.QueryCoords != nil && req.RadiusMax == nil {
		// just discovered this golang quirk
		defaultValue := defaultRadiusMax
		req.RadiusMax = &defaultValue
	}

	if req.RatingCategoryType != "" {
		if _, ok := validRatingCategories[req.RatingCategoryType]; !ok {
			return ErrInvalidRatingCategory
		}
	}
	if req.RatingCategorySlug != "" && req.RatingCategoryType == "" {
		return ErrRatingSlugWithoutType
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

func formatLocationLabel(focus *repository.FocusLocation, coords *model.Coordinates) (string, string) {
	if focus != nil {
		switch focus.Type {
		case constants.LocationTypeArea, constants.LocationTypeDistrict:
			return focus.Name, "in " + focus.Name
		case constants.LocationTypeCafe, constants.LocationTypePOI:
			return focus.Name, "near " + focus.Name
		}
	}
	if coords != nil {
		return "", "near Selected Spot"
	}
	return "", ""
}

func (s *CafeService) buildSearchDescription(req *model.CafeSearchRequest, focus *repository.FocusLocation, tag *repository.Tag) string {
	tagOnly := tag != nil &&
		focus == nil &&
		req.QueryCoords == nil &&
		req.RatingCategorySlug == "" &&
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

func formatPriceRange(min, max *int) *string {
	switch {
	case min != nil && max != nil:
		s := fmt.Sprintf("Rp. %s - Rp. %s", formatThousand(*min), formatThousand(*max))
		return &s
	case min != nil:
		s := fmt.Sprintf("start from Rp. %s", formatThousand(*min))
		return &s
	case max != nil:
		s := fmt.Sprintf("up to Rp. %s", formatThousand(*max))
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

func (s *CafeService) GetByID(ctx context.Context, locationID string) (*model.CafeDetailResponse, error) {
	row, err := s.repo.CafeByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}

	images, err := s.repo.CafeImagesByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}

	priceRank, err := s.repo.CafePriceRankByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}

	imgs := make([]model.CafeImage, 0, len(images))
	for _, img := range images {
		imgs = append(imgs, model.CafeImage{URL: img.URL, Alt: img.Alt})
	}

	var rank *model.CafeRank
	if priceRank != nil {
		rank = &model.CafeRank{Type: *priceRank, Label: priceRankLabels[*priceRank]}
	}

	var desc *string
	if row.Description != "" {
		desc = &row.Description
	}

	var loc *model.CafeLocation
	if row.AreaID != nil {
		loc = &model.CafeLocation{ID: *row.AreaID, Name: *row.AreaName}
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
		Location:    loc,
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

func (s *CafeService) GetReviewByID(ctx context.Context, locationID string) (*model.CafeReviewResponse, error) {
	exists, err := s.repo.CafeExistsByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrCafeNotFound
	}

	reviewRow, err := s.repo.CafeReviewByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}
	if reviewRow == nil {
		return &model.CafeReviewResponse{
			Tags:    []model.CafeTag{},
			Ratings: model.RatingsResponse{},
		}, nil
	}

	tagRows, err := s.repo.CafeTagsByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}

	ratingRows, err := s.repo.CafeRatingsByLocationID(ctx, locationID)
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
				Score:       r.Score,
				Description: r.Description,
				Range:       []model.RatingRange{},
			}
		}
		entry.Range = append(entry.Range, model.RatingRange{
			Name:        r.RangeName,
			Description: r.RangeDesc,
			LowerBound:  r.LowerBound,
			UpperBound:  r.UpperBound,
		})
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
