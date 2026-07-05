package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/gin-gonic/gin"
)

// filterService is the consumer-defined seam over *service.FilterService so the
// handler can be unit-tested against a mock; the concrete service satisfies it.
type filterService interface {
	Get(ctx context.Context, lang string, enrich bool) (*model.FiltersResponse, error)
}

type FilterHandler struct {
	svc filterService
}

func NewFilterHandler(svc filterService) *FilterHandler {
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
