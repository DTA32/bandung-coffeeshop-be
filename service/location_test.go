package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPolygonJSON(t *testing.T) {
	assert.Nil(t, polygonJSON(nil))

	geo := `{"type":"Polygon","coordinates":[]}`
	got := polygonJSON(&geo)
	assert.Equal(t, json.RawMessage(geo), got)
}

func TestLocationGetByID_Errors(t *testing.T) {
	t.Run("getbyid error propagates", func(t *testing.T) {
		repo := &mockLocationRepo{}
		repo.On("GetByID", mock.Anything, "x", "en").Return(nil, repository.ErrLocationNotFound)
		_, err := NewLocationService(repo).GetByID(context.Background(), "x", "en")
		assert.ErrorIs(t, err, repository.ErrLocationNotFound)
	})

	t.Run("cafe type rejected", func(t *testing.T) {
		repo := &mockLocationRepo{}
		repo.On("GetByID", mock.Anything, "c1", "en").
			Return(&repository.LocationDetailRow{ID: "c1", Type: constants.LocationTypeCafe}, nil)
		_, err := NewLocationService(repo).GetByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, ErrLocationIsCafe)
	})

	t.Run("ancestors error propagates", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockLocationRepo{}
		repo.On("GetByID", mock.Anything, "a1", "en").
			Return(&repository.LocationDetailRow{ID: "a1", Type: constants.LocationTypeArea}, nil)
		repo.On("Ancestors", mock.Anything, "a1").Return(nil, boom)
		_, err := NewLocationService(repo).GetByID(context.Background(), "a1", "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("descendants error propagates", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockLocationRepo{}
		repo.On("GetByID", mock.Anything, "a1", "en").
			Return(&repository.LocationDetailRow{ID: "a1", Type: constants.LocationTypeArea}, nil)
		repo.On("Ancestors", mock.Anything, "a1").Return([]model.Location{}, nil)
		repo.On("Descendants", mock.Anything, mock.Anything).Return(nil, boom)
		_, err := NewLocationService(repo).GetByID(context.Background(), "a1", "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("images error propagates", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockLocationRepo{}
		repo.On("GetByID", mock.Anything, "a1", "en").
			Return(&repository.LocationDetailRow{ID: "a1", Type: constants.LocationTypeArea}, nil)
		repo.On("Ancestors", mock.Anything, "a1").Return([]model.Location{}, nil)
		repo.On("Descendants", mock.Anything, mock.Anything).Return([]model.Location{}, nil)
		repo.On("Images", mock.Anything, "a1", "en").Return(nil, boom)
		_, err := NewLocationService(repo).GetByID(context.Background(), "a1", "en")
		assert.ErrorIs(t, err, boom)
	})
}

func locationRepoForGet(row *repository.LocationDetailRow, images []model.LocationImage) *mockLocationRepo {
	repo := &mockLocationRepo{}
	repo.On("GetByID", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)
	repo.On("Ancestors", mock.Anything, mock.Anything).Return([]model.Location{}, nil)
	repo.On("Descendants", mock.Anything, mock.Anything).Return([]model.Location{}, nil)
	repo.On("Images", mock.Anything, mock.Anything, mock.Anything).Return(images, nil)
	return repo
}

func TestLocationGetByID_ShowMapMatrix(t *testing.T) {
	withImg := []model.LocationImage{{URL: "u1"}}
	tests := []struct {
		name     string
		locType  string
		images   []model.LocationImage
		wantShow bool
	}{
		{"district always shows map", constants.LocationTypeDistrict, nil, true},
		{"district shows map even with images", constants.LocationTypeDistrict, withImg, true},
		{"area without images shows map", constants.LocationTypeArea, nil, true},
		{"area with images hides map", constants.LocationTypeArea, withImg, false},
		{"poi without images hides map", constants.LocationTypePOI, nil, false},
		{"poi with images hides map", constants.LocationTypePOI, withImg, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := locationRepoForGet(&repository.LocationDetailRow{ID: "l1", Type: tt.locType}, tt.images)
			res, err := NewLocationService(repo).GetByID(context.Background(), "l1", "en")
			require.NoError(t, err)
			assert.Equal(t, tt.wantShow, res.ShowMap)
		})
	}
}

func TestLocationGetByID_WelcomeAndPolygon(t *testing.T) {
	t.Run("welcome text only for area", func(t *testing.T) {
		areaRepo := locationRepoForGet(&repository.LocationDetailRow{ID: "a1", Type: constants.LocationTypeArea}, nil)
		areaRes, err := NewLocationService(areaRepo).GetByID(context.Background(), "a1", "en")
		require.NoError(t, err)
		assert.True(t, areaRes.ShowWelcomeText)

		distRepo := locationRepoForGet(&repository.LocationDetailRow{ID: "d1", Type: constants.LocationTypeDistrict}, nil)
		distRes, err := NewLocationService(distRepo).GetByID(context.Background(), "d1", "en")
		require.NoError(t, err)
		assert.False(t, distRes.ShowWelcomeText)
	})

	t.Run("polygon passed through", func(t *testing.T) {
		geo := `{"type":"Polygon"}`
		repo := locationRepoForGet(&repository.LocationDetailRow{ID: "d1", Type: constants.LocationTypeDistrict, PolygonGeoJSON: &geo}, nil)
		res, err := NewLocationService(repo).GetByID(context.Background(), "d1", "en")
		require.NoError(t, err)
		assert.Equal(t, json.RawMessage(geo), res.Polygon)
	})
}

func TestListDistricts(t *testing.T) {
	t.Run("districts error propagates", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockLocationRepo{}
		repo.On("Districts", mock.Anything).Return(nil, boom)
		_, err := NewLocationService(repo).ListDistricts(context.Background(), "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("descendants error mid-loop propagates", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockLocationRepo{}
		repo.On("Districts", mock.Anything).
			Return([]repository.LocationDetailRow{{ID: "d1", Type: constants.LocationTypeDistrict}}, nil)
		repo.On("Descendants", mock.Anything, mock.Anything).Return(nil, boom)
		_, err := NewLocationService(repo).ListDistricts(context.Background(), "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("images error mid-loop propagates", func(t *testing.T) {
		boom := errors.New("db error")
		repo := &mockLocationRepo{}
		repo.On("Districts", mock.Anything).
			Return([]repository.LocationDetailRow{{ID: "d1", Type: constants.LocationTypeDistrict}}, nil)
		repo.On("Descendants", mock.Anything, mock.Anything).Return([]model.Location{}, nil)
		repo.On("Images", mock.Anything, "d1", mock.Anything).Return(nil, boom)
		_, err := NewLocationService(repo).ListDistricts(context.Background(), "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("empty list is non-nil", func(t *testing.T) {
		repo := &mockLocationRepo{}
		repo.On("Districts", mock.Anything).Return([]repository.LocationDetailRow{}, nil)
		res, err := NewLocationService(repo).ListDistricts(context.Background(), "en")
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Empty(t, res)
	})

	t.Run("builds summaries", func(t *testing.T) {
		repo := &mockLocationRepo{}
		repo.On("Districts", mock.Anything).Return([]repository.LocationDetailRow{
			{ID: "d1", Name: "Coblong", Type: constants.LocationTypeDistrict},
			{ID: "d2", Name: "Lengkong", Type: constants.LocationTypeDistrict},
		}, nil)
		repo.On("Descendants", mock.Anything, mock.Anything).Return([]model.Location{{ID: "a1", Type: constants.LocationTypeArea}}, nil)
		repo.On("Images", mock.Anything, mock.Anything, mock.Anything).Return([]model.LocationImage{{URL: "u1"}}, nil)

		res, err := NewLocationService(repo).ListDistricts(context.Background(), "en")
		require.NoError(t, err)
		require.Len(t, res, 2)
		assert.Equal(t, "d1", res[0].ID)
		assert.NotNil(t, res[0].Ancestors)
		assert.Empty(t, res[0].Ancestors)
		assert.False(t, res[0].ShowMap)
		assert.False(t, res[0].ShowWelcomeText)
		assert.Len(t, res[0].Descendants, 1)
		assert.Len(t, res[0].Images, 1)
	})
}
