package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func (s *DBStore) ListCloudAccounts(ctx context.Context) ([]model.CloudAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ca.id, ca.platform_id, p.code, p.name, ca.name, ca.account_id, ca.default_region, ca.environment,
		       ca.owner, ca.criticality, ca.access_key_id_masked, ca.secret_access_key_masked, ca.sync_enabled,
		       ca.sync_mode, ca.sync_cron, ca.last_sync_at, ca.last_sync_status, ca.last_sync_summary,
		       ca.cost_currency, ca.last_month_cost, ca.last_month_to_date_cost, ca.current_month_cost, ca.forecast_month_cost,
		       ca.month_over_month_delta, ca.last_cost_sync_at, ca.last_cost_sync_status, ca.last_cost_sync_summary,
		       ca.created_at, ca.updated_at
		FROM cloud_accounts ca
		JOIN platforms p ON p.id = ca.platform_id
		ORDER BY p.code ASC, ca.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.CloudAccount
	for rows.Next() {
		var item model.CloudAccount
		var syncEnabled int
		if err := rows.Scan(
			&item.ID, &item.PlatformID, &item.PlatformCode, &item.PlatformName, &item.Name, &item.AccountID,
			&item.DefaultRegion, &item.Environment, &item.Owner, &item.Criticality,
			&item.AccessKeyIDMasked, &item.SecretAccessKeyMasked, &syncEnabled, &item.SyncMode, &item.SyncCron,
			&item.LastSyncAt, &item.LastSyncStatus, &item.LastSyncSummary,
			&item.CostCurrency, &item.LastMonthCost, &item.LastMonthToDateCost, &item.CurrentMonthCost, &item.ForecastMonthCost,
			&item.MonthOverMonthDelta, &item.LastCostSyncAt, &item.LastCostSyncStatus, &item.LastCostSyncSummary,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.SyncEnabled = syncEnabled == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) GetCloudAccount(ctx context.Context, id string) (model.CloudAccount, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT ca.id, ca.platform_id, p.code, p.name, ca.name, ca.account_id, ca.default_region, ca.environment,
		       ca.owner, ca.criticality, ca.access_key_id_masked, ca.secret_access_key_masked, ca.sync_enabled,
		       ca.sync_mode, ca.sync_cron, ca.last_sync_at, ca.last_sync_status, ca.last_sync_summary,
		       ca.cost_currency, ca.last_month_cost, ca.last_month_to_date_cost, ca.current_month_cost, ca.forecast_month_cost,
		       ca.month_over_month_delta, ca.last_cost_sync_at, ca.last_cost_sync_status, ca.last_cost_sync_summary,
		       ca.created_at, ca.updated_at
		FROM cloud_accounts ca
		JOIN platforms p ON p.id = ca.platform_id
		WHERE ca.id = ?
	`, id)

	var item model.CloudAccount
	var syncEnabled int
	if err := row.Scan(
		&item.ID, &item.PlatformID, &item.PlatformCode, &item.PlatformName, &item.Name, &item.AccountID,
		&item.DefaultRegion, &item.Environment, &item.Owner, &item.Criticality,
		&item.AccessKeyIDMasked, &item.SecretAccessKeyMasked, &syncEnabled, &item.SyncMode, &item.SyncCron,
		&item.LastSyncAt, &item.LastSyncStatus, &item.LastSyncSummary,
		&item.CostCurrency, &item.LastMonthCost, &item.LastMonthToDateCost, &item.CurrentMonthCost, &item.ForecastMonthCost,
		&item.MonthOverMonthDelta, &item.LastCostSyncAt, &item.LastCostSyncStatus, &item.LastCostSyncSummary,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CloudAccount{}, os.ErrNotExist
		}
		return model.CloudAccount{}, err
	}
	item.SyncEnabled = syncEnabled == 1
	return item, nil
}

func (s *DBStore) GetCloudAccountSecrets(ctx context.Context, id string) (string, string, error) {
	accessKeyID, accessKeyErr := s.getCredentialPlainValue(ctx, "cloud_account", id, "access_key_id", "default")
	secretAccessKey, secretErr := s.getCredentialPlainValue(ctx, "cloud_account", id, "secret_access_key", "default")
	if accessKeyErr == nil && secretErr == nil {
		return accessKeyID, secretAccessKey, nil
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT access_key_id, secret_access_key
		FROM cloud_accounts
		WHERE id = ?
	`, id)

	var legacyAccessKeyID string
	var legacySecretAccessKey string
	if err := row.Scan(&legacyAccessKeyID, &legacySecretAccessKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", os.ErrNotExist
		}
		return "", "", err
	}
	if legacyAccessKeyID != "" || legacySecretAccessKey != "" {
		_ = s.upsertCloudAccountCredentials(ctx, id, legacyAccessKeyID, legacySecretAccessKey)
	}
	return legacyAccessKeyID, legacySecretAccessKey, nil
}

func (s *DBStore) CreateCloudAccount(ctx context.Context, req model.CloudAccountUpsertRequest) (model.CloudAccount, error) {
	req = normalizeCloudAccountRequest(req)
	if err := validateCloudAccountRequest(req); err != nil {
		return model.CloudAccount{}, err
	}

	platform, err := s.getPlatformByCode(ctx, req.PlatformCode)
	if err != nil {
		return model.CloudAccount{}, err
	}

	now := time.Now().Format(time.RFC3339)
	id := newID("acct")

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cloud_accounts (
			id, platform_id, name, account_id, default_region, environment, owner, criticality,
			access_key_id, secret_access_key, access_key_id_masked, secret_access_key_masked,
			sync_enabled, sync_mode, sync_cron, last_sync_at, last_sync_status, last_sync_summary,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', '', ?, ?)
	`,
		id, platform.ID, req.Name, req.AccountID, req.DefaultRegion, req.Environment, req.Owner, req.Criticality,
		"", "", maskAccessKey(req.AccessKeyID), maskSecretKey(req.SecretAccessKey),
		boolToInt(req.SyncEnabled), req.SyncMode, req.SyncCron, now, now,
	)
	if err != nil {
		return model.CloudAccount{}, err
	}
	if err := s.upsertCloudAccountCredentials(ctx, id, req.AccessKeyID, req.SecretAccessKey); err != nil {
		return model.CloudAccount{}, err
	}

	return s.GetCloudAccount(ctx, id)
}

func (s *DBStore) UpdateCloudAccount(ctx context.Context, id string, req model.CloudAccountUpsertRequest) (model.CloudAccount, error) {
	req = normalizeCloudAccountRequest(req)
	if err := validateCloudAccountRequest(req); err != nil {
		return model.CloudAccount{}, err
	}

	existingRow := s.db.QueryRowContext(ctx, `
		SELECT platform_id, access_key_id, secret_access_key, created_at
		FROM cloud_accounts
		WHERE id = ?
	`, id)

	var oldPlatformID string
	var oldAccessKey string
	var oldSecret string
	var createdAt string
	if err := existingRow.Scan(&oldPlatformID, &oldAccessKey, &oldSecret, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CloudAccount{}, os.ErrNotExist
		}
		return model.CloudAccount{}, err
	}

	platform, err := s.getPlatformByCode(ctx, req.PlatformCode)
	if err != nil {
		return model.CloudAccount{}, err
	}

	accessKey := req.AccessKeyID
	if accessKey == "" {
		accessKey, _ = s.getCredentialPlainValue(ctx, "cloud_account", id, "access_key_id", "default")
		if accessKey == "" {
			accessKey = oldAccessKey
		}
	}
	secretKey := req.SecretAccessKey
	if secretKey == "" {
		secretKey, _ = s.getCredentialPlainValue(ctx, "cloud_account", id, "secret_access_key", "default")
		if secretKey == "" {
			secretKey = oldSecret
		}
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE cloud_accounts
		SET platform_id = ?, name = ?, account_id = ?, default_region = ?, environment = ?, owner = ?, criticality = ?,
			access_key_id = ?, secret_access_key = ?, access_key_id_masked = ?, secret_access_key_masked = ?,
			sync_enabled = ?, sync_mode = ?, sync_cron = ?, updated_at = ?
		WHERE id = ?
	`,
		platform.ID, req.Name, req.AccountID, req.DefaultRegion, req.Environment, req.Owner, req.Criticality,
		"", "", maskAccessKey(accessKey), maskSecretKey(secretKey),
		boolToInt(req.SyncEnabled), req.SyncMode, req.SyncCron, time.Now().Format(time.RFC3339), id,
	)
	if err != nil {
		return model.CloudAccount{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return model.CloudAccount{}, err
	}
	if err := s.upsertCloudAccountCredentials(ctx, id, accessKey, secretKey); err != nil {
		return model.CloudAccount{}, err
	}

	_ = createdAt
	_ = oldPlatformID
	return s.GetCloudAccount(ctx, id)
}

func (s *DBStore) migrateCloudAccountCredentials(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, access_key_id, secret_access_key
		FROM cloud_accounts
		WHERE access_key_id <> '' OR secret_access_key <> ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type accountSecret struct {
		id              string
		accessKeyID     string
		secretAccessKey string
	}
	items := []accountSecret{}
	for rows.Next() {
		var item accountSecret
		if err := rows.Scan(&item.id, &item.accessKeyID, &item.secretAccessKey); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range items {
		if err := s.upsertCloudAccountCredentials(ctx, item.id, item.accessKeyID, item.secretAccessKey); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE cloud_accounts
			SET access_key_id = '', secret_access_key = '', updated_at = ?
			WHERE id = ?
		`, time.Now().Format(time.RFC3339), item.id); err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) SetCloudAccountSyncResult(ctx context.Context, accountID string, result model.CloudAccountSyncResult) error {
	summary := fmt.Sprintf("发现 %d 条，新增 %d 条，更新 %d 条，标记 stale %d 条", result.DiscoveredAssets, result.CreatedAssets, result.UpdatedAssets, result.StaleAssets)
	updateResult, err := s.db.ExecContext(ctx, `
		UPDATE cloud_accounts
		SET last_sync_at = ?, last_sync_status = ?, last_sync_summary = ?, updated_at = ?
		WHERE id = ?
	`,
		result.FinishedAt, "success", summary, time.Now().Format(time.RFC3339), accountID,
	)
	if err != nil {
		return err
	}
	return ensureRowsAffected(updateResult)
}

func (s *DBStore) SetCloudAccountCostResult(ctx context.Context, accountID string, result model.CloudAccountCostResult) error {
	updateResult, err := s.db.ExecContext(ctx, `
		UPDATE cloud_accounts
		SET cost_currency = ?, last_month_cost = ?, last_month_to_date_cost = ?, current_month_cost = ?, forecast_month_cost = ?,
			month_over_month_delta = ?, last_cost_sync_at = ?, last_cost_sync_status = ?, last_cost_sync_summary = ?,
			updated_at = ?
		WHERE id = ?
	`,
		result.Currency, result.LastMonthCost, result.LastMonthToDateCost, result.CurrentMonthCost, result.ForecastMonthCost,
		result.MonthOverMonthDelta, result.FinishedAt, "success", result.Summary, time.Now().Format(time.RFC3339), accountID,
	)
	if err != nil {
		return err
	}
	return ensureRowsAffected(updateResult)
}

func (s *DBStore) RecordCloudAccountSync(ctx context.Context, record model.CloudAccountSyncRecord) error {
	breakdownJSON, err := json.Marshal(record.Breakdown)
	if err != nil {
		return err
	}
	warningsJSON, err := json.Marshal(record.Warnings)
	if err != nil {
		return err
	}

	if record.ID == "" {
		record.ID = newID("sync")
	}
	recordSummary := strings.TrimSpace(record.Summary)
	if recordSummary == "" {
		recordSummary = fmt.Sprintf("发现 %d 条，新增 %d 条，更新 %d 条，标记 stale %d 条", record.DiscoveredAssets, record.CreatedAssets, record.UpdatedAssets, record.StaleAssets)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cloud_account_syncs (
			id, cloud_account_id, started_at, finished_at, status, discovered_assets, created_assets,
			updated_assets, warnings_json, breakdown_json, summary
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID, record.CloudAccountID, record.StartedAt, record.FinishedAt, record.Status,
		record.DiscoveredAssets, record.CreatedAssets, record.UpdatedAssets,
		string(warningsJSON), string(breakdownJSON), recordSummary,
	)
	return err
}

func (s *DBStore) listRecentSyncs(ctx context.Context, limit int) ([]model.CloudAccountSyncRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, cloud_account_id, started_at, finished_at, status, discovered_assets, created_assets,
		       updated_assets, warnings_json, breakdown_json, summary
		FROM cloud_account_syncs
		ORDER BY started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.CloudAccountSyncRecord
	for rows.Next() {
		item, err := scanSyncRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
