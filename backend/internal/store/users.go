package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ahid-prospect/backend/internal/domain"
)

var ErrNotFound = errors.New("not found")

func (s *Store) GetUser(ctx context.Context, id string) (*domain.User, error) {
	row := s.Pool.QueryRow(ctx, `SELECT id, name, role, branch_id FROM users WHERE id = $1`, id)
	var u domain.User
	if err := row.Scan(&u.ID, &u.Name, &u.Role, &u.BranchID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, role, branch_id FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Role, &u.BranchID); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) ListBranches(ctx context.Context) ([]domain.Branch, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, COALESCE(region, '') FROM branches ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Branch
	for rows.Next() {
		var b domain.Branch
		if err := rows.Scan(&b.ID, &b.Name, &b.Region); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
