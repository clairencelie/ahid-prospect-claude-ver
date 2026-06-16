package store

import (
	"context"
	"encoding/json"

	"ahid-prospect/backend/internal/domain"
)

// WriteAudit inserts an immutable audit_logs row (FR8.1). before/after are
// marshaled as-is; pass nil for either when not applicable (e.g. a create
// has no "before").
func (s *Store) WriteAudit(ctx context.Context, actorID *string, action, entityType string, entityID *string, before, after any) error {
	beforeJSON, err := marshalOrNull(before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalOrNull(after)
	if err != nil {
		return err
	}

	_, err = s.Pool.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, before, after)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		actorID, action, entityType, entityID, beforeJSON, afterJSON,
	)
	return err
}

func marshalOrNull(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func (s *Store) ListAuditLogs(ctx context.Context) ([]domain.AuditLog, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, actor_id, action, entity_type, entity_id, before, after, created_at
		FROM audit_logs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AuditLog
	for rows.Next() {
		var a domain.AuditLog
		if err := rows.Scan(&a.ID, &a.ActorID, &a.Action, &a.EntityType, &a.EntityID, &a.Before, &a.After, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
