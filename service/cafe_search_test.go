package service

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// captureSearch stubs repo.Search, recording the params it receives so tests can
// assert what the service derived. The returned pointer is filled on the call.
func captureSearch(repo *mockCafeRepo, rows []repository.CafeSearchRow, total int) *repository.CafeSearchParams {
	captured := &repository.CafeSearchParams{}
	repo.On("Search", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		*captured = args.Get(1).(repository.CafeSearchParams)
	}).Return(rows, total, nil)
	return captured
}

func TestSearch_ValidationShortCircuits(t *testing.T) {
	repo := &mockCafeRepo{} // no expectations: any repo call would panic
	svc := NewCafeService(repo)

	_, err := svc.Search(context.Background(), model.CafeSearchRequest{Sort: "bogus"})
	assert.ErrorIs(t, err, ErrInvalidSort)
}

func TestSearch_ResolveFocusErrorPropagates(t *testing.T) {
	repo := &mockCafeRepo{}
	repo.On("ResolveFocus", mock.Anything, "x", constants.LocationTypeCafe, mock.Anything).
		Return(nil, repository.ErrFocusNotFound)
	svc := NewCafeService(repo)

	_, err := svc.Search(context.Background(), model.CafeSearchRequest{QueryID: "x", QueryType: constants.LocationTypeCafe})
	assert.ErrorIs(t, err, repository.ErrFocusNotFound)
}

func TestSearch_RatingBuckets(t *testing.T) {
	t.Run("missing category id returns not found", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("RatingCategoriesByIDs", mock.Anything, []int{1, 2}).
			Return([]repository.RatingCategory{{ID: 1, Type: "vibe"}}, nil) // only 1 of 2
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{RatingIDs: []int{1, 2}})
		assert.ErrorIs(t, err, repository.ErrRatingCategoryNotFound)
	})

	t.Run("duplicate category type rejected", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("RatingCategoriesByIDs", mock.Anything, []int{1, 2}).
			Return([]repository.RatingCategory{
				{ID: 1, Type: "vibe", LowerBound: 1, UpperBound: 2},
				{ID: 2, Type: "vibe", LowerBound: 3, UpperBound: 4},
			}, nil)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{RatingIDs: []int{1, 2}})
		assert.ErrorIs(t, err, ErrDuplicateRatingType)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		boom := errors.New("db down")
		repo := &mockCafeRepo{}
		repo.On("RatingCategoriesByIDs", mock.Anything, []int{1}).Return(nil, boom)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{RatingIDs: []int{1}})
		assert.ErrorIs(t, err, boom)
	})

	t.Run("happy path builds rating filters", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("RatingCategoriesByIDs", mock.Anything, []int{1, 2}).
			Return([]repository.RatingCategory{
				{ID: 1, Type: "vibe", LowerBound: 1, UpperBound: 2},
				{ID: 2, Type: "noise", LowerBound: 3, UpperBound: 4},
			}, nil)
		captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{RatingIDs: []int{1, 2}})
		require.NoError(t, err)
		assert.ElementsMatch(t, []repository.RatingFilterParam{
			{Type: "vibe", Lower: 1, Upper: 2},
			{Type: "noise", Lower: 3, Upper: 4},
		}, captured.RatingFilters)
	})
}

func TestSearch_Tags(t *testing.T) {
	t.Run("unknown slug skipped, single known kept", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("TagBySlug", mock.Anything, "cozy", mock.Anything).
			Return(&repository.Tag{Slug: "cozy", Description: "Cozy spots"}, nil)
		repo.On("TagBySlug", mock.Anything, "ghost", mock.Anything).
			Return(nil, repository.ErrTagNotFound)
		captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
		svc := NewCafeService(repo)

		res, err := svc.Search(context.Background(), model.CafeSearchRequest{Tags: []string{"cozy", "ghost"}})
		require.NoError(t, err)
		assert.Equal(t, []string{"cozy"}, captured.TagSlugs)
		// exactly one resolved tag and no other filters => tag-only description.
		assert.Equal(t, "Cozy spots", res.SearchDescription)
	})

	t.Run("other tag error propagates", func(t *testing.T) {
		boom := errors.New("tag db error")
		repo := &mockCafeRepo{}
		repo.On("TagBySlug", mock.Anything, "cozy", mock.Anything).Return(nil, boom)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{Tags: []string{"cozy"}})
		assert.ErrorIs(t, err, boom)
	})

	t.Run("two resolved tags drop tag-only description", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("TagBySlug", mock.Anything, "cozy", mock.Anything).
			Return(&repository.Tag{Slug: "cozy", Description: "Cozy spots"}, nil)
		repo.On("TagBySlug", mock.Anything, "quiet", mock.Anything).
			Return(&repository.Tag{Slug: "quiet", Description: "Quiet spots"}, nil)
		captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
		svc := NewCafeService(repo)

		res, err := svc.Search(context.Background(), model.CafeSearchRequest{Tags: []string{"cozy", "quiet"}})
		require.NoError(t, err)
		assert.Equal(t, []string{"cozy", "quiet"}, captured.TagSlugs)
		assert.Equal(t, "", res.SearchDescription)
	})
}

func TestSearch_RepoSearchErrorPropagates(t *testing.T) {
	boom := errors.New("search db error")
	repo := &mockCafeRepo{}
	repo.On("Search", mock.Anything, mock.Anything).Return(nil, 0, boom)
	svc := NewCafeService(repo)

	_, err := svc.Search(context.Background(), model.CafeSearchRequest{})
	assert.ErrorIs(t, err, boom)
}

func TestSearch_OpenHourNow(t *testing.T) {
	repo := &mockCafeRepo{}
	captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
	svc := NewCafeService(repo)

	_, err := svc.Search(context.Background(), model.CafeSearchRequest{OpenHour: "now"})
	require.NoError(t, err)
	require.NotNil(t, captured.OpenHour)
	assert.Regexp(t, regexp.MustCompile(`^\d{2}:\d{2}$`), *captured.OpenHour)
}

func TestSearch_OpenHourLiteral(t *testing.T) {
	repo := &mockCafeRepo{}
	captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
	svc := NewCafeService(repo)

	_, err := svc.Search(context.Background(), model.CafeSearchRequest{OpenHour: "08:30"})
	require.NoError(t, err)
	require.NotNil(t, captured.OpenHour)
	assert.Equal(t, "08:30", *captured.OpenHour)
}

func TestSearch_ModeSelection(t *testing.T) {
	t.Run("area focus uses polygon mode", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("ResolveFocus", mock.Anything, "a1", constants.LocationTypeArea, mock.Anything).
			Return(&repository.FocusLocation{ID: "a1", Type: constants.LocationTypeArea, CenterLat: 1.5, CenterLng: 2.5}, nil)
		captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{QueryID: "a1", QueryType: constants.LocationTypeArea})
		require.NoError(t, err)
		assert.Equal(t, repository.SearchModePolygon, captured.Mode)
		assert.Equal(t, "a1", captured.PolygonLocID)
		require.NotNil(t, captured.FocusLat)
		require.NotNil(t, captured.FocusLng)
		assert.Equal(t, 1.5, *captured.FocusLat)
		assert.Equal(t, 2.5, *captured.FocusLng)
		assert.Nil(t, captured.ExcludeIDs)
	})

	t.Run("cafe focus uses radius mode and excludes itself", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("ResolveFocus", mock.Anything, "c1", constants.LocationTypeCafe, mock.Anything).
			Return(&repository.FocusLocation{ID: "c1", Type: constants.LocationTypeCafe, CenterLat: 1, CenterLng: 2}, nil)
		captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{QueryID: "c1", QueryType: constants.LocationTypeCafe})
		require.NoError(t, err)
		assert.Equal(t, repository.SearchModeRadius, captured.Mode)
		assert.Equal(t, []string{"c1"}, captured.ExcludeIDs)
		require.NotNil(t, captured.RadiusMax)
		assert.Equal(t, 3000, *captured.RadiusMax) // cafe default
	})

	t.Run("poi focus radius uses 2000 default", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("ResolveFocus", mock.Anything, "p1", constants.LocationTypePOI, mock.Anything).
			Return(&repository.FocusLocation{ID: "p1", Type: constants.LocationTypePOI, CenterLat: 1, CenterLng: 2}, nil)
		captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{QueryID: "p1", QueryType: constants.LocationTypePOI})
		require.NoError(t, err)
		assert.Equal(t, repository.SearchModeRadius, captured.Mode)
		assert.Equal(t, []string{"p1"}, captured.ExcludeIDs)
		require.NotNil(t, captured.RadiusMax)
		assert.Equal(t, 2000, *captured.RadiusMax)
	})

	t.Run("coords use radius mode without exclude", func(t *testing.T) {
		repo := &mockCafeRepo{}
		captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{QueryCoords: &model.Coordinates{Lat: 9, Lng: 8}})
		require.NoError(t, err)
		assert.Equal(t, repository.SearchModeRadius, captured.Mode)
		require.NotNil(t, captured.FocusLat)
		require.NotNil(t, captured.FocusLng)
		assert.Equal(t, 9.0, *captured.FocusLat)
		assert.Equal(t, 8.0, *captured.FocusLng)
		assert.Nil(t, captured.ExcludeIDs)
		require.NotNil(t, captured.RadiusMax)
		assert.Equal(t, 3000, *captured.RadiusMax)
	})

	t.Run("no focus or coords uses global mode", func(t *testing.T) {
		repo := &mockCafeRepo{}
		captured := captureSearch(repo, []repository.CafeSearchRow{}, 0)
		svc := NewCafeService(repo)

		_, err := svc.Search(context.Background(), model.CafeSearchRequest{})
		require.NoError(t, err)
		assert.Equal(t, repository.SearchModeGlobal, captured.Mode)
		assert.Nil(t, captured.FocusLat)
		assert.Nil(t, captured.ExcludeIDs)
	})
}

func TestSearch_DistanceEmission(t *testing.T) {
	rows := []repository.CafeSearchRow{
		{ID: "c1", Name: "Kopi A", Lat: floatPtr(1), Lng: floatPtr(2), DistanceM: intPtr(150)},
	}

	t.Run("distance emitted with coords", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("Search", mock.Anything, mock.Anything).Return(rows, 1, nil)
		svc := NewCafeService(repo)

		res, err := svc.Search(context.Background(), model.CafeSearchRequest{QueryCoords: &model.Coordinates{Lat: 1, Lng: 2}})
		require.NoError(t, err)
		require.Len(t, res.Cafes, 1)
		require.NotNil(t, res.Cafes[0].Distance)
		assert.Equal(t, 150, *res.Cafes[0].Distance)
		require.NotNil(t, res.Cafes[0].Coordinates)
		assert.Equal(t, model.Coordinates{Lat: 1, Lng: 2}, *res.Cafes[0].Coordinates)
	})

	t.Run("distance suppressed for focus radius without coords", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("ResolveFocus", mock.Anything, "c9", constants.LocationTypeCafe, mock.Anything).
			Return(&repository.FocusLocation{ID: "c9", Type: constants.LocationTypeCafe, CenterLat: 1, CenterLng: 2}, nil)
		repo.On("Search", mock.Anything, mock.Anything).Return(rows, 1, nil)
		svc := NewCafeService(repo)

		res, err := svc.Search(context.Background(), model.CafeSearchRequest{QueryID: "c9", QueryType: constants.LocationTypeCafe})
		require.NoError(t, err)
		require.Len(t, res.Cafes, 1)
		assert.Nil(t, res.Cafes[0].Distance)
	})
}

func TestSearch_ResponseAssembly(t *testing.T) {
	repo := &mockCafeRepo{}
	repo.On("ResolveFocus", mock.Anything, "a1", constants.LocationTypeArea, mock.Anything).
		Return(&repository.FocusLocation{
			ID: "a1", Name: "Dago", Type: constants.LocationTypeArea, Description: "Dago area",
			CenterLat: 1, CenterLng: 2,
			DistrictID: strPtr("d1"), DistrictName: strPtr("Coblong"),
		}, nil)
	repo.On("Search", mock.Anything, mock.Anything).Return([]repository.CafeSearchRow{
		{ID: "c1", Name: "Kopi A", Description: "desc", PriceRangeMin: intPtr(25000), PriceRangeMax: intPtr(50000), Area: strPtr("Dago")},
	}, 42, nil)
	svc := NewCafeService(repo)

	res, err := svc.Search(context.Background(), model.CafeSearchRequest{QueryID: "a1", QueryType: constants.LocationTypeArea, Lang: "en"})
	require.NoError(t, err)

	assert.Equal(t, 42, res.Total)
	assert.Equal(t, defaultPageNum, res.Page) // defaulted by validate
	assert.Equal(t, defaultPageSize, res.Size)
	assert.Equal(t, "Dago", res.LocationName)
	assert.Equal(t, "in Dago", res.FormattedLocationName)
	assert.Equal(t, "Dago area", res.SearchDescription)
	// breadcrumb: district then the area itself.
	require.Len(t, res.Locations, 2)
	assert.Equal(t, "d1", res.Locations[0].ID)
	assert.Equal(t, "a1", res.Locations[1].ID)
	// cafe price range formatted, no distance (no coords).
	require.Len(t, res.Cafes, 1)
	require.NotNil(t, res.Cafes[0].PriceRange)
	assert.Equal(t, "Rp. 25k - Rp. 50k", *res.Cafes[0].PriceRange)
	assert.Nil(t, res.Cafes[0].Distance)
}
