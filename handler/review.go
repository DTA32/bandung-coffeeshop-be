package handler

import (
	"errors"
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/middleware"
	"github.com/dta32/bandung-coffeeshop-be/model"
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

// GetMine and the write handlers below sit behind middleware.RequireUser.
func (h *ReviewHandler) GetMine(c *gin.Context) {
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.GetMine(c.Request.Context(), c.Param("id"), u.ID)
	if err != nil {
		h.writeError(c, err, "failed to fetch review")
		return
	}
	c.Header("Cache-Control", "no-store")
	helper.Success(c, res)
}

func (h *ReviewHandler) CreateMine(c *gin.Context) {
	var in model.ReviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		helper.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.CreateMine(c.Request.Context(), c.Param("id"), u.ID, in)
	if err != nil {
		h.writeError(c, err, "failed to save review")
		return
	}
	helper.SuccessStatus(c, http.StatusCreated, res)
}

func (h *ReviewHandler) UpdateMine(c *gin.Context) {
	var in model.ReviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		helper.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	u, _ := middleware.CurrentUser(c)
	res, err := h.svc.UpdateMine(c.Request.Context(), c.Param("id"), u.ID, in)
	if err != nil {
		h.writeError(c, err, "failed to save review")
		return
	}
	helper.Success(c, res)
}

func (h *ReviewHandler) DeleteMine(c *gin.Context) {
	u, _ := middleware.CurrentUser(c)
	if err := h.svc.DeleteMine(c.Request.Context(), c.Param("id"), u.ID); err != nil {
		h.writeError(c, err, "failed to delete review")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ReviewHandler) writeError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrReviewOverallScore),
		errors.Is(err, service.ErrReviewWFCScore),
		errors.Is(err, service.ErrReviewName),
		errors.Is(err, service.ErrReviewVisitedAt),
		errors.Is(err, service.ErrReviewContent),
		errors.Is(err, service.ErrReviewRatingType),
		errors.Is(err, service.ErrReviewRatingScore):
		helper.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrCafeNotFound),
		errors.Is(err, repository.ErrReviewNotFound):
		helper.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrReviewExists):
		helper.Error(c, http.StatusConflict, err.Error())
	default:
		helper.Error(c, http.StatusInternalServerError, fallback)
	}
}
