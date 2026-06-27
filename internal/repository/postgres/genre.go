package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arrowwhi/books_review_bot/internal/domain"
	"github.com/arrowwhi/books_review_bot/internal/repository"
)

type GenreRepo struct {
	pool *pgxpool.Pool
}

func NewGenreRepo(pool *pgxpool.Pool) repository.GenreRepository {
	return &GenreRepo{pool: pool}
}

func (r *GenreRepo) List(ctx context.Context) ([]*domain.Genre, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, is_default FROM genres ORDER BY is_default DESC, name`)
	if err != nil {
		return nil, fmt.Errorf("postgres.Genre.List: %w", err)
	}
	defer rows.Close()

	var genres []*domain.Genre
	for rows.Next() {
		var g domain.Genre
		if err := rows.Scan(&g.ID, &g.Name, &g.IsDefault); err != nil {
			return nil, fmt.Errorf("postgres.Genre.List scan: %w", err)
		}
		genres = append(genres, &g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.Genre.List rows: %w", err)
	}

	return genres, nil
}

func (r *GenreRepo) GetByID(ctx context.Context, id int32) (*domain.Genre, error) {
	var g domain.Genre
	err := r.pool.QueryRow(ctx, `SELECT id, name, is_default FROM genres WHERE id=$1`, id).
		Scan(&g.ID, &g.Name, &g.IsDefault)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("postgres.Genre.GetByID: %w", err)
	}
	return &g, nil
}

func (r *GenreRepo) Create(ctx context.Context, name string) (*domain.Genre, error) {
	var g domain.Genre
	err := r.pool.QueryRow(ctx, `INSERT INTO genres (name) VALUES ($1) RETURNING id, name, is_default`, name).
		Scan(&g.ID, &g.Name, &g.IsDefault)
	if err != nil {
		return nil, fmt.Errorf("postgres.Genre.Create: %w", err)
	}
	return &g, nil
}
