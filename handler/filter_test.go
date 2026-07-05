package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFilterGet_EnrichParsing(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantEnrich bool
	}{
		{"true enables enrich", "enrich_content=true", true},
		{"false disables enrich", "enrich_content=false", false},
		{"absent defaults to false", "", false},
		{"garbage defaults to false", "enrich_content=banana", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockFilterService{}
			svc.On("Get", mock.Anything, mock.Anything, tt.wantEnrich).Return(&model.FiltersResponse{}, nil)
			h := NewFilterHandler(svc)
			c, w := newTestContext(http.MethodGet, tt.query, nil, nil)

			h.Get(c)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, decodeEnvelope(t, w).Success)
			svc.AssertExpectations(t)
		})
	}
}

func TestFilterGet_Error(t *testing.T) {
	svc := &mockFilterService{}
	svc.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("boom"))
	h := NewFilterHandler(svc)
	c, w := newTestContext(http.MethodGet, "", nil, nil)

	h.Get(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	e := decodeEnvelope(t, w)
	assert.False(t, e.Success)
	assert.Equal(t, "failed to fetch filters", e.Error)
}
