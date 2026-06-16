package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ahid-prospect/backend/internal/domain"
)

type CreateProspectInput struct {
	RawCompanyName string
	NPWP           *string
	NIB            *string
	OccupationLOB  *string
	Address        *string
	Domain         *string
	Phone          *string
	Headcount      *int
	SourceBusiness *string
	BranchID       *string
	MarketingID    *string
	Stage          string
}

func (s *Store) CreateProspect(ctx context.Context, in CreateProspectInput) (*domain.Prospect, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO prospects (raw_company_name, npwp, nib, occupation_lob, address, domain, phone,
		                        headcount, source_business, branch_id, marketing_id, stage)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, company_id, raw_company_name, npwp, nib, occupation_lob, address, domain, phone,
		          headcount, source_business, branch_id, marketing_id, stage, status,
		          tanggal_masuk_uw, tanggal_keluar_uw, uw_rationale, created_at`,
		in.RawCompanyName, in.NPWP, in.NIB, in.OccupationLOB, in.Address, in.Domain, in.Phone,
		in.Headcount, in.SourceBusiness, in.BranchID, in.MarketingID, in.Stage,
	)
	return scanProspect(row)
}

func (s *Store) GetProspect(ctx context.Context, id string) (*domain.Prospect, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, company_id, raw_company_name, npwp, nib, occupation_lob, address, domain, phone,
		       headcount, source_business, branch_id, marketing_id, stage, status,
		       tanggal_masuk_uw, tanggal_keluar_uw, uw_rationale, created_at
		FROM prospects WHERE id = $1`, id)
	return scanProspect(row)
}

func (s *Store) ListProspects(ctx context.Context, marketingID *string) ([]domain.Prospect, error) {
	query := `
		SELECT id, company_id, raw_company_name, npwp, nib, occupation_lob, address, domain, phone,
		       headcount, source_business, branch_id, marketing_id, stage, status,
		       tanggal_masuk_uw, tanggal_keluar_uw, uw_rationale, created_at
		FROM prospects`
	args := []any{}
	if marketingID != nil {
		query += ` WHERE marketing_id = $1`
		args = append(args, *marketingID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Prospect
	for rows.Next() {
		p, err := scanProspectRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *Store) SetProspectCompany(ctx context.Context, prospectID, companyID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE prospects SET company_id = $1 WHERE id = $2`, companyID, prospectID)
	return err
}

func (s *Store) SetProspectStage(ctx context.Context, prospectID, stage string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE prospects SET stage = $1 WHERE id = $2`, stage, prospectID)
	return err
}

func (s *Store) RouteToUW(ctx context.Context, id string) (*domain.Prospect, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE prospects SET stage = 'uw', tanggal_masuk_uw = now() WHERE id = $1
		RETURNING id, company_id, raw_company_name, npwp, nib, occupation_lob, address, domain, phone,
		          headcount, source_business, branch_id, marketing_id, stage, status,
		          tanggal_masuk_uw, tanggal_keluar_uw, uw_rationale, created_at`, id)
	return scanProspect(row)
}

func (s *Store) CloseUW(ctx context.Context, id, rationale string) (*domain.Prospect, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE prospects SET tanggal_keluar_uw = now(), uw_rationale = $2 WHERE id = $1
		RETURNING id, company_id, raw_company_name, npwp, nib, occupation_lob, address, domain, phone,
		          headcount, source_business, branch_id, marketing_id, stage, status,
		          tanggal_masuk_uw, tanggal_keluar_uw, uw_rationale, created_at`, id, rationale)
	return scanProspect(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProspect(row pgx.Row) (*domain.Prospect, error) {
	p, err := scanProspectRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func scanProspectRow(row rowScanner) (*domain.Prospect, error) {
	var p domain.Prospect
	if err := row.Scan(&p.ID, &p.CompanyID, &p.RawCompanyName, &p.NPWP, &p.NIB, &p.OccupationLOB,
		&p.Address, &p.Domain, &p.Phone, &p.Headcount, &p.SourceBusiness, &p.BranchID, &p.MarketingID,
		&p.Stage, &p.Status, &p.TanggalMasukUW, &p.TanggalKeluarUW, &p.UWRationale, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}
