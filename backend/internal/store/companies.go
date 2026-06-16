package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ahid-prospect/backend/internal/domain"
)

// CompanyCandidate is a company master row plus the relationship-graph facts
// the matching engine needs (FR3.1 "same group" signal / FR4.3 verified vs
// unverified), pre-resolved here so the matching package stays DB-free.
type CompanyCandidate struct {
	Company                domain.Company
	HasVerifiedGroupLink   bool
	HasUnverifiedGroupLink bool
}

// ListCandidateCompanies returns every company in the master for the
// matching engine to score against. The seed dataset is demo-sized, so a
// full scan in Go is simpler and just as correct as a DB-side prefilter.
func (s *Store) ListCandidateCompanies(ctx context.Context) ([]CompanyCandidate, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT c.id, c.npwp, c.nib, c.legal_name, c.normalized_name, c.brand_names, c.domain, c.phone,
		       c.address, c.occupation_lob, c.group_id, c.source, c.confidence, c.verification_status,
		       c.created_at, c.updated_at,
		       EXISTS (SELECT 1 FROM company_relationships r
		               WHERE (r.parent_company_id = c.id OR r.child_company_id = c.id)
		                 AND r.verification_status = 'verified') AS has_verified_group,
		       EXISTS (SELECT 1 FROM company_relationships r
		               WHERE (r.parent_company_id = c.id OR r.child_company_id = c.id)
		                 AND r.verification_status = 'unverified') AS has_unverified_group
		FROM companies c`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompanyCandidate
	for rows.Next() {
		var cc CompanyCandidate
		c := &cc.Company
		if err := rows.Scan(&c.ID, &c.NPWP, &c.NIB, &c.LegalName, &c.NormalizedName, &c.BrandNames,
			&c.Domain, &c.Phone, &c.Address, &c.OccupationLOB, &c.GroupID, &c.Source, &c.Confidence,
			&c.VerificationStatus, &c.CreatedAt, &c.UpdatedAt,
			&cc.HasVerifiedGroupLink, &cc.HasUnverifiedGroupLink); err != nil {
			return nil, err
		}
		out = append(out, cc)
	}
	return out, rows.Err()
}

func (s *Store) GetCompany(ctx context.Context, id string) (*domain.Company, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, npwp, nib, legal_name, normalized_name, brand_names, domain, phone, address,
		       occupation_lob, group_id, source, confidence, verification_status, created_at, updated_at
		FROM companies WHERE id = $1`, id)

	var c domain.Company
	if err := row.Scan(&c.ID, &c.NPWP, &c.NIB, &c.LegalName, &c.NormalizedName, &c.BrandNames, &c.Domain,
		&c.Phone, &c.Address, &c.OccupationLOB, &c.GroupID, &c.Source, &c.Confidence, &c.VerificationStatus,
		&c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (s *Store) SearchCompanies(ctx context.Context, q string) ([]domain.Company, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, npwp, nib, legal_name, normalized_name, brand_names, domain, phone, address,
		       occupation_lob, group_id, source, confidence, verification_status, created_at, updated_at
		FROM companies
		WHERE $1 = '' OR normalized_name ILIKE '%' || $1 || '%' OR legal_name ILIKE '%' || $1 || '%'
		ORDER BY legal_name`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Company
	for rows.Next() {
		var c domain.Company
		if err := rows.Scan(&c.ID, &c.NPWP, &c.NIB, &c.LegalName, &c.NormalizedName, &c.BrandNames, &c.Domain,
			&c.Phone, &c.Address, &c.OccupationLOB, &c.GroupID, &c.Source, &c.Confidence, &c.VerificationStatus,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
