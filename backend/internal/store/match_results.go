package store

import (
	"context"
	"encoding/json"

	"ahid-prospect/backend/internal/domain"
)

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
		RETURNING id, prospect_id, candidate_company_id, composite_score, band, reason, matched_signals,
		          decided_by, maker_id, checker_id, decision, created_at`,
		in.ProspectID, in.CandidateCompanyID, in.CompositeScore, in.Band, in.Reason, signalsJSON,
	)

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
