package store

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func (s *DBStore) ListAuditEvents(ctx context.Context, limit int) ([]model.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor, actor_role, action, target_type, target_id, target_name,
		       outcome, ip, user_agent, summary, metadata_json, created_at
		FROM audit_events
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.AuditEvent{}
	for rows.Next() {
		item, err := scanAuditEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) RecordAuditEvent(ctx context.Context, event model.AuditEvent) error {
	now := time.Now().Format(time.RFC3339)
	if strings.TrimSpace(event.ID) == "" {
		event.ID = newID("audit")
	}
	if strings.TrimSpace(event.CreatedAt) == "" {
		event.CreatedAt = now
	}
	metadataJSON := "{}"
	if len(event.Metadata) > 0 {
		payload, err := json.Marshal(event.Metadata)
		if err != nil {
			return err
		}
		metadataJSON = string(payload)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_events (
			id, actor, actor_role, action, target_type, target_id, target_name,
			outcome, ip, user_agent, summary, metadata_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.Actor, event.ActorRole, event.Action, event.TargetType, event.TargetID, event.TargetName,
		event.Outcome, event.IP, event.UserAgent, event.Summary, metadataJSON, event.CreatedAt)
	return err
}
