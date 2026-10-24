package service

import (
	"context"
	"sync"
	"time"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"golang.org/x/sync/errgroup"
)

const (
	// sitemapTTL is how long a built sitemap is served before it's rebuilt.
	sitemapTTL = 15 * time.Minute
	// sitemapConcurrency caps the parallel count queries during a build.
	sitemapConcurrency = 4
)

type SitemapService struct {
	repo    *repository.SitemapRepository
	cafes   *repository.CafeRepository
	filters *FilterService

	mu      sync.Mutex
	cached  *model.SitemapResponse
	builtAt time.Time
}

func NewSitemapService(repo *repository.SitemapRepository, cafes *repository.CafeRepository, filters *FilterService) *SitemapService {
	return &SitemapService{repo: repo, cafes: cafes, filters: filters}
}

// srpFilter is one SRP-eligible filter value and the search params it sets.
type srpFilter struct {
	filter model.SitemapFilter
	apply  func(p *repository.CafeSearchParams)
}

// srpLocation is a location with a complete district → area → poi chain and
// the search focus that chain's SRP uses.
type srpLocation struct {
	chain []string
	apply func(p *repository.CafeSearchParams)
}

// Get returns the sitemap facts, rebuilding them at most every sitemapTTL.
// Concurrent callers wait for a running build instead of starting their own.
func (s *SitemapService) Get(ctx context.Context) (*model.SitemapResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil && time.Since(s.builtAt) < sitemapTTL {
		return s.cached, nil
	}
	res, err := s.build(ctx)
	if err != nil {
		return nil, err
	}
	s.cached, s.builtAt = res, time.Now()
	return res, nil
}

func (s *SitemapService) build(ctx context.Context) (*model.SitemapResponse, error) {
	cafeRows, err := s.repo.Cafes(ctx)
	if err != nil {
		return nil, err
	}
	cafes := make([]model.SitemapCafe, 0, len(cafeRows))
	for _, c := range cafeRows {
		cafes = append(cafes, model.SitemapCafe{ID: c.ID, UpdatedAt: c.UpdatedAt})
	}

	filters, err := s.srpFilters(ctx)
	if err != nil {
		return nil, err
	}
	locations, err := s.srpLocations(ctx)
	if err != nil {
		return nil, err
	}

	// Filter-only and location-only SRPs first; a location × filter page can
	// only match cafes both of its parents match, so it's skipped when either
	// parent is empty.
	filterOnly, err := s.countAll(ctx, nil, filters)
	if err != nil {
		return nil, err
	}
	locationOnly, err := s.countAll(ctx, locations, []srpFilter{{}})
	if err != nil {
		return nil, err
	}

	srps := make([]model.SitemapSRP, 0)
	srps = appendNonEmpty(srps, filterOnly)
	srps = appendNonEmpty(srps, locationOnly)

	var liveFilters []srpFilter
	for i, f := range filters {
		if filterOnly[i].Count > 0 {
			liveFilters = append(liveFilters, f)
		}
	}
	var liveLocations []srpLocation
	for i, l := range locations {
		if locationOnly[i].Count > 0 {
			liveLocations = append(liveLocations, l)
		}
	}
	if len(liveLocations) > 0 && len(liveFilters) > 0 {
		combos, err := s.countAll(ctx, liveLocations, liveFilters)
		if err != nil {
			return nil, err
		}
		srps = appendNonEmpty(srps, combos)
	}

	return &model.SitemapResponse{Cafes: cafes, SRPs: srps}, nil
}

// srpFilters lists the SRP-eligible filter values (those with a slug), built
// from /v1/filters so the slugs and bounds are exactly what the FE resolves.
func (s *SitemapService) srpFilters(ctx context.Context) ([]srpFilter, error) {
	opts, err := s.filters.Get(ctx, constants.DefaultLang, false)
	if err != nil {
		return nil, err
	}
	var filters []srpFilter
	for _, t := range opts.Tags {
		if t.Slug == "" {
			continue
		}
		slug := t.Slug
		filters = append(filters, srpFilter{
			filter: model.SitemapFilter{Type: "tag", Slug: slug},
			apply:  func(p *repository.CafeSearchParams) { p.TagSlugs = []string{slug} },
		})
	}
	for _, t := range opts.PriceTiers {
		if t.Slug == "" {
			continue
		}
		tier := t
		filters = append(filters, srpFilter{
			filter: model.SitemapFilter{Type: "price", Slug: tier.Slug},
			apply: func(p *repository.CafeSearchParams) {
				p.PriceMin = &tier.Min
				p.PriceMax = tier.Max
			},
		})
	}
	for _, c := range opts.RatingCategories {
		for _, o := range c.Options {
			if o.Slug == "" {
				continue
			}
			rf := repository.RatingFilterParam{Type: c.Type, Lower: o.LowerBound, Upper: o.UpperBound}
			filters = append(filters, srpFilter{
				filter: model.SitemapFilter{Type: "rating", Slug: o.Slug},
				apply:  func(p *repository.CafeSearchParams) { p.RatingFilters = []repository.RatingFilterParam{rf} },
			})
		}
	}
	return filters, nil
}

// srpLocations lists the locations an SRP path can name. The path's depth is
// its location type (district / area / poi), so a location whose ancestors
// don't fill the chain (e.g. a poi outside every area) has no SRP. The focus
// mirrors CafeService.Search: polygon for district / area, a poiRadiusMax
// radius around a poi that excludes the poi itself.
func (s *SitemapService) srpLocations(ctx context.Context) ([]srpLocation, error) {
	rows, err := s.repo.Locations(ctx)
	if err != nil {
		return nil, err
	}
	var locations []srpLocation
	for _, r := range rows {
		row := r
		var chain []string
		switch {
		case row.Type == constants.LocationTypeDistrict:
			chain = []string{row.ID}
		case row.Type == constants.LocationTypeArea && row.DistrictID != nil:
			chain = []string{*row.DistrictID, row.ID}
		case row.Type == constants.LocationTypePOI && row.DistrictID != nil && row.AreaID != nil:
			chain = []string{*row.DistrictID, *row.AreaID, row.ID}
		default:
			continue
		}
		apply := func(p *repository.CafeSearchParams) {
			p.Mode = repository.SearchModePolygon
			p.PolygonLocID = row.ID
		}
		if row.Type == constants.LocationTypePOI {
			radius := poiRadiusMax
			apply = func(p *repository.CafeSearchParams) {
				p.Mode = repository.SearchModeRadius
				p.FocusLat = &row.CenterLat
				p.FocusLng = &row.CenterLng
				p.RadiusMax = &radius
				p.ExcludeIDs = []string{row.ID}
			}
		}
		locations = append(locations, srpLocation{chain: chain, apply: apply})
	}
	return locations, nil
}

// countAll counts every location × filter pair, in order (locations outer).
// A nil locations slice means "no location"; a zero srpFilter means "no
// filter".
func (s *SitemapService) countAll(ctx context.Context, locations []srpLocation, filters []srpFilter) ([]model.SitemapSRP, error) {
	if locations == nil {
		locations = []srpLocation{{chain: []string{}}}
	}
	out := make([]model.SitemapSRP, len(locations)*len(filters))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(sitemapConcurrency)
	for i, loc := range locations {
		for j, f := range filters {
			idx := i*len(filters) + j
			g.Go(func() error {
				p := repository.CafeSearchParams{Mode: repository.SearchModeGlobal, Status: constants.LocationStatusActive}
				if loc.apply != nil {
					loc.apply(&p)
				}
				if f.apply != nil {
					f.apply(&p)
				}
				count, lastModified, err := s.cafes.SearchStats(gctx, p)
				if err != nil {
					return err
				}
				srp := model.SitemapSRP{Locations: loc.chain, Count: count}
				if f.apply != nil {
					filter := f.filter
					srp.Filter = &filter
				}
				if lastModified != nil {
					srp.LastModified = *lastModified
				}
				out[idx] = srp
				return nil
			})
		}
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

func appendNonEmpty(dst, src []model.SitemapSRP) []model.SitemapSRP {
	for _, s := range src {
		if s.Count > 0 {
			dst = append(dst, s)
		}
	}
	return dst
}
