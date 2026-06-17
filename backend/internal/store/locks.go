package store

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"ahid-prospect/backend/internal/domain"
)

// ReleaseExpiredLock implements the FR5.2 "lazy check on read" sweep: before
// looking up an active lock for this key, expire it if its TTL has passed.
// Returns the released lock (for an FR8.1 audit entry), or nil if nothing
// needed releasing.
func (s *Store) ReleaseExpiredLock(ctx context.Context, npwp string) (*domain.ProspectLock, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE prospect_locks SET status = 'released', release_reason = 'expired'
		WHERE npwp = $1 AND status = 'active' AND expires_at < now()
		RETURNING id, prospect_id, company_id, npwp, branch_id, marketing_id, locked_at, expires_at, status, release_reason`,
		npwp)
	lock, err := scanLock(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return lock, nil
}

func (s *Store) GetActiveLock(ctx context.Context, npwp string) (*domain.ProspectLock, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, prospect_id, company_id, npwp, branch_id, marketing_id, locked_at, expires_at, status, release_reason
		FROM prospect_locks WHERE npwp = $1 AND status = 'active' LIMIT 1`, npwp)
	lock, err := scanLock(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return lock, nil
}

type CreateLockInput struct {
	ProspectID  string
	CompanyID   *string
	NPWP        string
	BranchID    *string
	MarketingID *string
	TTLDays     int
	Status      string // "active" | "in_conflict"
}

func (s *Store) CreateLock(ctx context.Context, in CreateLockInput) (*domain.ProspectLock, error) {
	ttl := strconv.Itoa(in.TTLDays) + " days"
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO prospect_locks (prospect_id, company_id, npwp, branch_id, marketing_id, expires_at, status)
		VALUES ($1,$2,$3,$4,$5, now() + $6::interval, $7)
		RETURNING id, prospect_id, company_id, npwp, branch_id, marketing_id, locked_at, expires_at, status, release_reason`,
		in.ProspectID, in.CompanyID, in.NPWP, in.BranchID, in.MarketingID, ttl, in.Status,
	)
	return scanLock(row)
}

func (s *Store) ListLockConflicts(ctx context.Context) ([]domain.ProspectLock, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, prospect_id, company_id, npwp, branch_id, marketing_id, locked_at, expires_at, status, release_reason
		FROM prospect_locks WHERE status = 'in_conflict' ORDER BY locked_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ProspectLock
	for rows.Next() {
		l, err := scanLockRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

func scanLock(row pgx.Row) (*domain.ProspectLock, error) {
	l, err := scanLockRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return l, nil
}

func scanLockRow(row rowScanner) (*domain.ProspectLock, error) {
	var l domain.ProspectLock
	if err := row.Scan(&l.ID, &l.ProspectID, &l.CompanyID, &l.NPWP, &l.BranchID, &l.MarketingID,
		&l.LockedAt, &l.ExpiresAt, &l.Status, &l.ReleaseReason); err != nil {
		return nil, err
	}
	return &l, nil
}
