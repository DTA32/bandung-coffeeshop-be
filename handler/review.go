package handler

import (
	"errors"
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) List(c *gin.Context) {
	res, err := h.svc.List(c.Request.Context(), c.Param("id"), c.Query("cursor"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCursor):
			helper.Error(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, repository.ErrCafeNotFound):
			helper.Error(c, http.StatusNotFound, err.Error())
		default:
			helper.Error(c, http.StatusInternalServerError, "failed to fetch reviews")
		}
		return
	}
	helper.Success(c, res)
}
