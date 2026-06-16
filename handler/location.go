package handler

import (
	"errors"
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	svc *service.LocationService
}

func NewLocationHandler(svc *service.LocationService) *LocationHandler {
	return &LocationHandler{svc: svc}
}

func (h *LocationHandler) Quicksearch(c *gin.Context) {
	q := c.Query("q")
	locType := c.Query("type")

	results, err := h.svc.Quicksearch(c.Request.Context(), q, locType)
	if err != nil {
		if errors.Is(err, service.ErrInvalidLocationType) {
			helper.Error(c, http.StatusBadRequest, "invalid location type")
			return
		}
		helper.Error(c, http.StatusInternalServerError, "search failed")
		return
	}

	helper.Success(c, results)
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
