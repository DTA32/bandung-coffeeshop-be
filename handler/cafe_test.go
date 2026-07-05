package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestParseCSV(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty returns nil", "", nil},
		{"single", "a", []string{"a"}},
		{"trims and drops empties", " a , , b ", []string{"a", "b"}},
		{"dedupes preserving order", "a,b,a,c,b", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseCSV(tt.in))
		})
	}
}

func TestParseIntCSV(t *testing.T) {
	t.Run("valid dedupes preserving order", func(t *testing.T) {
		got, err := parseIntCSV("1, 2 ,1,3")
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, got)
	})

	t.Run("skips empty parts", func(t *testing.T) {
		got, err := parseIntCSV("1,,2")
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, got)
	})

	t.Run("errors on non-integer", func(t *testing.T) {
		_, err := parseIntCSV("1,abc")
		assert.Error(t, err)
	})
}

func TestCafeSearch_BadParams(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantMsg string
	}{
		{"invalid ratings", "ratings=1,abc", "invalid ratings"},
		{"negative price_min", "price_min=-1", "invalid price_min"},
		{"price_min too large", "price_min=1000000", "invalid price_min"},
		{"non-numeric price_min", "price_min=abc", "invalid price_min"},
		{"negative price_max", "price_max=-5", "invalid price_max"},
		{"price_max too large", "price_max=1000000", "invalid price_max"},
		{"non-numeric price_max", "price_max=abc", "invalid price_max"},
		{"invalid query_coords", "query_coords=999,0", "invalid query_coords"},
		{"zero radius_max", "radius_max=0", "invalid radius_max"},
		{"non-numeric radius_max", "radius_max=abc", "invalid radius_max"},
		{"invalid is_featured", "is_featured=maybe", "invalid is_featured"},
		{"zero page", "page=0", "invalid page"},
		{"non-numeric size", "size=abc", "invalid size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockCafeService{} // must not be called
			h := NewCafeHandler(svc)
			c, w := newTestContext(http.MethodGet, tt.query, nil, nil)

			h.Search(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			e := decodeEnvelope(t, w)
			assert.False(t, e.Success)
			assert.Equal(t, tt.wantMsg, e.Error)
			svc.AssertNotCalled(t, "Search", mock.Anything, mock.Anything)
		})
	}
}

func TestCafeSearch_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{"validation sentinel maps to 400", service.ErrInvalidSort, http.StatusBadRequest, "invalid sort"},
		{"duplicate rating maps to 400", service.ErrDuplicateRatingType, http.StatusBadRequest, service.ErrDuplicateRatingType.Error()},
		{"focus not found maps to 404", repository.ErrFocusNotFound, http.StatusNotFound, "focus location not found"},
		{"tag not found maps to 404", repository.ErrTagNotFound, http.StatusNotFound, "tag not found"},
		{"unknown error maps to 500", errors.New("boom"), http.StatusInternalServerError, "search failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockCafeService{}
			svc.On("Search", mock.Anything, mock.Anything).Return(nil, tt.err)
			h := NewCafeHandler(svc)
			c, w := newTestContext(http.MethodGet, "", nil, nil)

			h.Search(c)

			assert.Equal(t, tt.wantCode, w.Code)
			e := decodeEnvelope(t, w)
			assert.False(t, e.Success)
			assert.Equal(t, tt.wantMsg, e.Error)
		})
	}
}

func TestCafeSearch_Success(t *testing.T) {
	svc := &mockCafeService{}
	svc.On("Search", mock.Anything, mock.Anything).Return(&model.CafeSearchResponse{Total: 3}, nil)
	h := NewCafeHandler(svc)
	c, w := newTestContext(http.MethodGet, "query_id=a1&query_type=area&tags=cozy,quiet", nil, nil)

	h.Search(c)

	assert.Equal(t, http.StatusOK, w.Code)
	e := decodeEnvelope(t, w)
	assert.True(t, e.Success)
	svc.AssertExpectations(t)
}

func TestCafeSearch_AllParamsValidReachService(t *testing.T) {
	svc := &mockCafeService{}
	svc.On("Search", mock.Anything, mock.Anything).Return(&model.CafeSearchResponse{}, nil)
	h := NewCafeHandler(svc)
	query := "ratings=1,2&price_min=10000&price_max=50000&query_coords=-6.9,107.6" +
		"&radius_max=2000&is_featured=true&page=2&size=10&open_hour=09:00&sort=rating&order=asc"
	c, w := newTestContext(http.MethodGet, query, nil, nil)

	h.Search(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, decodeEnvelope(t, w).Success)

	// the parsed request reached the service with every optional field populated.
	req := svc.Calls[0].Arguments.Get(1).(model.CafeSearchRequest)
	assert.Equal(t, []int{1, 2}, req.RatingIDs)
	require.NotNil(t, req.PriceMin)
	assert.Equal(t, 10000, *req.PriceMin)
	require.NotNil(t, req.PriceMax)
	assert.Equal(t, 50000, *req.PriceMax)
	require.NotNil(t, req.QueryCoords)
	assert.Equal(t, -6.9, req.QueryCoords.Lat)
	assert.Equal(t, 107.6, req.QueryCoords.Lng)
	require.NotNil(t, req.RadiusMax)
	assert.Equal(t, 2000, *req.RadiusMax)
	require.NotNil(t, req.IsFeatured)
	assert.True(t, *req.IsFeatured)
	assert.Equal(t, 2, req.Page)
	assert.Equal(t, 10, req.Size)
}

func TestCafeGetByID(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string // "" => expect success
	}{
		{"not found maps to 404", repository.ErrCafeNotFound, http.StatusNotFound, "cafe not found"},
		{"other error maps to 500 with message", errors.New("boom"), http.StatusInternalServerError, "boom"},
		{"success", nil, http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockCafeService{}
			if tt.err != nil {
				svc.On("GetByID", mock.Anything, "c1", mock.Anything).Return(nil, tt.err)
			} else {
				svc.On("GetByID", mock.Anything, "c1", mock.Anything).Return(&model.CafeDetailResponse{ID: "c1"}, nil)
			}
			h := NewCafeHandler(svc)
			c, w := newTestContext(http.MethodGet, "", gin.Params{{Key: "id", Value: "c1"}}, nil)

			h.GetByID(c)

			assert.Equal(t, tt.wantCode, w.Code)
			e := decodeEnvelope(t, w)
			if tt.wantMsg == "" {
				assert.True(t, e.Success)
			} else {
				assert.Equal(t, tt.wantMsg, e.Error)
			}
		})
	}
}

func TestCafeGetReview(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{"not found maps to 404", repository.ErrCafeNotFound, http.StatusNotFound, "cafe not found"},
		{"other error maps to generic 500", errors.New("boom"), http.StatusInternalServerError, "failed to fetch review"},
		{"success", nil, http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockCafeService{}
			if tt.err != nil {
				svc.On("GetReviewByID", mock.Anything, "c1", mock.Anything).Return(nil, tt.err)
			} else {
				svc.On("GetReviewByID", mock.Anything, "c1", mock.Anything).Return(&model.CafeReviewResponse{}, nil)
			}
			h := NewCafeHandler(svc)
			c, w := newTestContext(http.MethodGet, "", gin.Params{{Key: "id", Value: "c1"}}, nil)

			h.GetReview(c)

			assert.Equal(t, tt.wantCode, w.Code)
			e := decodeEnvelope(t, w)
			if tt.wantMsg == "" {
				assert.True(t, e.Success)
			} else {
				assert.Equal(t, tt.wantMsg, e.Error)
			}
		})
	}
}
