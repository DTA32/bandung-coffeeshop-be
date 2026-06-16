package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
)

var ErrInvalidLocationType = errors.New("invalid location type")

var ErrLocationIsCafe = errors.New("location is a cafe")

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

// GetByID assembles a single location's detail. Cafes are rejected with
// ErrLocationIsCafe so callers use the dedicated cafe endpoint instead.
func (s *LocationService) GetByID(ctx context.Context, id, lang string) (*model.LocationDetail, error) {
	row, err := s.repo.GetByID(ctx, id, lang)
	if err != nil {
		return nil, err
	}
	if row.Type == constants.LocationTypeCafe {
		return nil, ErrLocationIsCafe
	}

	ancestors, err := s.repo.Ancestors(ctx, id)
	if err != nil {
		return nil, err
	}
	descendants, err := s.repo.Descendants(ctx, row)
	if err != nil {
		return nil, err
	}
	images, err := s.repo.Images(ctx, id, lang)
	if err != nil {
		return nil, err
	}

	// map always shown on districts or POI, and will be shown when location have no images
	showMap := row.Type == constants.LocationTypeDistrict || (row.Type != constants.LocationTypePOI && len(images) == 0)

	return &model.LocationDetail{
		ID:              row.ID,
		Name:            row.Name,
		Description:     row.Description,
		Type:            row.Type,
		Ancestors:       ancestors,
		Descendants:     descendants,
		Images:          images,
		ShowWelcomeText: row.Type == constants.LocationTypeArea,
		ShowMap:         showMap,
		Polygon:         polygonJSON(row.PolygonGeoJSON),
	}, nil
}

// ListDistricts is the no-id fallback: every district with its flat descendants
// (areas + pois) and images.
func (s *LocationService) ListDistricts(ctx context.Context, lang string) ([]model.LocationDetail, error) {
	districts, err := s.repo.Districts(ctx)
	if err != nil {
		return nil, err
	}

	summaries := make([]model.LocationDetail, 0, len(districts))
	for i := range districts {
		d := districts[i]
		descendants, err := s.repo.Descendants(ctx, &d)
		if err != nil {
			return nil, err
		}
		images, err := s.repo.Images(ctx, d.ID, lang)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, model.LocationDetail{
			ID:          d.ID,
			Name:        d.Name,
			Type:        d.Type,
			Ancestors:   []model.Location{},
			Descendants: descendants,
			Images:      images,
		})
	}
	return summaries, nil
}

// polygonJSON wraps a raw GeoJSON string so it embeds as a JSON object; a nil
// geometry marshals to JSON null.
func polygonJSON(geojson *string) json.RawMessage {
	if geojson == nil {
		return nil
	}
	return json.RawMessage(*geojson)
}
