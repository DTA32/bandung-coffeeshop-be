package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// Compile-time checks that each mock satisfies the consumer interface.
var (
	_ cafeService        = (*mockCafeService)(nil)
	_ filterService      = (*mockFilterService)(nil)
	_ locationService    = (*mockLocationService)(nil)
	_ quicksearchService = (*mockQuicksearchService)(nil)
)

// ---- cafeService ----

type mockCafeService struct{ mock.Mock }

func (m *mockCafeService) Search(ctx context.Context, req model.CafeSearchRequest) (*model.CafeSearchResponse, error) {
	args := m.Called(ctx, req)
	res, _ := args.Get(0).(*model.CafeSearchResponse)
	return res, args.Error(1)
}

func (m *mockCafeService) GetByID(ctx context.Context, locationID, lang string) (*model.CafeDetailResponse, error) {
	args := m.Called(ctx, locationID, lang)
	res, _ := args.Get(0).(*model.CafeDetailResponse)
	return res, args.Error(1)
}

func (m *mockCafeService) GetReviewByID(ctx context.Context, locationID, lang string) (*model.CafeReviewResponse, error) {
	args := m.Called(ctx, locationID, lang)
	res, _ := args.Get(0).(*model.CafeReviewResponse)
	return res, args.Error(1)
}

// ---- filterService ----

type mockFilterService struct{ mock.Mock }

func (m *mockFilterService) Get(ctx context.Context, lang string, enrich bool) (*model.FiltersResponse, error) {
	args := m.Called(ctx, lang, enrich)
	res, _ := args.Get(0).(*model.FiltersResponse)
	return res, args.Error(1)
}

// ---- locationService ----

type mockLocationService struct{ mock.Mock }

func (m *mockLocationService) GetByID(ctx context.Context, id, lang string) (*model.LocationDetail, error) {
	args := m.Called(ctx, id, lang)
	res, _ := args.Get(0).(*model.LocationDetail)
	return res, args.Error(1)
}

func (m *mockLocationService) ListDistricts(ctx context.Context, lang string) ([]model.LocationDetail, error) {
	args := m.Called(ctx, lang)
	res, _ := args.Get(0).([]model.LocationDetail)
	return res, args.Error(1)
}

// ---- quicksearchService ----

type mockQuicksearchService struct{ mock.Mock }

func (m *mockQuicksearchService) Quicksearch(ctx context.Context, q, searchType, lang string) ([]model.QuicksearchResult, error) {
	args := m.Called(ctx, q, searchType, lang)
	res, _ := args.Get(0).([]model.QuicksearchResult)
	return res, args.Error(1)
}

// ---- shared test helpers ----

// newTestContext builds a gin context backed by a recorder for handler tests.
// rawQuery is the URL query string without the leading "?"; params are path
// params (e.g. {{Key: "id", Value: "abc"}}); headers may be nil.
func newTestContext(method, rawQuery string, params gin.Params, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	target := "/"
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	c.Params = params
	return c, w
}

// envelope is the decoded JSON response envelope (helper.Success / helper.Error).
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var e envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &e))
	return e
}
