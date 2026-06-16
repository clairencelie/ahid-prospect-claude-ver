package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ahid-prospect/backend/internal/domain"
)

type CreateRelationshipInput struct {
	ParentCompanyID string
	ChildCompanyID  string
	RelationType    string
	Confidence      float64
	Source          string
}

func (s *Store) CreateRelationship(ctx context.Context, in CreateRelationshipInput) (*domain.CompanyRelationship, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO company_relationships (parent_company_id, child_company_id, relation_type, confidence, source)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, parent_company_id, child_company_id, relation_type, confidence, source, verification_status, created_at`,
		in.ParentCompanyID, in.ChildCompanyID, in.RelationType, in.Confidence, in.Source,
	)
	return scanRelationship(row)
}

func (s *Store) GetRelationship(ctx context.Context, id string) (*domain.CompanyRelationship, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, parent_company_id, child_company_id, relation_type, confidence, source, verification_status, created_at
		FROM company_relationships WHERE id = $1`, id)
	return scanRelationship(row)
}

func (s *Store) ListRelationshipsForCompany(ctx context.Context, companyID string) ([]domain.CompanyRelationship, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, parent_company_id, child_company_id, relation_type, confidence, source, verification_status, created_at
		FROM company_relationships
		WHERE parent_company_id = $1 OR child_company_id = $1
		ORDER BY created_at`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.CompanyRelationship
	for rows.Next() {
		r, err := scanRelationshipRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *Store) SetRelationshipVerification(ctx context.Context, id string, status domain.VerificationStatus) (*domain.CompanyRelationship, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE company_relationships SET verification_status = $2 WHERE id = $1
		RETURNING id, parent_company_id, child_company_id, relation_type, confidence, source, verification_status, created_at`,
		id, status,
	)
	return scanRelationship(row)
}

func scanRelationship(row pgx.Row) (*domain.CompanyRelationship, error) {
	r, err := scanRelationshipRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return r, nil
}

func scanRelationshipRow(row rowScanner) (*domain.CompanyRelationship, error) {
	var r domain.CompanyRelationship
	if err := row.Scan(&r.ID, &r.ParentCompanyID, &r.ChildCompanyID, &r.RelationType, &r.Confidence,
		&r.Source, &r.VerificationStatus, &r.CreatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}
