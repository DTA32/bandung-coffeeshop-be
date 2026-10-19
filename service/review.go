package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
)

var ErrInvalidCursor = errors.New("invalid cursor")

const reviewPageSize = 5

type ReviewService struct {
	repo *repository.ReviewRepository
}

func NewReviewService(repo *repository.ReviewRepository) *ReviewService {
	return &ReviewService{repo: repo}
}

// List returns one page of the cafe's visible reviews. cursor is the opaque
// next_cursor of the previous page ("" = first page).
func (s *ReviewService) List(ctx context.Context, locationID, cursor string) (*model.ReviewListResponse, error) {
	var cur *repository.ReviewCursor
	if cursor != "" {
		c, err := decodeReviewCursor(cursor)
		if err != nil {
			return nil, err
		}
		cur = c
	}

	cafeID, err := s.repo.CafeIDByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}

	// One look-ahead row tells whether another page exists.
	rows, err := s.repo.VisibleReviews(ctx, cafeID, cur, reviewPageSize+1)
	if err != nil {
		return nil, err
	}
	var next *string
	if len(rows) > reviewPageSize {
		rows = rows[:reviewPageSize]
		last := rows[len(rows)-1]
		c := encodeReviewCursor(repository.ReviewCursor{IsPinned: last.IsPinned, UpdatedAt: last.UpdatedAtKey, ID: last.ID})
		next = &c
	}

	items, err := s.withRatings(ctx, rows)
	if err != nil {
		return nil, err
	}
	return &model.ReviewListResponse{Items: items, NextCursor: next}, nil
}

// withRatings converts rows to items, loading every row's ratings in one query.
func (s *ReviewService) withRatings(ctx context.Context, rows []repository.ReviewRow) ([]model.ReviewItem, error) {
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	ratings, err := s.repo.RatingsByReviewIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := make([]model.ReviewItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, toReviewItem(r, ratings[r.ID]))
	}
	return items, nil
}

func toReviewItem(r repository.ReviewRow, ratings map[string]float64) model.ReviewItem {
	if ratings == nil {
		ratings = map[string]float64{}
	}
	return model.ReviewItem{
		ID:               r.ID,
		ReviewerName:     r.ReviewerName,
		OverallScore:     r.OverallScore,
		WFCScore:         r.WFCScore,
		Content:          r.Content,
		VisitedAt:        r.VisitedAt,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
		ExternalURLTitle: r.ExternalURLTitle,
		ExternalURLLink:  r.ExternalURLLink,
		Ratings:          ratings,
	}
}

// The cursor is base64url (unpadded) of "<0|1>|<updated_at µs>|<id>".
func encodeReviewCursor(c repository.ReviewCursor) string {
	pinned := "0"
	if c.IsPinned {
		pinned = "1"
	}
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s|%s|%d", pinned, c.UpdatedAt, c.ID)))
}

func decodeReviewCursor(s string) (*repository.ReviewCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || (parts[0] != "0" && parts[0] != "1") {
		return nil, ErrInvalidCursor
	}
	if _, err := time.Parse(repository.ReviewCursorTimeLayout, parts[1]); err != nil {
		return nil, ErrInvalidCursor
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil || id <= 0 {
		return nil, ErrInvalidCursor
	}
	return &repository.ReviewCursor{IsPinned: parts[0] == "1", UpdatedAt: parts[1], ID: id}, nil
}
