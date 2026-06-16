package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/dta32/bandung-coffeeshop-be/service"
	"github.com/gin-gonic/gin"
)

// parseCSV splits a comma-separated query value into trimmed, de-duplicated,
// non-empty parts.
func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, p := range strings.Split(raw, ",") {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// parseIntCSV is parseCSV for integers, erroring on any non-integer part.
func parseIntCSV(raw string) ([]int, error) {
	seen := make(map[int]struct{})
	var out []int
	for _, p := range strings.Split(raw, ",") {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out, nil
}

type CafeHandler struct {
	svc *service.CafeService
}

func NewCafeHandler(svc *service.CafeService) *CafeHandler {
	return &CafeHandler{svc: svc}
}

func (h *CafeHandler) Search(c *gin.Context) {
	req := model.CafeSearchRequest{
		QueryID:   c.Query("query_id"),
		QueryType: c.Query("query_type"),
		Tags:      parseCSV(c.Query("tags")),
		OpenHour:  c.Query("open_hour"),
		Lang:      helper.Lang(c),
		Sort:      c.Query("sort"),
		Order:     c.Query("order"),
	}

	if raw := c.Query("ratings"); raw != "" {
		ids, err := parseIntCSV(raw)
		if err != nil {
			helper.Error(c, http.StatusBadRequest, "invalid ratings")
			return
		}
		req.RatingIDs = ids
	}

	if raw := c.Query("price_min"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 || v > 999999 {
			helper.Error(c, http.StatusBadRequest, "invalid price_min")
			return
		}
		req.PriceMin = &v
	}

	if raw := c.Query("price_max"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 || v > 999999 {
			helper.Error(c, http.StatusBadRequest, "invalid price_max")
			return
		}
		req.PriceMax = &v
	}

	if raw := c.Query("query_coords"); raw != "" {
		coords, err := service.ParseCoords(raw)
		if err != nil {
			helper.Error(c, http.StatusBadRequest, "invalid query_coords")
			return
		}
		req.QueryCoords = coords
	}

	if raw := c.Query("radius_max"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			helper.Error(c, http.StatusBadRequest, "invalid radius_max")
			return
		}
		req.RadiusMax = &v
	}

	if raw := c.Query("is_featured"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			helper.Error(c, http.StatusBadRequest, "invalid is_featured")
			return
		}
		req.IsFeatured = &v
	}

	if raw := c.Query("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			helper.Error(c, http.StatusBadRequest, "invalid page")
			return
		}
		req.Page = v
	}

	if raw := c.Query("size"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			helper.Error(c, http.StatusBadRequest, "invalid size")
			return
		}
		req.Size = v
	}

	res, err := h.svc.Search(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidLocationType),
			errors.Is(err, service.ErrInvalidSort),
			errors.Is(err, service.ErrInvalidOrder),
			errors.Is(err, service.ErrInvalidCoords),
			errors.Is(err, service.ErrCoordsConflictsWithID),
			errors.Is(err, service.ErrQueryTypeWithoutID),
			errors.Is(err, service.ErrIDWithoutType),
			errors.Is(err, service.ErrDistanceSortNeedsCoords),
			errors.Is(err, service.ErrInvalidRatingCategory),
			errors.Is(err, service.ErrRatingSlugWithoutType),
			errors.Is(err, service.ErrInvalidOpenHour),
			errors.Is(err, service.ErrInvalidPriceRange),
			errors.Is(err, service.ErrDuplicateRatingType):
			helper.Error(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, repository.ErrFocusNotFound),
			errors.Is(err, repository.ErrRatingCategoryNotFound),
			errors.Is(err, repository.ErrTagNotFound):
			helper.Error(c, http.StatusNotFound, err.Error())
		default:
			helper.Error(c, http.StatusInternalServerError, "search failed")
		}
		return
	}

	helper.Success(c, res)
}

func (h *CafeHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	res, err := h.svc.GetByID(c.Request.Context(), id, helper.Lang(c))
	if err != nil {
		if errors.Is(err, repository.ErrCafeNotFound) {
			helper.Error(c, http.StatusNotFound, err.Error())
		} else {
			helper.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	helper.Success(c, res)
}

func (h *CafeHandler) GetReview(c *gin.Context) {
	id := c.Param("id")
	res, err := h.svc.GetReviewByID(c.Request.Context(), id, helper.Lang(c))
	if err != nil {
		if errors.Is(err, repository.ErrCafeNotFound) {
			helper.Error(c, http.StatusNotFound, err.Error())
		} else {
			helper.Error(c, http.StatusInternalServerError, "failed to fetch review")
		}
		return
	}
	helper.Success(c, res)
}
