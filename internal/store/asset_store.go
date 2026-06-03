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

func (s *DBStore) CreateAsset(ctx context.Context, asset model.Asset) (model.Asset, error) {
	asset = normalizeAsset(asset)
	if err := validateAsset(asset); err != nil {
		return model.Asset{}, err
	}

	now := time.Now().Format(time.RFC3339)
	asset.ID = newID("asset")
	asset.CreatedAt = now
	asset.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO assets (
			id, platform_id, platform_code, platform_name, cloud_account_id, cloud_account_name, account_id,
			project_code, category, resource_type, region, environment, name, endpoint, owner, status, criticality,
			last_checked_at, tags_csv, notes, specs_json, source, external_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		asset.ID, asset.PlatformID, asset.PlatformCode, asset.PlatformName, asset.CloudAccountID, asset.CloudAccountName, asset.AccountID,
		asset.ProjectCode, asset.Category, asset.ResourceType, asset.Region, asset.Environment, asset.Name, asset.Endpoint,
		asset.Owner, asset.Status, asset.Criticality, asset.LastCheckedAt, joinTags(asset.Tags), asset.Notes, mustJSON(asset.Specs),
		asset.Source, asset.ExternalID, asset.CreatedAt, asset.UpdatedAt,
	)
	if err != nil {
		return model.Asset{}, err
	}
	return asset, nil
}

func (s *DBStore) UpdateAsset(ctx context.Context, id string, asset model.Asset) (model.Asset, error) {
	asset = normalizeAsset(asset)
	if err := validateAsset(asset); err != nil {
		return model.Asset{}, err
	}

	existing, err := s.getAsset(ctx, id)
	if err != nil {
		return model.Asset{}, err
	}

	asset.ID = existing.ID
	asset.CreatedAt = existing.CreatedAt
	asset.UpdatedAt = time.Now().Format(time.RFC3339)

	result, err := s.db.ExecContext(ctx, `
		UPDATE assets
		SET platform_id = ?, platform_code = ?, platform_name = ?, cloud_account_id = ?, cloud_account_name = ?,
			account_id = ?, project_code = ?, category = ?, resource_type = ?, region = ?, environment = ?, name = ?, endpoint = ?,
			owner = ?, status = ?, criticality = ?, last_checked_at = ?, tags_csv = ?, notes = ?, specs_json = ?, source = ?,
			external_id = ?, updated_at = ?
		WHERE id = ?
	`,
		asset.PlatformID, asset.PlatformCode, asset.PlatformName, asset.CloudAccountID, asset.CloudAccountName,
		asset.AccountID, asset.ProjectCode, asset.Category, asset.ResourceType, asset.Region, asset.Environment, asset.Name, asset.Endpoint,
		asset.Owner, asset.Status, asset.Criticality, asset.LastCheckedAt, joinTags(asset.Tags), asset.Notes, mustJSON(asset.Specs),
		asset.Source, asset.ExternalID, asset.UpdatedAt, asset.ID,
	)
	if err != nil {
		return model.Asset{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return model.Asset{}, err
	}
	return asset, nil
}

func (s *DBStore) UpsertAssetBySource(ctx context.Context, asset model.Asset) (model.Asset, bool, error) {
	asset = normalizeAsset(asset)
	if err := validateAsset(asset); err != nil {
		return model.Asset{}, false, err
	}

	if strings.TrimSpace(asset.Source) == "" || strings.TrimSpace(asset.ExternalID) == "" {
		createdAsset, err := s.CreateAsset(ctx, asset)
		return createdAsset, true, err
	}

	var existingID string
	var createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, created_at
		FROM assets
		WHERE source = ? AND external_id = ?
		LIMIT 1
	`, asset.Source, asset.ExternalID).Scan(&existingID, &createdAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.Asset{}, false, err
	}

	if errors.Is(err, sql.ErrNoRows) {
		createdAsset, err := s.CreateAsset(ctx, asset)
		return createdAsset, true, err
	}

	asset.ID = existingID
	asset.CreatedAt = createdAt
	asset.UpdatedAt = time.Now().Format(time.RFC3339)

	result, err := s.db.ExecContext(ctx, `
		UPDATE assets
		SET platform_id = ?, platform_code = ?, platform_name = ?, cloud_account_id = ?, cloud_account_name = ?,
			account_id = ?, project_code = ?, category = ?, resource_type = ?, region = ?, environment = ?, name = ?, endpoint = ?,
			owner = ?, status = ?, criticality = ?, last_checked_at = ?, tags_csv = ?, notes = ?, specs_json = ?, updated_at = ?
		WHERE id = ?
	`,
		asset.PlatformID, asset.PlatformCode, asset.PlatformName, asset.CloudAccountID, asset.CloudAccountName,
		asset.AccountID, asset.ProjectCode, asset.Category, asset.ResourceType, asset.Region, asset.Environment, asset.Name, asset.Endpoint,
		asset.Owner, asset.Status, asset.Criticality, asset.LastCheckedAt, joinTags(asset.Tags), asset.Notes, mustJSON(asset.Specs),
		asset.UpdatedAt, asset.ID,
	)
	if err != nil {
		return model.Asset{}, false, err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return model.Asset{}, false, err
	} else if affected == 0 && s.dialect != dialectMySQL {
		return model.Asset{}, false, os.ErrNotExist
	}

	return asset, false, nil
}

func (s *DBStore) MarkAssetsStaleBySource(ctx context.Context, cloudAccountID string, source string, activeExternalIDs []string, syncedRegions []string, checkedAt string) (int, error) {
	cloudAccountID = strings.TrimSpace(cloudAccountID)
	source = strings.TrimSpace(source)
	checkedAt = strings.TrimSpace(checkedAt)
	if cloudAccountID == "" || source == "" {
		return 0, nil
	}

	regions := map[string]struct{}{}
	for _, region := range syncedRegions {
		region = strings.TrimSpace(region)
		if region != "" {
			regions[region] = struct{}{}
		}
	}
	if len(regions) == 0 {
		return 0, nil
	}

	active := map[string]struct{}{}
	for _, externalID := range activeExternalIDs {
		externalID = strings.TrimSpace(externalID)
		if externalID != "" {
			active[externalID] = struct{}{}
		}
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, external_id, region
		FROM assets
		WHERE cloud_account_id = ? AND source = ? AND external_id <> '' AND status <> 'stale'
	`, cloudAccountID, source)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type staleCandidate struct {
		id         string
		externalID string
		region     string
	}
	candidates := []staleCandidate{}
	for rows.Next() {
		var item staleCandidate
		if err := rows.Scan(&item.id, &item.externalID, &item.region); err != nil {
			return 0, err
		}
		if _, ok := regions[item.region]; !ok {
			continue
		}
		if _, ok := active[item.externalID]; !ok {
			candidates = append(candidates, item)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(candidates) == 0 {
		return 0, nil
	}

	now := time.Now().Format(time.RFC3339)
	count := 0
	for _, item := range candidates {
		result, err := s.db.ExecContext(ctx, `
			UPDATE assets
			SET status = 'stale', last_checked_at = ?, updated_at = ?
			WHERE id = ? AND status <> 'stale'
		`, checkedAt, now, item.id)
		if err != nil {
			return count, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return count, err
		}
		count += int(affected)
	}
	return count, nil
}

func (s *DBStore) DeleteAsset(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM assets WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return ensureRowsAffected(result)
}

func (s *DBStore) GetAsset(ctx context.Context, id string) (model.Asset, error) {
	return s.getAsset(ctx, id)
}

func (s *DBStore) CreateChange(ctx context.Context, change model.ChangeRecord) (model.ChangeRecord, error) {
	change = normalizeChange(change)
	if err := validateChange(change); err != nil {
		return model.ChangeRecord{}, err
	}
	if _, err := s.getAsset(ctx, change.AssetID); err != nil {
		return model.ChangeRecord{}, err
	}

	now := time.Now().Format(time.RFC3339)
	change.ID = newID("chg")
	change.CreatedAt = now
	change.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO changes (
			id, asset_id, title, category, executor, risk_level, window, status,
			summary, rollback_plan, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		change.ID, change.AssetID, change.Title, change.Category, change.Executor, change.RiskLevel,
		change.Window, change.Status, change.Summary, change.RollbackPlan, change.CreatedAt, change.UpdatedAt,
	)
	if err != nil {
		return model.ChangeRecord{}, err
	}
	return change, nil
}

func (s *DBStore) UpdateChange(ctx context.Context, id string, change model.ChangeRecord) (model.ChangeRecord, error) {
	change = normalizeChange(change)
	if err := validateChange(change); err != nil {
		return model.ChangeRecord{}, err
	}
	if _, err := s.getAsset(ctx, change.AssetID); err != nil {
		return model.ChangeRecord{}, err
	}

	existing, err := s.getChange(ctx, id)
	if err != nil {
		return model.ChangeRecord{}, err
	}

	change.ID = existing.ID
	change.CreatedAt = existing.CreatedAt
	change.UpdatedAt = time.Now().Format(time.RFC3339)

	result, err := s.db.ExecContext(ctx, `
		UPDATE changes
		SET asset_id = ?, title = ?, category = ?, executor = ?, risk_level = ?, window = ?,
			status = ?, summary = ?, rollback_plan = ?, updated_at = ?
		WHERE id = ?
	`,
		change.AssetID, change.Title, change.Category, change.Executor, change.RiskLevel, change.Window,
		change.Status, change.Summary, change.RollbackPlan, change.UpdatedAt, change.ID,
	)
	if err != nil {
		return model.ChangeRecord{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return model.ChangeRecord{}, err
	}
	return change, nil
}

func (s *DBStore) DeleteChange(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM changes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return ensureRowsAffected(result)
}

func (s *DBStore) listAssets(ctx context.Context) ([]model.Asset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, platform_id, platform_code, platform_name, cloud_account_id, cloud_account_name, account_id,
		       project_code, category, resource_type, region, environment, name, endpoint, owner, status, criticality,
		       last_checked_at, tags_csv, notes, specs_json, source, external_id, created_at, updated_at
		FROM assets
		ORDER BY updated_at DESC
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

func (s *DBStore) listChanges(ctx context.Context) ([]model.ChangeRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, asset_id, title, category, executor, risk_level, window, status,
		       summary, rollback_plan, created_at, updated_at
		FROM changes
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ChangeRecord
	for rows.Next() {
		item, err := scanChange(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) getAsset(ctx context.Context, id string) (model.Asset, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, platform_id, platform_code, platform_name, cloud_account_id, cloud_account_name, account_id,
		       project_code, category, resource_type, region, environment, name, endpoint, owner, status, criticality,
		       last_checked_at, tags_csv, notes, specs_json, source, external_id, created_at, updated_at
		FROM assets
		WHERE id = ?
	`, id)

	item, err := scanAsset(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Asset{}, os.ErrNotExist
		}
		return model.Asset{}, err
	}
	return item, nil
}

func (s *DBStore) getChange(ctx context.Context, id string) (model.ChangeRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, asset_id, title, category, executor, risk_level, window, status,
		       summary, rollback_plan, created_at, updated_at
		FROM changes
		WHERE id = ?
	`, id)

	item, err := scanChange(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ChangeRecord{}, os.ErrNotExist
		}
		return model.ChangeRecord{}, err
	}
	return item, nil
}
