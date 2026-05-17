package handler

import (
	"errors"
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/helper"
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
