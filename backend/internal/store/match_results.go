package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"ahid-prospect/backend/internal/domain"
)

const matchResultColumns = `id, prospect_id, candidate_company_id, composite_score, band, reason, matched_signals,
	          decided_by, maker_id, checker_id, decision, created_at`

type InsertMatchResultInput struct {
	ProspectID         string
	CandidateCompanyID *string
	CompositeScore     float64
	Band               domain.Band
	Reason             string
	MatchedSignals     []domain.MatchedSignal
}

func (s *Store) InsertMatchResult(ctx context.Context, in InsertMatchResultInput) (*domain.MatchResult, error) {
	signalsJSON, err := json.Marshal(in.MatchedSignals)
	if err != nil {
		return nil, err
	}

	row := s.Pool.QueryRow(ctx, `
		INSERT INTO match_results (prospect_id, candidate_company_id, composite_score, band, reason, matched_signals)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING `+matchResultColumns,
		in.ProspectID, in.CandidateCompanyID, in.CompositeScore, in.Band, in.Reason, signalsJSON,
	)
	return scanMatchResult(row)
}

func (s *Store) GetMatchResult(ctx context.Context, id string) (*domain.MatchResult, error) {
	row := s.Pool.QueryRow(ctx, `SELECT `+matchResultColumns+` FROM match_results WHERE id = $1`, id)
	return scanMatchResult(row)
}

// ListReviewQueue returns REVIEW-band results still awaiting a final
// checker decision (FR6.1) — once a checker approves, checker_id is set and
// the item drops out of the queue.
func (s *Store) ListReviewQueue(ctx context.Context) ([]domain.MatchResult, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+matchResultColumns+`
		FROM match_results WHERE band = 'REVIEW' AND checker_id IS NULL
		ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.MatchResult
	for rows.Next() {
		mr, err := scanMatchResultRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *mr)
	}
	return out, rows.Err()
}

// SetMakerDecision records the branch_co's proposed resolution (FR6.2).
func (s *Store) SetMakerDecision(ctx context.Context, id, makerID, decision string) (*domain.MatchResult, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE match_results SET maker_id = $2, decision = $3, decided_by = 'user'
		WHERE id = $1 RETURNING `+matchResultColumns,
		id, makerID, decision,
	)
	return scanMatchResult(row)
}

// SetCheckerApproval finalizes (approve=true) or sends the item back to the
// maker (approve=false, clearing the prior decision) per FR6.2.
func (s *Store) SetCheckerApproval(ctx context.Context, id, checkerID string, approve bool) (*domain.MatchResult, error) {
	var row pgx.Row
	if approve {
		row = s.Pool.QueryRow(ctx, `
			UPDATE match_results SET checker_id = $2
			WHERE id = $1 RETURNING `+matchResultColumns,
			id, checkerID,
		)
	} else {
		row = s.Pool.QueryRow(ctx, `
			UPDATE match_results SET maker_id = NULL, decision = NULL
			WHERE id = $1 RETURNING `+matchResultColumns,
			id,
		)
	}
	return scanMatchResult(row)
}

func scanMatchResult(row pgx.Row) (*domain.MatchResult, error) {
	mr, err := scanMatchResultRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return mr, nil
}

func scanMatchResultRow(row rowScanner) (*domain.MatchResult, error) {
	var mr domain.MatchResult
	var rawSignals []byte
	if err := row.Scan(&mr.ID, &mr.ProspectID, &mr.CandidateCompanyID, &mr.CompositeScore, &mr.Band,
		&mr.Reason, &rawSignals, &mr.DecidedBy, &mr.MakerID, &mr.CheckerID, &mr.Decision, &mr.CreatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rawSignals, &mr.MatchedSignals); err != nil {
		return nil, err
	}
	return &mr, nil
}
