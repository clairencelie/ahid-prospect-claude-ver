package store

import "context"

type MonitoringSummary struct {
	BandCounts            map[string]int `json:"band_counts"`
	ReviewQueueSize       int            `json:"review_queue_size"`
	LockConflictQueueSize int            `json:"lock_conflict_queue_size"`
	LocksActive           int            `json:"locks_active"`
	LocksExpired          int            `json:"locks_expired"`
	EdgesVerified         int            `json:"edges_verified"`
	EdgesUnverified       int            `json:"edges_unverified"`
}

// ComputeMonitoringSummary implements FR8.2's dashboard counts.
func (s *Store) ComputeMonitoringSummary(ctx context.Context) (*MonitoringSummary, error) {
	summary := &MonitoringSummary{BandCounts: map[string]int{"PASS": 0, "REVIEW": 0, "BLOCK": 0}}

	bandRows, err := s.Pool.Query(ctx, `SELECT band, count(*) FROM match_results GROUP BY band`)
	if err != nil {
		return nil, err
	}
	for bandRows.Next() {
		var band string
		var count int
		if err := bandRows.Scan(&band, &count); err != nil {
			bandRows.Close()
			return nil, err
		}
		summary.BandCounts[band] = count
	}
	bandRows.Close()
	if err := bandRows.Err(); err != nil {
		return nil, err
	}

	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM match_results WHERE band = 'REVIEW' AND checker_id IS NULL`,
	).Scan(&summary.ReviewQueueSize); err != nil {
		return nil, err
	}

	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM prospect_locks WHERE status = 'in_conflict'`,
	).Scan(&summary.LockConflictQueueSize); err != nil {
		return nil, err
	}

	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM prospect_locks WHERE status = 'active' AND expires_at >= now()`,
	).Scan(&summary.LocksActive); err != nil {
		return nil, err
	}

	if err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM prospect_locks
		WHERE (status = 'active' AND expires_at < now()) OR (status = 'released' AND release_reason = 'expired')`,
	).Scan(&summary.LocksExpired); err != nil {
		return nil, err
	}

	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM company_relationships WHERE verification_status = 'verified'`,
	).Scan(&summary.EdgesVerified); err != nil {
		return nil, err
	}
	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM company_relationships WHERE verification_status = 'unverified'`,
	).Scan(&summary.EdgesUnverified); err != nil {
		return nil, err
	}

	return summary, nil
}
