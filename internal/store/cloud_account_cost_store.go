package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func (s *DBStore) UpsertCloudAccountCostRecords(ctx context.Context, records []model.CloudAccountCostRecord) error {
	if len(records) == 0 {
		return nil
	}
	now := time.Now().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, record := range records {
		record.CloudAccountID = strings.TrimSpace(record.CloudAccountID)
		record.PeriodStart = strings.TrimSpace(record.PeriodStart)
		record.PeriodEnd = strings.TrimSpace(record.PeriodEnd)
		record.Granularity = strings.ToLower(strings.TrimSpace(record.Granularity))
		record.DimensionType = strings.ToLower(strings.TrimSpace(record.DimensionType))
		record.DimensionName = strings.TrimSpace(record.DimensionName)
		if record.CloudAccountID == "" || record.PeriodStart == "" || record.PeriodEnd == "" || record.Granularity == "" || record.DimensionType == "" || record.DimensionName == "" {
			continue
		}
		if record.ID == "" {
			record.ID = cloudAccountCostRecordID(record)
		}
		if strings.TrimSpace(record.Source) == "" {
			record.Source = "aws_cost_explorer"
		}
		if strings.TrimSpace(record.SyncedAt) == "" {
			record.SyncedAt = now
		}
		if strings.TrimSpace(record.CreatedAt) == "" {
			record.CreatedAt = now
		}
		record.UpdatedAt = now
		if strings.TrimSpace(record.Amount) == "" {
			record.Amount = "0.00"
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cloud_account_cost_records (
				id, cloud_account_id, period_start, period_end, granularity, dimension_type,
				dimension_name, currency, amount, source, synced_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				currency = excluded.currency,
				amount = excluded.amount,
				source = excluded.source,
				synced_at = excluded.synced_at,
				updated_at = excluded.updated_at
		`, record.ID, record.CloudAccountID, record.PeriodStart, record.PeriodEnd, record.Granularity, record.DimensionType,
			record.DimensionName, record.Currency, record.Amount, record.Source, record.SyncedAt, record.CreatedAt, record.UpdatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *DBStore) ListCloudAccountCostRecords(ctx context.Context, limit int) ([]model.CloudAccountCostRecord, error) {
	if limit <= 0 {
		limit = 2000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, cloud_account_id, period_start, period_end, granularity, dimension_type,
		       dimension_name, currency, amount, source, synced_at, created_at, updated_at
		FROM cloud_account_cost_records
		ORDER BY period_start DESC, dimension_type ASC, dimension_name ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.CloudAccountCostRecord
	for rows.Next() {
		var item model.CloudAccountCostRecord
		if err := rows.Scan(&item.ID, &item.CloudAccountID, &item.PeriodStart, &item.PeriodEnd, &item.Granularity, &item.DimensionType,
			&item.DimensionName, &item.Currency, &item.Amount, &item.Source, &item.SyncedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func cloudAccountCostRecordID(record model.CloudAccountCostRecord) string {
	material := strings.Join([]string{
		record.CloudAccountID,
		record.PeriodStart,
		record.PeriodEnd,
		strings.ToLower(record.Granularity),
		strings.ToLower(record.DimensionType),
		strings.ToLower(record.DimensionName),
	}, "|")
	sum := sha256.Sum256([]byte(material))
	return "cost-" + hex.EncodeToString(sum[:])[:24]
}
