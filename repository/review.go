package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A review is publicly visible when it is approved and was written either as
// a partnership review (no user) or by a still-active user. Every query that
// serves or aggregates public reviews must use these two fragments together,
// with cafe_review aliased as rv:
//
//	FROM cafe_review rv ` + visibleReviewJoin + `
//	WHERE ... AND ` + visibleReviewCond
const (
	visibleReviewJoin = `LEFT JOIN users u ON u.id = rv.user_id`
	visibleReviewCond = `rv.status = 'approved' AND (rv.user_id IS NULL OR u.status = 'active')`
)

// reviewColumns are the columns scanned by scanReview, with cafe_review
// aliased as rv. Never select is_subjective / personal_score / content_indo:
// 012_user_reviews_cleanup.sql drops them while this code is live.
const reviewColumns = `rv.id, rv.reviewer_name, rv.status::text,
	rv.overall_score::float8, rv.wfc_score::float8, rv.content,
	TO_CHAR(rv.visited_at, 'YYYY-MM'),
	TO_CHAR(rv.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	TO_CHAR(rv.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	rv.external_url_title, rv.external_url_link,
	rv.is_pinned, TO_CHAR(rv.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.US')`

// ReviewCursorTimeLayout is the Go layout of ReviewRow.UpdatedAtKey (the
// microsecond-precision updated_at used for keyset pagination).
const ReviewCursorTimeLayout = "2006-01-02T15:04:05.000000"

var (
	ErrReviewNotFound = errors.New("review not found")
	ErrReviewExists   = errors.New("you already reviewed this cafe")
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

type ReviewRow struct {
	ID               int
	ReviewerName     string
	Status           string
	OverallScore     *float64
	WFCScore         *float64
	Content          *string
	VisitedAt        *string // "YYYY-MM"
	CreatedAt        string
	UpdatedAt        string
	ExternalURLTitle *string
	ExternalURLLink  *string
	// Keyset position.
	IsPinned     bool
	UpdatedAtKey string // updated_at at µs precision, ReviewCursorTimeLayout
}

// ReviewCursor is the keyset position after which the next page starts.
type ReviewCursor struct {
	IsPinned  bool
	UpdatedAt string // ReviewCursorTimeLayout
	ID        int
}

func scanReview(row pgx.Row) (*ReviewRow, error) {
	var r ReviewRow
	if err := row.Scan(
		&r.ID, &r.ReviewerName, &r.Status,
		&r.OverallScore, &r.WFCScore, &r.Content,
		&r.VisitedAt, &r.CreatedAt, &r.UpdatedAt,
		&r.ExternalURLTitle, &r.ExternalURLLink,
		&r.IsPinned, &r.UpdatedAtKey,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

// CafeIDByLocationID resolves a cafe's location slug to cafe.id, ignoring
// deleted locations.
func (r *ReviewRepository) CafeIDByLocationID(ctx context.Context, locationID string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, `
		SELECT c.id FROM cafe c
		JOIN location l ON l.id = c.location_id
		WHERE l.id = $1 AND l.type = 'cafe' AND l.status <> 'deleted'
	`, locationID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrCafeNotFound
	}
	return id, err
}

// VisibleReviews returns up to limit visible reviews of the cafe ordered
// is_pinned DESC, updated_at DESC, id DESC, starting after cursor (nil = first
// page). All three keys sort descending, so the keyset is a plain row
// comparison with is_pinned cast to int.
func (r *ReviewRepository) VisibleReviews(ctx context.Context, cafeID int, cursor *ReviewCursor, limit int) ([]ReviewRow, error) {
	args := []any{cafeID, limit}
	keyset := ""
	if cursor != nil {
		pinned := 0
		if cursor.IsPinned {
			pinned = 1
		}
		args = append(args, pinned, cursor.UpdatedAt, cursor.ID)
		keyset = `AND (rv.is_pinned::int, rv.updated_at, rv.id) < ($3, $4::timestamp, $5)`
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s
		FROM cafe_review rv
		%s
		WHERE rv.cafe_id = $1 AND %s
		%s
		ORDER BY rv.is_pinned DESC, rv.updated_at DESC, rv.id DESC
		LIMIT $2
	`, reviewColumns, visibleReviewJoin, visibleReviewCond, keyset), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ReviewRow
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *rv)
	}
	return results, rows.Err()
}

// RatingsByReviewIDs returns review id → category type → score.
func (r *ReviewRepository) RatingsByReviewIDs(ctx context.Context, ids []int) (map[int]map[string]float64, error) {
	out := make(map[int]map[string]float64, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT review_id, category_type, score::float8
		FROM cafe_rating
		WHERE review_id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id    int
			typ   string
			score float64
		)
		if err := rows.Scan(&id, &typ, &score); err != nil {
			return nil, err
		}
		if out[id] == nil {
			out[id] = make(map[string]float64)
		}
		out[id][typ] = score
	}
	return out, rows.Err()
}

// Mine returns the user's non-deleted review of the cafe.
func (r *ReviewRepository) Mine(ctx context.Context, cafeID, userID int) (*ReviewRow, error) {
	rv, err := scanReview(r.db.QueryRow(ctx, `
		SELECT `+reviewColumns+`
		FROM cafe_review rv
		WHERE rv.cafe_id = $1 AND rv.user_id = $2 AND rv.status <> 'deleted'
	`, cafeID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReviewNotFound
	}
	return rv, err
}

// RatingBound is the score range a rating type accepts: the span of its
// rating_category buckets.
type RatingBound struct{ Min, Max float64 }

// RatingBounds returns the accepted score range per rating type. Types with
// no buckets are absent (and therefore not rateable).
func (r *ReviewRepository) RatingBounds(ctx context.Context) (map[string]RatingBound, error) {
	rows, err := r.db.Query(ctx, `
		SELECT rtl.type, MIN(rc.lower_bound)::float8, MAX(rc.upper_bound)::float8
		FROM rating_type_label rtl
		JOIN rating_category rc ON rc.type = rtl.type
		GROUP BY rtl.type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]RatingBound)
	for rows.Next() {
		var (
			typ string
			b   RatingBound
		)
		if err := rows.Scan(&typ, &b.Min, &b.Max); err != nil {
			return nil, err
		}
		out[typ] = b
	}
	return out, rows.Err()
}

// ReviewWrite is a validated user review. VisitedAt is "YYYY-MM-01" or nil.
type ReviewWrite struct {
	CafeID, UserID int
	ReviewerName   string
	OverallScore   float64
	WFCScore       *float64
	VisitedAt      *string
	Content        *string
	Ratings        map[string]float64
}

const uniqueViolation = "23505"

// Create inserts the user's review as pending together with its ratings.
// ErrReviewExists when the user already has a non-deleted review of the cafe.
func (r *ReviewRepository) Create(ctx context.Context, w ReviewWrite) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var dummy int
		err := tx.QueryRow(ctx, `
			SELECT 1 FROM cafe_review
			WHERE cafe_id = $1 AND user_id = $2 AND status <> 'deleted'
		`, w.CafeID, w.UserID).Scan(&dummy)
		if err == nil {
			return ErrReviewExists
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		var id int
		err = tx.QueryRow(ctx, `
			INSERT INTO cafe_review (cafe_id, user_id, reviewer_name, status,
			                         overall_score, wfc_score, visited_at, content)
			VALUES ($1, $2, $3, 'pending', $4, $5, $6::date, $7)
			RETURNING id
		`, w.CafeID, w.UserID, w.ReviewerName, w.OverallScore, w.WFCScore, w.VisitedAt, w.Content).Scan(&id)
		if err != nil {
			// A concurrent create lost the race on the one-live-review index.
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == "idx_cafe_review_cafe_user" {
				return ErrReviewExists
			}
			return err
		}
		return replaceRatings(ctx, tx, id, w)
	})
}

// Update overwrites the user's review, sends it back to pending and replaces
// its ratings. ErrReviewNotFound when the user has no non-deleted review.
func (r *ReviewRepository) Update(ctx context.Context, w ReviewWrite) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var id int
		err := tx.QueryRow(ctx, `
			UPDATE cafe_review
			SET reviewer_name = $3, overall_score = $4, wfc_score = $5,
			    visited_at = $6::date, content = $7,
			    status = 'pending', updated_at = now()
			WHERE cafe_id = $1 AND user_id = $2 AND status <> 'deleted'
			RETURNING id
		`, w.CafeID, w.UserID, w.ReviewerName, w.OverallScore, w.WFCScore, w.VisitedAt, w.Content).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrReviewNotFound
		}
		if err != nil {
			return err
		}
		return replaceRatings(ctx, tx, id, w)
	})
}

// replaceRatings swaps the review's cafe_rating rows for w.Ratings.
//
// Until 012_user_reviews_cleanup.sql runs, cafe_rating is still UNIQUE on
// (cafe_id, category_type), so this fails for a second reviewer of an
// already-rated cafe; 012 replaces it with UNIQUE (review_id, category_type).
func replaceRatings(ctx context.Context, tx pgx.Tx, reviewID int, w ReviewWrite) error {
	if _, err := tx.Exec(ctx, `DELETE FROM cafe_rating WHERE review_id = $1`, reviewID); err != nil {
		return err
	}
	if len(w.Ratings) == 0 {
		return nil
	}
	types := make([]string, 0, len(w.Ratings))
	scores := make([]float64, 0, len(w.Ratings))
	for t, s := range w.Ratings {
		types = append(types, t)
		scores = append(scores, s)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO cafe_rating (cafe_id, category_type, score, review_id)
		SELECT $1, x.t, x.s, $2
		FROM unnest($3::text[], $4::float8[]) AS x(t, s)
	`, w.CafeID, reviewID, types, scores)
	return err
}

// SoftDelete marks the user's review deleted. ErrReviewNotFound when there
// is no non-deleted review.
func (r *ReviewRepository) SoftDelete(ctx context.Context, cafeID, userID int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE cafe_review
		SET status = 'deleted', updated_at = now()
		WHERE cafe_id = $1 AND user_id = $2 AND status <> 'deleted'
	`, cafeID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrReviewNotFound
	}
	return nil
}
