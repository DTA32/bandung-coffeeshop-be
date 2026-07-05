package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetReview_NotFoundAndErrors(t *testing.T) {
	t.Run("exists check error propagates", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockCafeRepo{}
		repo.On("CafeExistsByLocationID", mock.Anything, "c1").Return(false, boom)
		_, err := NewCafeService(repo).GetReviewByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("missing cafe returns not found", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("CafeExistsByLocationID", mock.Anything, "c1").Return(false, nil)
		_, err := NewCafeService(repo).GetReviewByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, repository.ErrCafeNotFound)
	})

	t.Run("review lookup error propagates", func(t *testing.T) {
		boom := errors.New("review db error")
		repo := &mockCafeRepo{}
		repo.On("CafeExistsByLocationID", mock.Anything, "c1").Return(true, nil)
		repo.On("CafeReviewByLocationID", mock.Anything, "c1", "en").Return(nil, boom)
		_, err := NewCafeService(repo).GetReviewByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("tags lookup error propagates", func(t *testing.T) {
		boom := errors.New("tags db error")
		repo := &mockCafeRepo{}
		repo.On("CafeExistsByLocationID", mock.Anything, "c1").Return(true, nil)
		repo.On("CafeReviewByLocationID", mock.Anything, "c1", "en").Return(&repository.ReviewRow{}, nil)
		repo.On("CafeTagsByLocationID", mock.Anything, "c1", "en").Return(nil, boom)
		_, err := NewCafeService(repo).GetReviewByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("ratings lookup error propagates", func(t *testing.T) {
		boom := errors.New("ratings db error")
		repo := &mockCafeRepo{}
		repo.On("CafeExistsByLocationID", mock.Anything, "c1").Return(true, nil)
		repo.On("CafeReviewByLocationID", mock.Anything, "c1", "en").Return(&repository.ReviewRow{}, nil)
		repo.On("CafeTagsByLocationID", mock.Anything, "c1", "en").Return([]repository.CafeTagRow{}, nil)
		repo.On("CafeRatingsByLocationID", mock.Anything, "c1", "en").Return(nil, boom)
		_, err := NewCafeService(repo).GetReviewByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, boom)
	})
}

func TestGetReview_ExistsButNoReview(t *testing.T) {
	repo := &mockCafeRepo{}
	repo.On("CafeExistsByLocationID", mock.Anything, "c1").Return(true, nil)
	repo.On("CafeReviewByLocationID", mock.Anything, "c1", "en").Return(nil, nil) // nil review row

	res, err := NewCafeService(repo).GetReviewByID(context.Background(), "c1", "en")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.NotNil(t, res.Tags)
	assert.Empty(t, res.Tags)
	assert.NotNil(t, res.Ratings)
	assert.Empty(t, res.Ratings)
}

func TestGetReview_Aggregation(t *testing.T) {
	repo := &mockCafeRepo{}
	repo.On("CafeExistsByLocationID", mock.Anything, "c1").Return(true, nil)
	repo.On("CafeReviewByLocationID", mock.Anything, "c1", "en").Return(&repository.ReviewRow{
		IsSubjective: true,
		OverallScore: floatPtr(4.2),
		WFCScore:     floatPtr(3.8),
		Content:      strPtr("Lovely place"),
		VisitedAt:    strPtr("2026-01-02"),
		UpdatedAt:    "2026-01-03",
	}, nil)
	repo.On("CafeTagsByLocationID", mock.Anything, "c1", "en").Return([]repository.CafeTagRow{
		{Name: "Cozy", Slug: strPtr("cozy")},
		{Name: "WiFi", Slug: nil},
	}, nil)
	// Category "vibe" score=3 with three buckets exercising the slug rule;
	// plus a second category "noise" to verify grouping.
	repo.On("CafeRatingsByLocationID", mock.Anything, "c1", "en").Return([]repository.CafeRatingRow{
		{CategoryType: "vibe", TypeLabel: "Vibe", Score: 3, Description: "vibe desc",
			RangeName: "Calm", RangeDesc: "calm", LowerBound: 1, UpperBound: 2, Slug: strPtr("calm")}, // out of range -> nil slug
		{CategoryType: "vibe", TypeLabel: "Vibe", Score: 3, Description: "vibe desc",
			RangeName: "Lively", RangeDesc: "lively", LowerBound: 3, UpperBound: 5, Slug: strPtr("lively")}, // in range -> slug
		{CategoryType: "vibe", TypeLabel: "Vibe", Score: 3, Description: "vibe desc",
			RangeName: "Mystery", RangeDesc: "mystery", LowerBound: 3, UpperBound: 5, Slug: nil}, // in range but nil slug
		{CategoryType: "noise", TypeLabel: "Noise", Score: 2, Description: "noise desc",
			RangeName: "Quiet", RangeDesc: "quiet", LowerBound: 1, UpperBound: 2, Slug: strPtr("quiet")}, // boundary inclusive (upper)
	}, nil)

	res, err := NewCafeService(repo).GetReviewByID(context.Background(), "c1", "en")
	require.NoError(t, err)

	// review fields
	assert.True(t, res.IsSubjective)
	assert.Equal(t, 4.2, *res.OverallScore)
	assert.Equal(t, "Lovely place", *res.Content)
	assert.Equal(t, "2026-01-03", res.UpdatedAt)

	// tags
	require.Len(t, res.Tags, 2)
	assert.Equal(t, "Cozy", res.Tags[0].Name)
	require.NotNil(t, res.Tags[0].Slug)
	assert.Equal(t, "cozy", *res.Tags[0].Slug)
	assert.Nil(t, res.Tags[1].Slug)

	// ratings grouped by category type
	require.Len(t, res.Ratings, 2)
	vibe, ok := res.Ratings["vibe"]
	require.True(t, ok)
	assert.Equal(t, "Vibe", vibe.DisplayName)
	assert.Equal(t, float64(3), vibe.Score)
	require.Len(t, vibe.Range, 3)
	assert.Nil(t, vibe.Range[0].Slug)     // out of range
	require.NotNil(t, vibe.Range[1].Slug) // in range + slug
	assert.Equal(t, "lively-vibe", *vibe.Range[1].Slug)
	assert.Nil(t, vibe.Range[2].Slug) // in range, nil slug

	noise, ok := res.Ratings["noise"]
	require.True(t, ok)
	require.Len(t, noise.Range, 1)
	require.NotNil(t, noise.Range[0].Slug) // score==upper bound is inclusive
	assert.Equal(t, "quiet-noise", *noise.Range[0].Slug)
}
