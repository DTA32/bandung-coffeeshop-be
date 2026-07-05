package service

import (
	"context"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/stretchr/testify/mock"
)

// Compile-time checks that each mock satisfies the consumer interface it stands
// in for. If a service interface changes, these break first.
var (
	_ cafeRepository        = (*mockCafeRepo)(nil)
	_ filterRepository      = (*mockFilterRepo)(nil)
	_ locationRepository    = (*mockLocationRepo)(nil)
	_ quicksearchRepository = (*mockQuicksearchRepo)(nil)
)

// ---- cafeRepository ----

type mockCafeRepo struct{ mock.Mock }

func (m *mockCafeRepo) ResolveFocus(ctx context.Context, id, queryType, lang string) (*repository.FocusLocation, error) {
	args := m.Called(ctx, id, queryType, lang)
	f, _ := args.Get(0).(*repository.FocusLocation)
	return f, args.Error(1)
}

func (m *mockCafeRepo) RatingCategoriesByIDs(ctx context.Context, ids []int) ([]repository.RatingCategory, error) {
	args := m.Called(ctx, ids)
	cats, _ := args.Get(0).([]repository.RatingCategory)
	return cats, args.Error(1)
}

func (m *mockCafeRepo) TagBySlug(ctx context.Context, slug, lang string) (*repository.Tag, error) {
	args := m.Called(ctx, slug, lang)
	t, _ := args.Get(0).(*repository.Tag)
	return t, args.Error(1)
}

func (m *mockCafeRepo) Search(ctx context.Context, p repository.CafeSearchParams) ([]repository.CafeSearchRow, int, error) {
	args := m.Called(ctx, p)
	rows, _ := args.Get(0).([]repository.CafeSearchRow)
	return rows, args.Int(1), args.Error(2)
}

func (m *mockCafeRepo) CafeByLocationID(ctx context.Context, locationID, lang string) (*repository.CafeDetailRow, error) {
	args := m.Called(ctx, locationID, lang)
	row, _ := args.Get(0).(*repository.CafeDetailRow)
	return row, args.Error(1)
}

func (m *mockCafeRepo) CafeImagesByLocationID(ctx context.Context, locationID, lang string) ([]repository.CafeImageRow, error) {
	args := m.Called(ctx, locationID, lang)
	imgs, _ := args.Get(0).([]repository.CafeImageRow)
	return imgs, args.Error(1)
}

func (m *mockCafeRepo) CafePriceRankByLocationID(ctx context.Context, locationID string) (*int, error) {
	args := m.Called(ctx, locationID)
	rank, _ := args.Get(0).(*int)
	return rank, args.Error(1)
}

func (m *mockCafeRepo) CafeExistsByLocationID(ctx context.Context, locationID string) (bool, error) {
	args := m.Called(ctx, locationID)
	return args.Bool(0), args.Error(1)
}

func (m *mockCafeRepo) CafeReviewByLocationID(ctx context.Context, locationID, lang string) (*repository.ReviewRow, error) {
	args := m.Called(ctx, locationID, lang)
	row, _ := args.Get(0).(*repository.ReviewRow)
	return row, args.Error(1)
}

func (m *mockCafeRepo) CafeTagsByLocationID(ctx context.Context, locationID, lang string) ([]repository.CafeTagRow, error) {
	args := m.Called(ctx, locationID, lang)
	tags, _ := args.Get(0).([]repository.CafeTagRow)
	return tags, args.Error(1)
}

func (m *mockCafeRepo) CafeRatingsByLocationID(ctx context.Context, locationID, lang string) ([]repository.CafeRatingRow, error) {
	args := m.Called(ctx, locationID, lang)
	ratings, _ := args.Get(0).([]repository.CafeRatingRow)
	return ratings, args.Error(1)
}

// ---- filterRepository ----

type mockFilterRepo struct{ mock.Mock }

func (m *mockFilterRepo) Tags(ctx context.Context, lang string) ([]repository.FilterTagRow, error) {
	args := m.Called(ctx, lang)
	rows, _ := args.Get(0).([]repository.FilterTagRow)
	return rows, args.Error(1)
}

func (m *mockFilterRepo) RatingCategories(ctx context.Context, lang string) ([]repository.FilterRatingRow, error) {
	args := m.Called(ctx, lang)
	rows, _ := args.Get(0).([]repository.FilterRatingRow)
	return rows, args.Error(1)
}

// ---- locationRepository ----

type mockLocationRepo struct{ mock.Mock }

func (m *mockLocationRepo) GetByID(ctx context.Context, id, lang string) (*repository.LocationDetailRow, error) {
	args := m.Called(ctx, id, lang)
	row, _ := args.Get(0).(*repository.LocationDetailRow)
	return row, args.Error(1)
}

func (m *mockLocationRepo) Ancestors(ctx context.Context, id string) ([]model.Location, error) {
	args := m.Called(ctx, id)
	locs, _ := args.Get(0).([]model.Location)
	return locs, args.Error(1)
}

func (m *mockLocationRepo) Descendants(ctx context.Context, row *repository.LocationDetailRow) ([]model.Location, error) {
	args := m.Called(ctx, row)
	locs, _ := args.Get(0).([]model.Location)
	return locs, args.Error(1)
}

func (m *mockLocationRepo) Images(ctx context.Context, id, lang string) ([]model.LocationImage, error) {
	args := m.Called(ctx, id, lang)
	imgs, _ := args.Get(0).([]model.LocationImage)
	return imgs, args.Error(1)
}

func (m *mockLocationRepo) Districts(ctx context.Context) ([]repository.LocationDetailRow, error) {
	args := m.Called(ctx)
	rows, _ := args.Get(0).([]repository.LocationDetailRow)
	return rows, args.Error(1)
}

// ---- quicksearchRepository ----

type mockQuicksearchRepo struct{ mock.Mock }

func (m *mockQuicksearchRepo) Locations(ctx context.Context, q, locType string, limit int) ([]model.QuicksearchResult, error) {
	args := m.Called(ctx, q, locType, limit)
	res, _ := args.Get(0).([]model.QuicksearchResult)
	return res, args.Error(1)
}

func (m *mockQuicksearchRepo) Filters(ctx context.Context, q, lang string, limit int) ([]model.QuicksearchResult, error) {
	args := m.Called(ctx, q, lang, limit)
	res, _ := args.Get(0).([]model.QuicksearchResult)
	return res, args.Error(1)
}

// intPtr / strPtr / boolPtr / floatPtr are pointer helpers shared by service tests.
func intPtr(v int) *int           { return &v }
func strPtr(v string) *string     { return &v }
func boolPtr(v bool) *bool        { return &v }
func floatPtr(v float64) *float64 { return &v }
