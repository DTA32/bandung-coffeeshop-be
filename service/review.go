package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

// Review body validation errors (400).
var (
	ErrReviewOverallScore = errors.New("overall_score is required and must be between 1 and 5")
	ErrReviewWFCScore     = errors.New("wfc_score must be between 1 and 5")
	ErrReviewName         = errors.New("reviewer_name is required and must be at most 50 characters")
	ErrReviewVisitedAt    = errors.New("visited_at must be YYYY-MM and not in the future")
	ErrReviewContent      = errors.New("content must be at most 2000 characters")
	ErrReviewRatingType   = errors.New("unknown rating type")
	ErrReviewRatingScore  = errors.New("rating score out of range")
)

const (
	reviewScoreMin      = 1
	reviewScoreMax      = 5
	reviewNameMaxLen    = 50
	reviewContentMaxLen = 2000
)

// GetMine returns the user's non-deleted review of the cafe.
func (s *ReviewService) GetMine(ctx context.Context, locationID string, userID int) (*model.MyReview, error) {
	cafeID, err := s.repo.CafeIDByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}
	return s.mine(ctx, cafeID, userID)
}

func (s *ReviewService) mine(ctx context.Context, cafeID, userID int) (*model.MyReview, error) {
	row, err := s.repo.Mine(ctx, cafeID, userID)
	if err != nil {
		return nil, err
	}
	items, err := s.withRatings(ctx, []repository.ReviewRow{*row})
	if err != nil {
		return nil, err
	}
	return &model.MyReview{ReviewItem: items[0], Status: row.Status}, nil
}

// CreateMine submits the user's review of the cafe as pending.
func (s *ReviewService) CreateMine(ctx context.Context, locationID string, userID int, in model.ReviewInput) (*model.MyReview, error) {
	w, err := s.prepareWrite(ctx, locationID, userID, in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, *w); err != nil {
		return nil, err
	}
	return s.mine(ctx, w.CafeID, userID)
}

// UpdateMine overwrites the user's review; it goes back to pending.
func (s *ReviewService) UpdateMine(ctx context.Context, locationID string, userID int, in model.ReviewInput) (*model.MyReview, error) {
	w, err := s.prepareWrite(ctx, locationID, userID, in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, *w); err != nil {
		return nil, err
	}
	return s.mine(ctx, w.CafeID, userID)
}

// DeleteMine soft-deletes the user's review.
func (s *ReviewService) DeleteMine(ctx context.Context, locationID string, userID int) error {
	cafeID, err := s.repo.CafeIDByLocationID(ctx, locationID)
	if err != nil {
		return err
	}
	return s.repo.SoftDelete(ctx, cafeID, userID)
}

// prepareWrite validates the body and resolves the cafe.
func (s *ReviewService) prepareWrite(ctx context.Context, locationID string, userID int, in model.ReviewInput) (*repository.ReviewWrite, error) {
	w := repository.ReviewWrite{UserID: userID, WFCScore: in.WFCScore}

	if in.OverallScore == nil || !inScoreRange(*in.OverallScore) {
		return nil, ErrReviewOverallScore
	}
	w.OverallScore = *in.OverallScore
	if in.WFCScore != nil && !inScoreRange(*in.WFCScore) {
		return nil, ErrReviewWFCScore
	}

	if in.ReviewerName == nil {
		return nil, ErrReviewName
	}
	w.ReviewerName = strings.TrimSpace(*in.ReviewerName)
	if n := utf8.RuneCountInString(w.ReviewerName); n == 0 || n > reviewNameMaxLen {
		return nil, ErrReviewName
	}

	if in.VisitedAt != nil && *in.VisitedAt != "" {
		t, err := time.Parse("2006-01", *in.VisitedAt)
		if err != nil {
			return nil, ErrReviewVisitedAt
		}
		now := time.Now().In(wib)
		if t.After(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)) {
			return nil, ErrReviewVisitedAt
		}
		v := t.Format("2006-01-02")
		w.VisitedAt = &v
	}

	if in.Content != nil {
		c := strings.TrimSpace(*in.Content)
		if utf8.RuneCountInString(c) > reviewContentMaxLen {
			return nil, ErrReviewContent
		}
		if c != "" {
			w.Content = &c
		}
	}

	if len(in.Ratings) > 0 {
		bounds, err := s.repo.RatingBounds(ctx)
		if err != nil {
			return nil, err
		}
		for typ, score := range in.Ratings {
			b, ok := bounds[typ]
			if !ok {
				return nil, fmt.Errorf("%w: %s", ErrReviewRatingType, typ)
			}
			if score < b.Min || score > b.Max {
				return nil, fmt.Errorf("%w: %s must be between %s and %s", ErrReviewRatingScore, typ,
					strconv.FormatFloat(b.Min, 'f', -1, 64), strconv.FormatFloat(b.Max, 'f', -1, 64))
			}
		}
		w.Ratings = in.Ratings
	}

	cafeID, err := s.repo.CafeIDByLocationID(ctx, locationID)
	if err != nil {
		return nil, err
	}
	w.CafeID = cafeID
	return &w, nil
}

func inScoreRange(v float64) bool {
	return v >= reviewScoreMin && v <= reviewScoreMax
}
