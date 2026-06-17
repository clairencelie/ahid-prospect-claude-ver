package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ahid-prospect/backend/internal/domain"
)

const enrichmentJobColumns = `id, company_name, normalized_name, status, result, requested_at, completed_at`

func (s *Store) EnqueueEnrichmentJob(ctx context.Context, companyName, normalizedName string) (*domain.EnrichmentJob, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO enrichment_jobs (company_name, normalized_name)
		VALUES ($1,$2) RETURNING `+enrichmentJobColumns,
		companyName, normalizedName,
	)
	return scanEnrichmentJob(row)
}

// HasPendingEnrichmentJob avoids enqueueing duplicate jobs for the same
// company while one is already queued/running.
func (s *Store) HasPendingEnrichmentJob(ctx context.Context, normalizedName string) (bool, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM enrichment_jobs WHERE normalized_name = $1 AND status IN ('queued','running'))`,
		normalizedName,
	).Scan(&exists)
	return exists, err
}

func (s *Store) GetEnrichmentJob(ctx context.Context, id string) (*domain.EnrichmentJob, error) {
	row := s.Pool.QueryRow(ctx, `SELECT `+enrichmentJobColumns+` FROM enrichment_jobs WHERE id = $1`, id)
	return scanEnrichmentJob(row)
}

func (s *Store) ListQueuedEnrichmentJobs(ctx context.Context, limit int) ([]domain.EnrichmentJob, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+enrichmentJobColumns+` FROM enrichment_jobs
		WHERE status = 'queued' ORDER BY requested_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.EnrichmentJob
	for rows.Next() {
		j, err := scanEnrichmentJobRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func (s *Store) MarkEnrichmentJobRunning(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE enrichment_jobs SET status = 'running' WHERE id = $1`, id)
	return err
}

func (s *Store) CompleteEnrichmentJob(ctx context.Context, id string, status string, result []byte) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE enrichment_jobs SET status = $2, result = $3, completed_at = now() WHERE id = $1`,
		id, status, result,
	)
	return err
}

// FindOrCreateCompany resolves a company by normalized name, creating an
// unverified golden record if the AI-suggested entity isn't in the master
// yet (FR7.3: the new row and any edge attached to it stay unverified until
// an admin verifies them). source is "ai_mock" (fixture) or "gemini" (real
// API suggestion), recorded for provenance.
func (s *Store) FindOrCreateCompany(ctx context.Context, normalizedName, legalName, source string, confidence float64) (*domain.Company, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, npwp, nib, legal_name, normalized_name, brand_names, domain, phone, address,
		       occupation_lob, group_id, source, confidence, verification_status, created_at, updated_at
		FROM companies WHERE normalized_name = $1 LIMIT 1`, normalizedName)

	var c domain.Company
	err := row.Scan(&c.ID, &c.NPWP, &c.NIB, &c.LegalName, &c.NormalizedName, &c.BrandNames, &c.Domain,
		&c.Phone, &c.Address, &c.OccupationLOB, &c.GroupID, &c.Source, &c.Confidence, &c.VerificationStatus,
		&c.CreatedAt, &c.UpdatedAt)
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	insertRow := s.Pool.QueryRow(ctx, `
		INSERT INTO companies (legal_name, normalized_name, source, confidence, verification_status)
		VALUES ($1,$2,$3,$4,'unverified')
		RETURNING id, npwp, nib, legal_name, normalized_name, brand_names, domain, phone, address,
		          occupation_lob, group_id, source, confidence, verification_status, created_at, updated_at`,
		legalName, normalizedName, source, confidence,
	)
	if err := insertRow.Scan(&c.ID, &c.NPWP, &c.NIB, &c.LegalName, &c.NormalizedName, &c.BrandNames, &c.Domain,
		&c.Phone, &c.Address, &c.OccupationLOB, &c.GroupID, &c.Source, &c.Confidence, &c.VerificationStatus,
		&c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateRelationshipUnverified writes the AI-suggested edge, idempotently —
// the worker may rediscover the same edge across runs. If an edge between
// this pair already exists (e.g. a manually-verified one from seed data),
// it's left untouched and returned as-is rather than overwritten.
func (s *Store) CreateRelationshipUnverified(ctx context.Context, parentID, childID, relationType, source string, confidence float64) (*domain.CompanyRelationship, error) {
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO company_relationships (parent_company_id, child_company_id, relation_type, confidence, source, verification_status)
		VALUES ($1,$2,$3,$4,$5,'unverified')
		ON CONFLICT (parent_company_id, child_company_id, relation_type) DO NOTHING
		RETURNING id, parent_company_id, child_company_id, relation_type, confidence, source, verification_status, created_at`,
		parentID, childID, relationType, confidence, source,
	)
	rel, err := scanRelationship(row)
	if err == nil {
		return rel, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	existing := s.Pool.QueryRow(ctx, `
		SELECT id, parent_company_id, child_company_id, relation_type, confidence, source, verification_status, created_at
		FROM company_relationships WHERE parent_company_id = $1 AND child_company_id = $2 AND relation_type = $3`,
		parentID, childID, relationType,
	)
	return scanRelationship(existing)
}

func scanEnrichmentJob(row pgx.Row) (*domain.EnrichmentJob, error) {
	j, err := scanEnrichmentJobRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return j, nil
}

func scanEnrichmentJobRow(row rowScanner) (*domain.EnrichmentJob, error) {
	var j domain.EnrichmentJob
	if err := row.Scan(&j.ID, &j.CompanyName, &j.NormalizedName, &j.Status, &j.Result, &j.RequestedAt, &j.CompletedAt); err != nil {
		return nil, err
	}
	return &j, nil
}
