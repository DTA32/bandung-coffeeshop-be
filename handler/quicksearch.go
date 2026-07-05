package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

// quicksearchService is the consumer-defined seam over *service.QuicksearchService
// so the handler can be unit-tested against a mock; the concrete service satisfies it.
type quicksearchService interface {
	Quicksearch(ctx context.Context, q, searchType, lang string) ([]model.QuicksearchResult, error)
}

type QuicksearchHandler struct {
	svc quicksearchService
}

func NewQuicksearchHandler(svc quicksearchService) *QuicksearchHandler {
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
