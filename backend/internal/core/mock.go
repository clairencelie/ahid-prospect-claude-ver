package core

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MockCoreClient reads the seeded clients/policies tables, standing in for
// the real core system API in the demo (§9).
type MockCoreClient struct {
	pool *pgxpool.Pool
}

func NewMockCoreClient(pool *pgxpool.Pool) *MockCoreClient {
	return &MockCoreClient{pool: pool}
}

func (m *MockCoreClient) LookupClientByNPWP(ctx context.Context, npwp string) (*Client, error) {
	row := m.pool.QueryRow(ctx,
		`SELECT id, npwp, client_name, owning_branch_id FROM clients WHERE npwp = $1 LIMIT 1`, npwp)

	var c Client
	if err := row.Scan(&c.ID, &c.NPWP, &c.ClientName, &c.OwningBranchID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (m *MockCoreClient) ActivePoliciesByNPWP(ctx context.Context, npwp string) ([]Policy, error) {
	rows, err := m.pool.Query(ctx,
		`SELECT id, policy_no, status, period_start::text, period_end::text, owning_branch_id
		 FROM policies WHERE npwp = $1 AND status = 'active'`, npwp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Policy
	for rows.Next() {
		var p Policy
		var start, end *string
		if err := rows.Scan(&p.ID, &p.PolicyNo, &p.Status, &start, &end, &p.OwningBranchID); err != nil {
			return nil, err
		}
		p.PeriodStart, p.PeriodEnd = start, end
		out = append(out, p)
	}
	return out, rows.Err()
}
