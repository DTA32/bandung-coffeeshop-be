package handler

import (
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

type SitemapHandler struct {
	svc *service.SitemapService
}

func NewSitemapHandler(svc *service.SitemapService) *SitemapHandler {
	return &SitemapHandler{svc: svc}
}

func (h *SitemapHandler) Get(c *gin.Context) {
	res, err := h.svc.Get(c.Request.Context())
	if err != nil {
		helper.Error(c, http.StatusInternalServerError, "failed to build sitemap")
		return
	}
	helper.Success(c, res)
}
