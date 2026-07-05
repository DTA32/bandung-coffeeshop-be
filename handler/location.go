package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

// locationService is the consumer-defined seam over *service.LocationService so
// the handler can be unit-tested against a mock; the concrete service satisfies it.
type locationService interface {
	GetByID(ctx context.Context, id, lang string) (*model.LocationDetail, error)
	ListDistricts(ctx context.Context, lang string) ([]model.LocationDetail, error)
}

type LocationHandler struct {
	svc locationService
}

func NewLocationHandler(svc locationService) *LocationHandler {
	return &LocationHandler{svc: svc}
}

func (h *LocationHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	res, err := h.svc.GetByID(c.Request.Context(), id, helper.Lang(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLocationIsCafe):
			helper.Error(c, http.StatusBadRequest, "location is a cafe; use the cafe endpoint")
		case errors.Is(err, repository.ErrLocationNotFound):
			helper.Error(c, http.StatusNotFound, "location not found")
		default:
			helper.Error(c, http.StatusInternalServerError, "failed to fetch location")
		}
		return
	}

	helper.Success(c, res)
}

func (h *LocationHandler) List(c *gin.Context) {
	res, err := h.svc.ListDistricts(c.Request.Context(), helper.Lang(c))
	if err != nil {
		helper.Error(c, http.StatusInternalServerError, "failed to list districts")
		return
	}

	helper.Success(c, res)
}
