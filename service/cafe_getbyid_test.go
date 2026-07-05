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

func TestGetByID_RepoErrorsPropagate(t *testing.T) {
	boom := errors.New("db error")

	t.Run("cafe lookup error", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("CafeByLocationID", mock.Anything, "c1", "en").Return(nil, boom)
		svc := NewCafeService(repo)

		_, err := svc.GetByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("images lookup error", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("CafeByLocationID", mock.Anything, "c1", "en").Return(&repository.CafeDetailRow{ID: "c1"}, nil)
		repo.On("CafeImagesByLocationID", mock.Anything, "c1", "en").Return(nil, boom)
		svc := NewCafeService(repo)

		_, err := svc.GetByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, boom)
	})

	t.Run("price rank lookup error", func(t *testing.T) {
		repo := &mockCafeRepo{}
		repo.On("CafeByLocationID", mock.Anything, "c1", "en").Return(&repository.CafeDetailRow{ID: "c1"}, nil)
		repo.On("CafeImagesByLocationID", mock.Anything, "c1", "en").Return([]repository.CafeImageRow{}, nil)
		repo.On("CafePriceRankByLocationID", mock.Anything, "c1").Return(nil, boom)
		svc := NewCafeService(repo)

		_, err := svc.GetByID(context.Background(), "c1", "en")
		assert.ErrorIs(t, err, boom)
	})
}

// getByIDRepo wires the three reads with the given row, images and rank.
func getByIDRepo(row *repository.CafeDetailRow, images []repository.CafeImageRow, rank *int) *mockCafeRepo {
	repo := &mockCafeRepo{}
	repo.On("CafeByLocationID", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)
	repo.On("CafeImagesByLocationID", mock.Anything, mock.Anything, mock.Anything).Return(images, nil)
	repo.On("CafePriceRankByLocationID", mock.Anything, mock.Anything).Return(rank, nil)
	return repo
}

func TestGetByID_PriceRank(t *testing.T) {
	t.Run("nil rank yields no rank", func(t *testing.T) {
		repo := getByIDRepo(&repository.CafeDetailRow{ID: "c1"}, nil, nil)
		res, err := NewCafeService(repo).GetByID(context.Background(), "c1", "en")
		require.NoError(t, err)
		assert.Nil(t, res.Price.Rank)
	})

	t.Run("known rank english label", func(t *testing.T) {
		repo := getByIDRepo(&repository.CafeDetailRow{ID: "c1"}, nil, intPtr(0))
		res, err := NewCafeService(repo).GetByID(context.Background(), "c1", constants.LangEnglish)
		require.NoError(t, err)
		require.NotNil(t, res.Price.Rank)
		assert.Equal(t, 0, res.Price.Rank.Type)
		// Pin the literal so a regression to the label text is caught.
		assert.Equal(t, "Bandung pricing - affordable for most", res.Price.Rank.Label)
	})

	t.Run("known rank indonesian label", func(t *testing.T) {
		repo := getByIDRepo(&repository.CafeDetailRow{ID: "c1"}, nil, intPtr(2))
		res, err := NewCafeService(repo).GetByID(context.Background(), "c1", constants.LangIndonesian)
		require.NoError(t, err)
		require.NotNil(t, res.Price.Rank)
		assert.Equal(t, "Harga Jakarta", res.Price.Rank.Label)
	})

	t.Run("out of range rank yields empty label", func(t *testing.T) {
		repo := getByIDRepo(&repository.CafeDetailRow{ID: "c1"}, nil, intPtr(5))
		res, err := NewCafeService(repo).GetByID(context.Background(), "c1", "en")
		require.NoError(t, err)
		require.NotNil(t, res.Price.Rank)
		assert.Equal(t, 5, res.Price.Rank.Type)
		assert.Equal(t, "", res.Price.Rank.Label)
	})
}

func TestGetByID_DescriptionAndAncestors(t *testing.T) {
	t.Run("empty description maps to nil", func(t *testing.T) {
		repo := getByIDRepo(&repository.CafeDetailRow{ID: "c1", Description: ""}, nil, nil)
		res, err := NewCafeService(repo).GetByID(context.Background(), "c1", "en")
		require.NoError(t, err)
		assert.Nil(t, res.Description)
	})

	t.Run("non-empty description set", func(t *testing.T) {
		repo := getByIDRepo(&repository.CafeDetailRow{ID: "c1", Description: "great coffee"}, nil, nil)
		res, err := NewCafeService(repo).GetByID(context.Background(), "c1", "en")
		require.NoError(t, err)
		require.NotNil(t, res.Description)
		assert.Equal(t, "great coffee", *res.Description)
	})

	t.Run("no ancestors", func(t *testing.T) {
		repo := getByIDRepo(&repository.CafeDetailRow{ID: "c1"}, nil, nil)
		res, err := NewCafeService(repo).GetByID(context.Background(), "c1", "en")
		require.NoError(t, err)
		assert.Empty(t, res.Locations)
	})

	t.Run("district and area ancestors ordered", func(t *testing.T) {
		repo := getByIDRepo(&repository.CafeDetailRow{
			ID:         "c1",
			DistrictID: strPtr("d1"), DistrictName: strPtr("Coblong"),
			AreaID: strPtr("a1"), AreaName: strPtr("Dago"),
		}, nil, nil)
		res, err := NewCafeService(repo).GetByID(context.Background(), "c1", "en")
		require.NoError(t, err)
		require.Len(t, res.Locations, 2)
		assert.Equal(t, "d1", res.Locations[0].ID)
		assert.Equal(t, constants.LocationTypeDistrict, res.Locations[0].Type)
		assert.Equal(t, "a1", res.Locations[1].ID)
		assert.Equal(t, constants.LocationTypeArea, res.Locations[1].Type)
	})
}

func TestGetByID_FullMapping(t *testing.T) {
	row := &repository.CafeDetailRow{
		ID: "c1", Name: "Kopi A", Status: "active", Description: "best",
		GmapsID: strPtr("g1"), Instagram: strPtr("@kopia"),
		OpenHour: strPtr("08:00"), CloseHour: strPtr("22:00"),
		PriceRangeMin: intPtr(20000), PriceRangeMax: intPtr(60000),
		CoffeePriceMin: intPtr(18000), CoffeePriceMax: intPtr(35000),
		SnackPriceMin: intPtr(10000), SnackPriceMax: intPtr(25000),
		FoodPriceMin: intPtr(30000), FoodPriceMax: intPtr(70000),
	}
	images := []repository.CafeImageRow{{URL: "u1", Alt: "front"}, {URL: "u2", Alt: "interior"}}
	repo := getByIDRepo(row, images, intPtr(1))

	res, err := NewCafeService(repo).GetByID(context.Background(), "c1", "en")
	require.NoError(t, err)

	assert.Equal(t, "c1", res.ID)
	assert.Equal(t, "Kopi A", res.Name)
	assert.Equal(t, "active", res.Status)
	assert.Equal(t, "g1", *res.GmapsID)
	assert.Equal(t, "@kopia", *res.Instagram)
	require.Len(t, res.Images, 2)
	assert.Equal(t, "u1", res.Images[0].URL)
	assert.Equal(t, "front", res.Images[0].Description)
	assert.Equal(t, 20000, *res.Price.PriceRangeMin)
	assert.Equal(t, 70000, *res.Price.FoodPriceMax)
	require.NotNil(t, res.Price.Rank)
	assert.Equal(t, 1, res.Price.Rank.Type)
}
