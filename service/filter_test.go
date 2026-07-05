package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSrpSlug(t *testing.T) {
	assert.Equal(t, "", srpSlug("", "vibe"))
	assert.Equal(t, "hangout-vibe", srpSlug("hangout", "vibe"))
	assert.Equal(t, "cheap-price-rank", srpSlug("cheap", "price-rank"))
}

func TestFilterGet_RepoErrors(t *testing.T) {
	boom := errors.New("db error")

	t.Run("tags error propagates", func(t *testing.T) {
		repo := &mockFilterRepo{}
		repo.On("Tags", mock.Anything, "en").Return(nil, boom)
		_, err := NewFilterService(repo).Get(context.Background(), "en", false)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("rating categories error propagates", func(t *testing.T) {
		repo := &mockFilterRepo{}
		repo.On("Tags", mock.Anything, "en").Return([]repository.FilterTagRow{}, nil)
		repo.On("RatingCategories", mock.Anything, "en").Return(nil, boom)
		_, err := NewFilterService(repo).Get(context.Background(), "en", false)
		assert.ErrorIs(t, err, boom)
	})
}

func filterRepoWith(tags []repository.FilterTagRow, ratings []repository.FilterRatingRow) *mockFilterRepo {
	repo := &mockFilterRepo{}
	repo.On("Tags", mock.Anything, mock.Anything).Return(tags, nil)
	repo.On("RatingCategories", mock.Anything, mock.Anything).Return(ratings, nil)
	return repo
}

func TestFilterGet_HappyPath(t *testing.T) {
	tags := []repository.FilterTagRow{{Name: "Cozy", Slug: "cozy", Description: "cozy desc"}}
	ratings := []repository.FilterRatingRow{
		{ID: 1, Type: "vibe", TypeLabel: "Vibe", Slug: "calm", Name: "Calm", Description: "d1", LongDescription: "ld1", Lower: 1, Upper: 2},
		{ID: 2, Type: "vibe", TypeLabel: "Vibe", Slug: "lively", Name: "Lively", Description: "d2", LongDescription: "ld2", Lower: 3, Upper: 5},
		{ID: 3, Type: "noise", TypeLabel: "Noise", Slug: "quiet", Name: "Quiet", Description: "d3", LongDescription: "ld3", Lower: 1, Upper: 2},
		{ID: 4, Type: constants.RatingCategoryPriceRank, TypeLabel: "Price", Slug: "cheap", Name: "Cheap", LongDescription: "lpd1", Lower: 0, Upper: 1},
		{ID: 5, Type: constants.RatingCategoryPriceRank, TypeLabel: "Price", Slug: "mid", Name: "Mid", LongDescription: "lpd2", Lower: 1, Upper: 2},
		{ID: 6, Type: constants.RatingCategoryPriceRank, TypeLabel: "Price", Slug: "premium", Name: "Premium", LongDescription: "lpd3", Lower: 2, Upper: 3},
	}

	t.Run("enrich false omits descriptions", func(t *testing.T) {
		res, err := NewFilterService(filterRepoWith(tags, ratings)).Get(context.Background(), "en", false)
		require.NoError(t, err)

		require.Len(t, res.Tags, 1)
		assert.Equal(t, "Cozy", res.Tags[0].Name)
		assert.Equal(t, "cozy", res.Tags[0].Slug)
		assert.Equal(t, "", res.Tags[0].Description)

		// rating categories grouped by type, order preserved (vibe then noise)
		require.Len(t, res.RatingCategories, 2)
		assert.Equal(t, "vibe", res.RatingCategories[0].Type)
		assert.Equal(t, "Vibe", res.RatingCategories[0].DisplayName)
		require.Len(t, res.RatingCategories[0].Options, 2)
		assert.Equal(t, "calm-vibe", res.RatingCategories[0].Options[0].Slug)
		assert.Equal(t, "", res.RatingCategories[0].Options[0].LongDescription)
		assert.Equal(t, "noise", res.RatingCategories[1].Type)
		require.Len(t, res.RatingCategories[1].Options, 1)

		// price tiers: last tier's Max becomes nil (open-ended top tier)
		require.Len(t, res.PriceTiers, 3)
		assert.Equal(t, "cheap-price-rank", res.PriceTiers[0].Slug)
		assert.Equal(t, 0, res.PriceTiers[0].Min)
		require.NotNil(t, res.PriceTiers[0].Max)
		assert.Equal(t, 1, *res.PriceTiers[0].Max)
		require.NotNil(t, res.PriceTiers[1].Max)
		assert.Equal(t, 2, *res.PriceTiers[1].Max)
		assert.Nil(t, res.PriceTiers[2].Max)
		assert.Equal(t, "", res.PriceTiers[0].LongDescription)
	})

	t.Run("enrich true includes descriptions", func(t *testing.T) {
		res, err := NewFilterService(filterRepoWith(tags, ratings)).Get(context.Background(), "en", true)
		require.NoError(t, err)
		assert.Equal(t, "cozy desc", res.Tags[0].Description)
		assert.Equal(t, "ld1", res.RatingCategories[0].Options[0].LongDescription)
		assert.Equal(t, "lpd1", res.PriceTiers[0].LongDescription)
	})
}

func TestFilterGet_SinglePriceTierMaxNil(t *testing.T) {
	ratings := []repository.FilterRatingRow{
		{ID: 1, Type: constants.RatingCategoryPriceRank, Slug: "only", Name: "Only", Lower: 0, Upper: 5},
	}
	res, err := NewFilterService(filterRepoWith(nil, ratings)).Get(context.Background(), "en", false)
	require.NoError(t, err)
	require.Len(t, res.PriceTiers, 1)
	assert.Nil(t, res.PriceTiers[0].Max) // the only (last) tier is open-ended
}

func TestFilterGet_BoundTruncation(t *testing.T) {
	ratings := []repository.FilterRatingRow{
		{ID: 1, Type: constants.RatingCategoryPriceRank, Slug: "a", Name: "A", Lower: 1.9, Upper: 2.9},
		{ID: 2, Type: constants.RatingCategoryPriceRank, Slug: "b", Name: "B", Lower: 3.9, Upper: 4.9},
	}
	res, err := NewFilterService(filterRepoWith(nil, ratings)).Get(context.Background(), "en", false)
	require.NoError(t, err)
	require.Len(t, res.PriceTiers, 2)
	assert.Equal(t, 1, res.PriceTiers[0].Min) // int(1.9) truncates
	require.NotNil(t, res.PriceTiers[0].Max)
	assert.Equal(t, 2, *res.PriceTiers[0].Max) // int(2.9) truncates
	assert.Equal(t, 3, res.PriceTiers[1].Min)
}

func TestFilterGet_EmptyResult(t *testing.T) {
	res, err := NewFilterService(filterRepoWith(nil, nil)).Get(context.Background(), "en", false)
	require.NoError(t, err)
	assert.NotNil(t, res.Tags)
	assert.NotNil(t, res.RatingCategories)
	assert.NotNil(t, res.PriceTiers)
	assert.Empty(t, res.PriceTiers)
}
