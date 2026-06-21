package service

import (
	"context"
	"errors"
	"strings"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
)

var ErrInvalidSearchType = errors.New("invalid search type")

const (
	// quicksearchLimit caps a single-source search (type=location/filter/specific).
	quicksearchLimit = 10
	// quicksearchSplitLimit caps each source when both are searched (type=all),
	// so the combined dropdown stays balanced (≤5 locations + ≤5 filters).
	quicksearchSplitLimit = 5
)

type QuicksearchService struct {
	repo *repository.QuicksearchRepository
}

func NewQuicksearchService(repo *repository.QuicksearchRepository) *QuicksearchService {
	return &QuicksearchService{repo: repo}
}

// resolveSearchType maps the quicksearch `type` param to what to query: whether
// to include locations, whether to include filters, and the location type to
// constrain locations to ("" = all). Beyond the specific location types it
// accepts the higher-level groups all/location/filter ("" defaults to all).
func resolveSearchType(searchType string) (includeLocations, includeFilters bool, locType string, err error) {
	switch searchType {
	case "", constants.QuicksearchTypeAll:
		return true, true, "", nil
	case constants.QuicksearchTypeLocation:
		return true, false, "", nil
	case constants.QuicksearchTypeFilter:
		return false, true, "", nil
	}
	if _, ok := validLocationTypes[searchType]; ok {
		return true, false, searchType, nil
	}
	return false, false, "", ErrInvalidSearchType
}

// Quicksearch is the typeahead over locations and SRP-eligible filters. The
// `searchType` selects the sources (see resolveSearchType); locations come
// first, then filters. Queries shorter than 2 chars return an empty list.
func (s *QuicksearchService) Quicksearch(ctx context.Context, q, searchType, lang string) ([]model.QuicksearchResult, error) {
	q = strings.TrimSpace(strings.ToLower(q))
	if len(q) < 2 {
		return []model.QuicksearchResult{}, nil
	}
	includeLocations, includeFilters, locType, err := resolveSearchType(searchType)
	if err != nil {
		return nil, err
	}

	// When both sources are searched (type=all), split the budget so neither
	// crowds the other out; a single-source search gets the full limit.
	limit := quicksearchLimit
	if includeLocations && includeFilters {
		limit = quicksearchSplitLimit
	}

	results := []model.QuicksearchResult{}
	if includeLocations {
		locs, err := s.repo.Locations(ctx, q, locType, limit)
		if err != nil {
			return nil, err
		}
		results = append(results, locs...)
	}
	if includeFilters {
		filters, err := s.repo.Filters(ctx, q, lang, limit)
		if err != nil {
			return nil, err
		}
		results = append(results, filters...)
	}
	return results, nil
}
