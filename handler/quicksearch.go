package handler

import (
	"errors"
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

type QuicksearchHandler struct {
	svc *service.QuicksearchService
}

func NewQuicksearchHandler(svc *service.QuicksearchService) *QuicksearchHandler {
	return &QuicksearchHandler{svc: svc}
}

func (h *QuicksearchHandler) Quicksearch(c *gin.Context) {
	q := c.Query("q")
	searchType := c.Query("type")

	results, err := h.svc.Quicksearch(c.Request.Context(), q, searchType, helper.Lang(c))
	if err != nil {
		if errors.Is(err, service.ErrInvalidSearchType) {
			helper.Error(c, http.StatusBadRequest, "invalid type")
			return
		}
		helper.Error(c, http.StatusInternalServerError, "search failed")
		return
	}

	helper.Success(c, results)
}
