package handler

import (
	"net/http"

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
	res, err := h.svc.Get(c.Request.Context(), helper.Lang(c))
	if err != nil {
		helper.Error(c, http.StatusInternalServerError, "failed to fetch filters")
		return
	}
	helper.Success(c, res)
}
