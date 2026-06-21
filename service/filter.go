package service

import (
	"context"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
)

// ratingTypeLabels are the localized display names for each rating category
// type, surfaced by the filter metadata endpoint (price-rank is handled by the
// dedicated price filter, so it is omitted from the rating groups).
var ratingTypeLabels = map[string]map[string]string{
	constants.LangEnglish: {
		constants.RatingCategoryVibe:       "Vibe",
		constants.RatingCategoryNoise:      "Noise Level",
		constants.RatingCategoryWifi:       "Wifi Speed",
		constants.RatingCategoryMeals:      "Meals Generosity",
		constants.RatingCategoryAtmosphere: "Atmosphere",
		constants.RatingCategoryParking:    "Parking",
	},
	constants.LangIndonesian: {
		constants.RatingCategoryVibe:       "Suasana",
		constants.RatingCategoryNoise:      "Tingkat Kebisingan",
		constants.RatingCategoryWifi:       "Kecepatan Wifi",
		constants.RatingCategoryMeals:      "Porsi Makanan",
		constants.RatingCategoryAtmosphere: "Atmosfer",
		constants.RatingCategoryParking:    "Parkir",
	},
}

// srpSlug composes the canonical SRP slug for a rating-sourced value by
// appending its category type ("hangout" + "vibe" → "hangout-vibe"). This
// namespaces the slug per dimension, keeping it self-describing and
// collision-proof in pretty URLs. Returns "" when the value has no slug (i.e.
// it isn't SRP-eligible).
func srpSlug(slug, categoryType string) string {
	if slug == "" {
		return ""
	}
	return slug + "-" + categoryType
}

type FilterService struct {
	repo *repository.FilterRepository
}

func NewFilterService(repo *repository.FilterRepository) *FilterService {
	return &FilterService{repo: repo}
}

// Get returns the option lists that power the explore filter modal: every
// selectable tag, the rating categories grouped by type (each with its
// buckets), and the price tiers. The price-rank rating category is surfaced as
// price tiers rather than a rating group, since price has its own filter.
//
// When enrich is true, tags carry their long description and rating buckets
// their long_description — the SRP page renders these as a blurb. The filter
// modal calls it with enrich=false to keep the payload light.
func (s *FilterService) Get(ctx context.Context, lang string, enrich bool) (*model.FiltersResponse, error) {
	tagRows, err := s.repo.Tags(ctx, lang)
	if err != nil {
		return nil, err
	}
	ratingRows, err := s.repo.RatingCategories(ctx, lang)
	if err != nil {
		return nil, err
	}

	tags := make([]model.FilterTag, 0, len(tagRows))
	for _, t := range tagRows {
		tag := model.FilterTag{Name: t.Name, Slug: t.Slug}
		if enrich {
			tag.Description = t.Description
		}
		tags = append(tags, tag)
	}

	labels := ratingTypeLabels[normLang(lang)]
	cats := make([]model.FilterRatingCategory, 0)
	idx := make(map[string]int) // type → index in cats, preserving query order
	priceTiers := make([]model.FilterPriceTier, 0, 3)

	for _, r := range ratingRows {
		if r.Type == constants.RatingCategoryPriceRank {
			max := int(r.Upper)
			tier := model.FilterPriceTier{
				Label: r.Name,
				Slug:  srpSlug(r.Slug, r.Type),
				Min:   int(r.Lower),
				Max:   &max,
			}
			if enrich {
				tier.LongDescription = r.LongDescription
			}
			priceTiers = append(priceTiers, tier)
			continue
		}
		i, ok := idx[r.Type]
		if !ok {
			cats = append(cats, model.FilterRatingCategory{
				Type:        r.Type,
				DisplayName: labels[r.Type],
				Options:     []model.FilterRatingOption{},
			})
			i = len(cats) - 1
			idx[r.Type] = i
		}
		opt := model.FilterRatingOption{
			ID:          r.ID,
			Slug:        srpSlug(r.Slug, r.Type),
			Name:        r.Name,
			Description: r.Description,
			LowerBound:  r.Lower,
			UpperBound:  r.Upper,
		}
		if enrich {
			opt.LongDescription = r.LongDescription
		}
		cats[i].Options = append(cats[i].Options, opt)
	}
	// Buckets are ordered by lower_bound, so the last price tier is the
	// open-ended top tier — drop its upper bound.
	if n := len(priceTiers); n > 0 {
		priceTiers[n-1].Max = nil
	}

	return &model.FiltersResponse{
		Tags:             tags,
		RatingCategories: cats,
		PriceTiers:       priceTiers,
	}, nil
}
