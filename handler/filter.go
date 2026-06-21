package handler

import (
	"net/http"
	"strconv"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

type FilterHandler struct {
	svc *service.FilterService
}

func NewFilterHandler(svc *service.FilterService) *FilterHandler {
	return &FilterHandler{svc: svc}
}

func (h *FilterHandler) Get(c *gin.Context) {
	// enrich_content (default false) opts into the heavier tag/rating blurbs that
	// the SRP page renders; the filter modal omits it for a lighter payload.
	enrich, _ := strconv.ParseBool(c.Query("enrich_content"))
	res, err := h.svc.Get(c.Request.Context(), helper.Lang(c), enrich)
	if err != nil {
		helper.Error(c, http.StatusInternalServerError, "failed to fetch filters")
		return
	}
	helper.Success(c, res)
}
