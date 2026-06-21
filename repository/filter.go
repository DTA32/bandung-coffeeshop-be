package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FilterRepository struct {
	db *pgxpool.Pool
}

func NewFilterRepository(db *pgxpool.Pool) *FilterRepository {
	return &FilterRepository{db: db}
}

type FilterTagRow struct {
	Name        string
	Slug        string
	Description string
}

type FilterRatingRow struct {
	ID              int
	Type            string
	Slug            string
	Name            string
	Description     string
	LongDescription string
	Lower           float64
	Upper           float64
}

// Tags lists every tag that has a usable slug, localized, for the filter
// modal's tag picker.
func (r *FilterRepository) Tags(ctx context.Context, lang string) ([]FilterTagRow, error) {
	nameExpr := localized("$1", "name_indo", "name")
	descExpr := localized("$1", "description_indo", "description")
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s, COALESCE(slug, ''), %s
		FROM tag
		WHERE slug IS NOT NULL AND slug <> ''
		ORDER BY id
	`, nameExpr, descExpr), lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FilterTagRow
	for rows.Next() {
		var row FilterTagRow
		if err := rows.Scan(&row.Name, &row.Slug, &row.Description); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// RatingCategories lists every rating bucket (all types), localized and ordered
// by type then lower_bound, for the filter modal. The service groups these rows
// by type.
func (r *FilterRepository) RatingCategories(ctx context.Context, lang string) ([]FilterRatingRow, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT id, type::text, COALESCE(slug, ''), %s, %s, %s, lower_bound, upper_bound
		FROM rating_category
		ORDER BY type, lower_bound
	`, localized("$1", "name_indo", "name"),
		localized("$1", "short_description_indo", "short_description"),
		localized("$1", "long_description_indo", "long_description")), lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FilterRatingRow
	for rows.Next() {
		var row FilterRatingRow
		if err := rows.Scan(&row.ID, &row.Type, &row.Slug, &row.Name, &row.Description, &row.LongDescription, &row.Lower, &row.Upper); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
