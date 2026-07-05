package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveSearchType(t *testing.T) {
	tests := []struct {
		in       string
		wantLoc  bool
		wantFil  bool
		wantType string
		wantErr  error
	}{
		{"", true, true, "", nil},
		{constants.QuicksearchTypeAll, true, true, "", nil},
		{constants.QuicksearchTypeLocation, true, false, "", nil},
		{constants.QuicksearchTypeFilter, false, true, "", nil},
		{constants.LocationTypeCafe, true, false, constants.LocationTypeCafe, nil},
		{constants.LocationTypePOI, true, false, constants.LocationTypePOI, nil},
		{constants.LocationTypeArea, true, false, constants.LocationTypeArea, nil},
		{constants.LocationTypeDistrict, true, false, constants.LocationTypeDistrict, nil},
		{"bogus", false, false, "", ErrInvalidSearchType},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			loc, fil, locType, err := resolveSearchType(tt.in)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantLoc, loc)
			assert.Equal(t, tt.wantFil, fil)
			assert.Equal(t, tt.wantType, locType)
		})
	}
}

func TestQuicksearch_ShortQueryShortCircuits(t *testing.T) {
	for _, q := range []string{"", "a", " a "} {
		t.Run("q="+q, func(t *testing.T) {
			repo := &mockQuicksearchRepo{} // any call would panic
			res, err := NewQuicksearchService(repo).Quicksearch(context.Background(), q, "all", "en")
			require.NoError(t, err)
			assert.NotNil(t, res)
			assert.Empty(t, res)
		})
	}
}

func TestQuicksearch_InvalidType(t *testing.T) {
	repo := &mockQuicksearchRepo{}
	_, err := NewQuicksearchService(repo).Quicksearch(context.Background(), "ab", "bogus", "en")
	assert.ErrorIs(t, err, ErrInvalidSearchType)
}

func TestQuicksearch_LimitSplit(t *testing.T) {
	t.Run("both sources split the budget to 5 each", func(t *testing.T) {
		repo := &mockQuicksearchRepo{}
		repo.On("Locations", mock.Anything, "ab", "", 5).Return([]model.QuicksearchResult{}, nil)
		repo.On("Filters", mock.Anything, "ab", "en", 5).Return([]model.QuicksearchResult{}, nil)
		_, err := NewQuicksearchService(repo).Quicksearch(context.Background(), "ab", "all", "en")
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("single source gets full limit of 10", func(t *testing.T) {
		repo := &mockQuicksearchRepo{}
		repo.On("Locations", mock.Anything, "ab", "", 10).Return([]model.QuicksearchResult{}, nil)
		_, err := NewQuicksearchService(repo).Quicksearch(context.Background(), "ab", "location", "en")
		require.NoError(t, err)
		repo.AssertExpectations(t)
		repo.AssertNotCalled(t, "Filters", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("filter-only queries filters with full limit", func(t *testing.T) {
		repo := &mockQuicksearchRepo{}
		repo.On("Filters", mock.Anything, "ab", "en", 10).Return([]model.QuicksearchResult{}, nil)
		_, err := NewQuicksearchService(repo).Quicksearch(context.Background(), "ab", "filter", "en")
		require.NoError(t, err)
		repo.AssertExpectations(t)
		repo.AssertNotCalled(t, "Locations", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("specific location type constrains locType", func(t *testing.T) {
		repo := &mockQuicksearchRepo{}
		repo.On("Locations", mock.Anything, "ab", constants.LocationTypeCafe, 10).Return([]model.QuicksearchResult{}, nil)
		_, err := NewQuicksearchService(repo).Quicksearch(context.Background(), "ab", constants.LocationTypeCafe, "en")
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestQuicksearch_ErrorsPropagate(t *testing.T) {
	boom := errors.New("db error")

	t.Run("locations error", func(t *testing.T) {
		repo := &mockQuicksearchRepo{}
		repo.On("Locations", mock.Anything, "ab", "", 10).Return(nil, boom)
		_, err := NewQuicksearchService(repo).Quicksearch(context.Background(), "ab", "location", "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("filters error", func(t *testing.T) {
		repo := &mockQuicksearchRepo{}
		repo.On("Filters", mock.Anything, "ab", "en", 10).Return(nil, boom)
		_, err := NewQuicksearchService(repo).Quicksearch(context.Background(), "ab", "filter", "en")
		assert.ErrorIs(t, err, boom)
	})
}

func TestQuicksearch_OrderingAndNormalization(t *testing.T) {
	repo := &mockQuicksearchRepo{}
	// "AB " normalizes to "ab"; locations come before filters in the result.
	repo.On("Locations", mock.Anything, "ab", "", 5).
		Return([]model.QuicksearchResult{{ID: "l1", Type: constants.LocationTypeArea}}, nil)
	repo.On("Filters", mock.Anything, "ab", "en", 5).
		Return([]model.QuicksearchResult{{ID: "f1", Type: constants.QuicksearchTypeFilter}}, nil)

	res, err := NewQuicksearchService(repo).Quicksearch(context.Background(), "AB ", "all", "en")
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, "l1", res[0].ID)
	assert.Equal(t, "f1", res[1].ID)
}
