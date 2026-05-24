package repository

import (
	"context"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LocationRepository struct {
	db *pgxpool.Pool
}

func NewLocationRepository(db *pgxpool.Pool) *LocationRepository {
	return &LocationRepository{db: db}
}

func (r *LocationRepository) Quicksearch(ctx context.Context, q, locType string) ([]model.QuicksearchResult, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, type
		 FROM location
		 WHERE name ILIKE '%' || $1 || '%'
		   AND ($2 = '' OR type = $2::location_type_enum)
		   AND status = 'active'
		 ORDER BY similarity(name, $1) DESC
		 LIMIT 10`,
		q, locType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.QuicksearchResult
	for rows.Next() {
		var res model.QuicksearchResult
		if err := rows.Scan(&res.ID, &res.Name, &res.Type); err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	if results == nil {
		results = []model.QuicksearchResult{}
	}
	return results, rows.Err()
}
