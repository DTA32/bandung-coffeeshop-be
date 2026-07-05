package service

import (
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validate never touches s.repo, so a zero-value service is enough.
func newValidateService() *CafeService { return &CafeService{} }

func TestValidate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		req     model.CafeSearchRequest
		wantErr error
	}{
		{"invalid query_type", model.CafeSearchRequest{QueryType: "village", QueryID: "x"}, ErrInvalidLocationType},
		{"invalid type beats missing id", model.CafeSearchRequest{QueryType: "village"}, ErrInvalidLocationType},
		{"query_type without id", model.CafeSearchRequest{QueryType: constants.LocationTypeCafe}, ErrQueryTypeWithoutID},
		{"id without type", model.CafeSearchRequest{QueryID: "abc"}, ErrIDWithoutType},
		{
			"coords conflicts with id",
			model.CafeSearchRequest{QueryID: "abc", QueryType: constants.LocationTypeCafe, QueryCoords: &model.Coordinates{Lat: 1, Lng: 2}},
			ErrCoordsConflictsWithID,
		},
		{"invalid open_hour", model.CafeSearchRequest{OpenHour: "25:00"}, ErrInvalidOpenHour},
		{"malformed open_hour single digit", model.CafeSearchRequest{OpenHour: "9:5"}, ErrInvalidOpenHour},
		{"non-time open_hour", model.CafeSearchRequest{OpenHour: "morning"}, ErrInvalidOpenHour},
		{"price min exceeds max", model.CafeSearchRequest{PriceMin: intPtr(50000), PriceMax: intPtr(25000)}, ErrInvalidPriceRange},
		{"invalid sort", model.CafeSearchRequest{Sort: "popularity"}, ErrInvalidSort},
		{"invalid order", model.CafeSearchRequest{Order: "ascending"}, ErrInvalidOrder},
		{"distance sort without coords or focus", model.CafeSearchRequest{Sort: constants.SortDistance}, ErrDistanceSortNeedsCoords},
		{
			"distance sort with area focus still needs coords",
			model.CafeSearchRequest{Sort: constants.SortDistance, QueryType: constants.LocationTypeArea, QueryID: "a1"},
			ErrDistanceSortNeedsCoords,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			err := newValidateService().validate(&req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidate_Valid(t *testing.T) {
	tests := []struct {
		name string
		req  model.CafeSearchRequest
	}{
		{"empty request is valid", model.CafeSearchRequest{}},
		{"now open_hour allowed", model.CafeSearchRequest{OpenHour: "now"}},
		{"valid open_hour", model.CafeSearchRequest{OpenHour: "09:30"}},
		{"equal price bounds allowed", model.CafeSearchRequest{PriceMin: intPtr(25000), PriceMax: intPtr(25000)}},
		{"only price min allowed", model.CafeSearchRequest{PriceMin: intPtr(25000)}},
		{"only price max allowed", model.CafeSearchRequest{PriceMax: intPtr(25000)}},
		{"empty order allowed", model.CafeSearchRequest{Order: ""}},
		{"asc order allowed", model.CafeSearchRequest{Order: constants.OrderAsc}},
		{"distance sort with coords", model.CafeSearchRequest{Sort: constants.SortDistance, QueryCoords: &model.Coordinates{Lat: 1, Lng: 2}}},
		{"distance sort with cafe focus", model.CafeSearchRequest{Sort: constants.SortDistance, QueryType: constants.LocationTypeCafe, QueryID: "c1"}},
		{"distance sort with poi focus", model.CafeSearchRequest{Sort: constants.SortDistance, QueryType: constants.LocationTypePOI, QueryID: "p1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			assert.NoError(t, newValidateService().validate(&req))
		})
	}
}

func TestValidate_RadiusMaxDefaulting(t *testing.T) {
	tests := []struct {
		name string
		req  model.CafeSearchRequest
		want *int
	}{
		{"coords defaults to 3000", model.CafeSearchRequest{QueryCoords: &model.Coordinates{Lat: 1, Lng: 2}}, intPtr(3000)},
		{"cafe focus defaults to 3000", model.CafeSearchRequest{QueryType: constants.LocationTypeCafe, QueryID: "c1"}, intPtr(3000)},
		{"poi focus defaults to 2000", model.CafeSearchRequest{QueryType: constants.LocationTypePOI, QueryID: "p1"}, intPtr(2000)},
		{"area focus leaves nil", model.CafeSearchRequest{QueryType: constants.LocationTypeArea, QueryID: "a1"}, nil},
		{"district focus leaves nil", model.CafeSearchRequest{QueryType: constants.LocationTypeDistrict, QueryID: "d1"}, nil},
		{"global leaves nil", model.CafeSearchRequest{}, nil},
		{"explicit radius preserved", model.CafeSearchRequest{QueryType: constants.LocationTypeCafe, QueryID: "c1", RadiusMax: intPtr(500)}, intPtr(500)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			require.NoError(t, newValidateService().validate(&req))
			if tt.want == nil {
				assert.Nil(t, req.RadiusMax)
				return
			}
			require.NotNil(t, req.RadiusMax)
			assert.Equal(t, *tt.want, *req.RadiusMax)
		})
	}
}

func TestValidate_SortAndPaginationDefaults(t *testing.T) {
	t.Run("empty sort defaults to default", func(t *testing.T) {
		req := model.CafeSearchRequest{}
		require.NoError(t, newValidateService().validate(&req))
		assert.Equal(t, constants.SortDefault, req.Sort)
	})

	tests := []struct {
		name               string
		page, size         int
		wantPage, wantSize int
	}{
		{"zero page and size get defaults", 0, 0, defaultPageNum, defaultPageSize},
		{"negative page and size get defaults", -3, -1, defaultPageNum, defaultPageSize},
		{"size over max is clamped", 1, 100, 1, maxPageSize},
		{"size at max preserved", 2, maxPageSize, 2, maxPageSize},
		{"valid values preserved", 3, 20, 3, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := model.CafeSearchRequest{Page: tt.page, Size: tt.size}
			require.NoError(t, newValidateService().validate(&req))
			assert.Equal(t, tt.wantPage, req.Page)
			assert.Equal(t, tt.wantSize, req.Size)
		})
	}
}
