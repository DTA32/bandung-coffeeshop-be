package repository

import (
	"context"
	"errors"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

const userColumns = `id, provider, provider_sub, email, name, avatar_url, status::text, created_at, updated_at`

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

type UpsertUserParams struct {
	Provider    string
	ProviderSub string
	Email       string
	Name        string
	AvatarURL   *string
}

// UpsertByProvider refreshes the profile on every login but never touches
// status, so a deactivated account stays deactivated.
func (r *UserRepository) UpsertByProvider(ctx context.Context, p UpsertUserParams) (*model.User, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO users (provider, provider_sub, email, name, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (provider, provider_sub) DO UPDATE
		SET email      = EXCLUDED.email,
		    name       = EXCLUDED.name,
		    avatar_url = EXCLUDED.avatar_url,
		    updated_at = now()
		RETURNING `+userColumns,
		p.Provider, p.ProviderSub, p.Email, p.Name, p.AvatarURL)
	return scanUser(row)
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*model.User, error) {
	row := r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return u, err
}

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	if err := row.Scan(&u.ID, &u.Provider, &u.ProviderSub, &u.Email, &u.Name, &u.AvatarURL, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}
