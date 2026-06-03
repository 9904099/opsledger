package store

import (
	"context"
)

func (s *DBStore) initialize(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, s.schemaSQL()); err != nil {
		return err
	}
	if err := s.migrateLegacySchema(ctx); err != nil {
		return err
	}
	if err := s.seedPlatforms(ctx); err != nil {
		return err
	}
	if err := s.seedEnvironments(ctx); err != nil {
		return err
	}
	if err := s.seedUsersAndPermissions(ctx, devSeedUsersEnabled()); err != nil {
		return err
	}
	if err := s.seedApprovalFlows(ctx); err != nil {
		return err
	}
	if envBool("OPSLEDGER_SEED_EXAMPLE_TOOLS", false) {
		if err := s.seedDefaultTools(ctx); err != nil {
			return err
		}
	}
	if err := s.backfillLegacyAssetRelations(ctx); err != nil {
		return err
	}
	return nil
}

func (s *DBStore) withTx(ctx context.Context, fn func(*dialectTx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
