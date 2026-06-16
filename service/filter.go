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
func (s *FilterService) Get(ctx context.Context, lang string) (*model.FiltersResponse, error) {
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
		tags = append(tags, model.FilterTag{Name: t.Name, Slug: t.Slug})
	}

	labels := ratingTypeLabels[normLang(lang)]
	cats := make([]model.FilterRatingCategory, 0)
	idx := make(map[string]int) // type → index in cats, preserving query order
	priceTiers := make([]model.FilterPriceTier, 0, 3)

	for _, r := range ratingRows {
		if r.Type == constants.RatingCategoryPriceRank {
			max := int(r.Upper)
			priceTiers = append(priceTiers, model.FilterPriceTier{
				Label: r.Name,
				Min:   int(r.Lower),
				Max:   &max,
			})
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
		cats[i].Options = append(cats[i].Options, model.FilterRatingOption{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			LowerBound:  r.Lower,
			UpperBound:  r.Upper,
		})
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
