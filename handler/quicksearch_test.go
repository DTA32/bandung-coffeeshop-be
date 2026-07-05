package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestQuicksearch_Mapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		result   []model.QuicksearchResult
		wantCode int
		wantMsg  string // "" => success
	}{
		{"invalid type maps to 400", service.ErrInvalidSearchType, nil, http.StatusBadRequest, "invalid type"},
		{"other error maps to 500", errors.New("boom"), nil, http.StatusInternalServerError, "search failed"},
		{"success", nil, []model.QuicksearchResult{{ID: "x"}}, http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockQuicksearchService{}
			// Lock q/type forwarding: a swapped or wrong-key read would miss the expectation.
			svc.On("Quicksearch", mock.Anything, "kopi", "all", mock.Anything).Return(tt.result, tt.err)
			h := NewQuicksearchHandler(svc)
			c, w := newTestContext(http.MethodGet, "q=kopi&type=all", nil, nil)

			h.Quicksearch(c)

			assert.Equal(t, tt.wantCode, w.Code)
			e := decodeEnvelope(t, w)
			if tt.wantMsg == "" {
				assert.True(t, e.Success)
			} else {
				assert.False(t, e.Success)
				assert.Equal(t, tt.wantMsg, e.Error)
			}
		})
	}
}
