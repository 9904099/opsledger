package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func (s *DBStore) CreateInspection(ctx context.Context, record model.InspectionRecord) (model.InspectionRecord, error) {
	record = normalizeInspection(record)
	if err := validateInspection(record); err != nil {
		return model.InspectionRecord{}, err
	}
	if _, err := s.getAsset(ctx, record.AssetID); err != nil {
		return model.InspectionRecord{}, err
	}

	now := time.Now().Format(time.RFC3339)
	record.ID = newID("insp")
	record.CreatedAt = now
	record.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO inspection_records (
			id, asset_id, executor, result, summary, checked_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID, record.AssetID, record.Executor, record.Result, record.Summary, record.CheckedAt, record.CreatedAt, record.UpdatedAt,
	)
	if err != nil {
		return model.InspectionRecord{}, err
	}
	return record, nil
}

func (s *DBStore) ListInspectionAttachments(ctx context.Context) ([]model.InspectionAttachment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, inspection_id, asset_id, file_name, content_type, size_bytes, uploader, description, created_at
		FROM inspection_attachments
		ORDER BY created_at DESC, id DESC
		LIMIT 1000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.InspectionAttachment{}
	for rows.Next() {
		item, err := scanInspectionAttachment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) CreateInspectionAttachment(ctx context.Context, req model.InspectionAttachmentCreateRequest) (model.InspectionAttachment, error) {
	req = normalizeInspectionAttachmentRequest(req)
	if err := validateInspectionAttachmentRequest(req); err != nil {
		return model.InspectionAttachment{}, err
	}
	inspection, err := s.getInspection(ctx, req.InspectionID)
	if err != nil {
		return model.InspectionAttachment{}, err
	}
	now := time.Now().Format(time.RFC3339)
	item := model.InspectionAttachment{
		ID:           newID("att"),
		InspectionID: req.InspectionID,
		AssetID:      inspection.AssetID,
		FileName:     req.FileName,
		ContentType:  req.ContentType,
		SizeBytes:    int64(len(req.Data)),
		Uploader:     req.Uploader,
		Description:  req.Description,
		CreatedAt:    now,
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO inspection_attachments (
			id, inspection_id, asset_id, file_name, content_type, size_bytes, uploader, description, data, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.InspectionID, item.AssetID, item.FileName, item.ContentType, item.SizeBytes,
		item.Uploader, item.Description, req.Data, item.CreatedAt)
	if err != nil {
		return model.InspectionAttachment{}, err
	}
	return item, nil
}

func (s *DBStore) GetInspectionAttachment(ctx context.Context, id string) (model.InspectionAttachment, []byte, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, inspection_id, asset_id, file_name, content_type, size_bytes, uploader, description, created_at, data
		FROM inspection_attachments
		WHERE id = ?
	`, strings.TrimSpace(id))
	var item model.InspectionAttachment
	var data []byte
	if err := row.Scan(
		&item.ID, &item.InspectionID, &item.AssetID, &item.FileName, &item.ContentType, &item.SizeBytes,
		&item.Uploader, &item.Description, &item.CreatedAt, &data,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.InspectionAttachment{}, nil, os.ErrNotExist
		}
		return model.InspectionAttachment{}, nil, err
	}
	return item, data, nil
}

func (s *DBStore) getInspection(ctx context.Context, id string) (model.InspectionRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, asset_id, executor, result, summary, checked_at, created_at, updated_at
		FROM inspection_records
		WHERE id = ?
	`, strings.TrimSpace(id))
	item, err := scanInspection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.InspectionRecord{}, os.ErrNotExist
	}
	return item, err
}

func (s *DBStore) ListProbeAssets(ctx context.Context) ([]model.Asset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, platform_id, platform_code, platform_name, cloud_account_id, cloud_account_name, account_id,
		       project_code, category, resource_type, region, environment, name, endpoint, owner, status, criticality,
		       last_checked_at, tags_csv, notes, specs_json, source, external_id, created_at, updated_at
		FROM assets
		WHERE platform_code = 'cloudflare'
		  AND resource_type = 'DNS Record'
		  AND (
		    `+s.dnsRecordTypePredicate()+`
		    OR tags_csv LIKE '%a%'
		    OR tags_csv LIKE '%cname%'
		  )
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Asset
	for rows.Next() {
		item, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) CreateProbe(ctx context.Context, record model.ProbeRecord) (model.ProbeRecord, error) {
	record.AssetID = strings.TrimSpace(record.AssetID)
	record.URL = strings.TrimSpace(record.URL)
	record.Method = strings.TrimSpace(record.Method)
	record.Status = strings.TrimSpace(record.Status)
	record.Error = strings.TrimSpace(record.Error)
	record.CheckedAt = strings.TrimSpace(record.CheckedAt)
	record.TLSExpiresAt = strings.TrimSpace(record.TLSExpiresAt)
	if record.AssetID == "" {
		return model.ProbeRecord{}, errors.New("asset_id is required")
	}
	if record.URL == "" {
		return model.ProbeRecord{}, errors.New("url is required")
	}
	if record.Method == "" {
		record.Method = "GET"
	}
	if record.Status == "" {
		record.Status = "failed"
	}
	if record.CheckedAt == "" {
		record.CheckedAt = time.Now().Format(time.RFC3339)
	}
	if _, err := s.getAsset(ctx, record.AssetID); err != nil {
		return model.ProbeRecord{}, err
	}

	now := time.Now().Format(time.RFC3339)
	record.ID = newID("probe")
	record.CreatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO probe_records (
			id, asset_id, url, method, status, status_code, latency_ms, error,
			checked_at, tls_expires_at, cert_days_remaining, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID, record.AssetID, record.URL, record.Method, record.Status, record.StatusCode, record.LatencyMS, record.Error,
		record.CheckedAt, record.TLSExpiresAt, record.CertDaysRemaining, record.CreatedAt,
	)
	if err != nil {
		return model.ProbeRecord{}, err
	}
	return record, nil
}

func (s *DBStore) LatestProbe(ctx context.Context, assetID string) (model.ProbeRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, asset_id, url, method, status, status_code, latency_ms, error,
		       checked_at, tls_expires_at, cert_days_remaining, created_at
		FROM probe_records
		WHERE asset_id = ?
		ORDER BY checked_at DESC, created_at DESC
		LIMIT 1
	`, strings.TrimSpace(assetID))
	item, err := scanProbe(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ProbeRecord{}, os.ErrNotExist
	}
	return item, err
}

func (s *DBStore) ListAlerts(ctx context.Context) ([]model.AlertRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ar.id, ar.asset_id, COALESCE(a.name, ''), ar.source, ar.severity, ar.status,
		       ar.title, ar.summary, ar.first_seen_at, ar.last_seen_at, ar.resolved_at,
		       ar.resolved_by, ar.resolution, ar.event_count, ar.created_at, ar.updated_at
		FROM alert_records ar
		LEFT JOIN assets a ON a.id = ar.asset_id
		ORDER BY
			CASE ar.status WHEN 'open' THEN 0 WHEN 'acknowledged' THEN 1 ELSE 2 END,
			ar.last_seen_at DESC,
			ar.created_at DESC
		LIMIT 500
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.AlertRecord{}
	for rows.Next() {
		item, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) UpsertAlert(ctx context.Context, req model.AlertUpsertRequest) (model.AlertRecord, error) {
	req = normalizeAlertUpsertRequest(req)
	if req.AssetID == "" {
		return model.AlertRecord{}, errors.New("asset_id is required")
	}
	if req.Title == "" {
		return model.AlertRecord{}, errors.New("title is required")
	}
	if _, err := s.getAsset(ctx, req.AssetID); err != nil {
		return model.AlertRecord{}, err
	}

	now := time.Now().Format(time.RFC3339)
	var existingID string
	err := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM alert_records
		WHERE asset_id = ? AND source = ? AND title = ? AND status IN ('open', 'acknowledged')
		ORDER BY last_seen_at DESC
		LIMIT 1
	`, req.AssetID, req.Source, req.Title).Scan(&existingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.AlertRecord{}, err
	}
	if existingID != "" {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE alert_records
			SET severity = ?, summary = ?, last_seen_at = ?, event_count = event_count + 1, updated_at = ?
			WHERE id = ?
		`, req.Severity, req.Summary, req.SeenAt, now, existingID); err != nil {
			return model.AlertRecord{}, err
		}
		return s.getAlert(ctx, existingID)
	}

	item := model.AlertRecord{
		ID:          newID("alert"),
		AssetID:     req.AssetID,
		Source:      req.Source,
		Severity:    req.Severity,
		Status:      "open",
		Title:       req.Title,
		Summary:     req.Summary,
		FirstSeenAt: req.SeenAt,
		LastSeenAt:  req.SeenAt,
		EventCount:  1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO alert_records (
			id, asset_id, source, severity, status, title, summary,
			first_seen_at, last_seen_at, resolved_at, resolved_by, resolution,
			event_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.AssetID, item.Source, item.Severity, item.Status, item.Title, item.Summary,
		item.FirstSeenAt, item.LastSeenAt, item.ResolvedAt, item.ResolvedBy, item.Resolution,
		item.EventCount, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return model.AlertRecord{}, err
	}
	return s.getAlert(ctx, item.ID)
}

func (s *DBStore) ResolveAlert(ctx context.Context, id string, req model.AlertResolveRequest) (model.AlertRecord, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.AlertRecord{}, errors.New("id is required")
	}
	req.Resolver = strings.TrimSpace(req.Resolver)
	req.Resolution = strings.TrimSpace(req.Resolution)
	if req.Resolver == "" {
		req.Resolver = "system"
	}
	if req.Resolution == "" {
		req.Resolution = "已处理"
	}
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		UPDATE alert_records
		SET status = 'resolved', resolved_at = ?, resolved_by = ?, resolution = ?, updated_at = ?
		WHERE id = ?
	`, now, req.Resolver, req.Resolution, now, id)
	if err != nil {
		return model.AlertRecord{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return model.AlertRecord{}, err
	}
	return s.getAlert(ctx, id)
}

func (s *DBStore) ResolveOpenAlertsForAssetSource(ctx context.Context, assetID string, source string, resolver string, resolution string) error {
	assetID = strings.TrimSpace(assetID)
	source = strings.TrimSpace(source)
	resolver = strings.TrimSpace(resolver)
	resolution = strings.TrimSpace(resolution)
	if assetID == "" || source == "" {
		return nil
	}
	if resolver == "" {
		resolver = "system"
	}
	if resolution == "" {
		resolution = "自动恢复"
	}
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE alert_records
		SET status = 'resolved', resolved_at = ?, resolved_by = ?, resolution = ?, updated_at = ?
		WHERE asset_id = ? AND source = ? AND status IN ('open', 'acknowledged')
	`, now, resolver, resolution, now, assetID, source)
	return err
}

func (s *DBStore) getAlert(ctx context.Context, id string) (model.AlertRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT ar.id, ar.asset_id, COALESCE(a.name, ''), ar.source, ar.severity, ar.status,
		       ar.title, ar.summary, ar.first_seen_at, ar.last_seen_at, ar.resolved_at,
		       ar.resolved_by, ar.resolution, ar.event_count, ar.created_at, ar.updated_at
		FROM alert_records ar
		LEFT JOIN assets a ON a.id = ar.asset_id
		WHERE ar.id = ?
	`, strings.TrimSpace(id))
	item, err := scanAlert(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.AlertRecord{}, os.ErrNotExist
	}
	return item, err
}

func (s *DBStore) listInspections(ctx context.Context) ([]model.InspectionRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, asset_id, executor, result, summary, checked_at, created_at, updated_at
		FROM inspection_records
		ORDER BY checked_at DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.InspectionRecord
	for rows.Next() {
		item, err := scanInspection(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) listRecentInspections(ctx context.Context, limit int) ([]model.InspectionRecord, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, asset_id, executor, result, summary, checked_at, created_at, updated_at
		FROM inspection_records
		ORDER BY checked_at DESC, created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.InspectionRecord
	for rows.Next() {
		item, err := scanInspection(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) listProbes(ctx context.Context, limit int) ([]model.ProbeRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, asset_id, url, method, status, status_code, latency_ms, error,
		       checked_at, tls_expires_at, cert_days_remaining, created_at
		FROM probe_records
		ORDER BY checked_at DESC, created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ProbeRecord
	for rows.Next() {
		item, err := scanProbe(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
