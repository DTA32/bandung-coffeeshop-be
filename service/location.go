package service

import (
	"context"
	"errors"
	"strings"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
)

var ErrInvalidLocationType = errors.New("invalid location type")

var validLocationTypes = map[string]struct{}{
	constants.LocationTypeCafe:     {},
	constants.LocationTypePOI:      {},
	constants.LocationTypeArea:     {},
	constants.LocationTypeDistrict: {},
}

var validRatingCategories = map[string]struct{}{
	constants.RatingCategoryPriceRank:  {},
	constants.RatingCategoryVibe:       {},
	constants.RatingCategoryNoise:      {},
	constants.RatingCategoryWifi:       {},
	constants.RatingCategoryMeals:      {},
	constants.RatingCategoryAtmosphere: {},
	constants.RatingCategoryParking:    {},
}

type LocationService struct {
	repo *repository.LocationRepository
}

func NewLocationService(repo *repository.LocationRepository) *LocationService {
	return &LocationService{repo: repo}
}

func (s *LocationService) Quicksearch(ctx context.Context, q, locType string) ([]model.QuicksearchResult, error) {
	q = strings.TrimSpace(strings.ToLower(q))
	if len(q) < 2 {
		return []model.QuicksearchResult{}, nil
	}
	if locType != "" {
		if _, ok := validLocationTypes[locType]; !ok {
			return nil, ErrInvalidLocationType
		}
	}
	return s.repo.Quicksearch(ctx, q, locType)
}
