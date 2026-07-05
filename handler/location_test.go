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
)

func TestLocationGetByID_Mapping(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string // "" => success
	}{
		{"cafe maps to 400", service.ErrLocationIsCafe, http.StatusBadRequest, "location is a cafe; use the cafe endpoint"},
		{"not found maps to 404", repository.ErrLocationNotFound, http.StatusNotFound, "location not found"},
		{"other maps to 500", errors.New("boom"), http.StatusInternalServerError, "failed to fetch location"},
		{"success", nil, http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockLocationService{}
			if tt.err != nil {
				svc.On("GetByID", mock.Anything, "d1", mock.Anything).Return(nil, tt.err)
			} else {
				svc.On("GetByID", mock.Anything, "d1", mock.Anything).Return(&model.LocationDetail{ID: "d1"}, nil)
			}
			h := NewLocationHandler(svc)
			c, w := newTestContext(http.MethodGet, "", gin.Params{{Key: "id", Value: "d1"}}, nil)

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

func TestLocationList(t *testing.T) {
	t.Run("error maps to 500", func(t *testing.T) {
		svc := &mockLocationService{}
		svc.On("ListDistricts", mock.Anything, mock.Anything).Return(nil, errors.New("boom"))
		h := NewLocationHandler(svc)
		c, w := newTestContext(http.MethodGet, "", nil, nil)

		h.List(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "failed to list districts", decodeEnvelope(t, w).Error)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockLocationService{}
		svc.On("ListDistricts", mock.Anything, mock.Anything).Return([]model.LocationDetail{{ID: "d1"}}, nil)
		h := NewLocationHandler(svc)
		c, w := newTestContext(http.MethodGet, "", nil, nil)

		h.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, decodeEnvelope(t, w).Success)
	})
}
