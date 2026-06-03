package store

import (
	"context"
	"time"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS platforms (
	id TEXT PRIMARY KEY,
	code TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_accounts (
	id TEXT PRIMARY KEY,
	platform_id TEXT NOT NULL,
	name TEXT NOT NULL,
	account_id TEXT NOT NULL DEFAULT '',
	default_region TEXT NOT NULL DEFAULT '',
	environment TEXT NOT NULL DEFAULT 'prod',
	owner TEXT NOT NULL DEFAULT '',
	criticality TEXT NOT NULL DEFAULT 'medium',
	access_key_id TEXT NOT NULL DEFAULT '',
	secret_access_key TEXT NOT NULL DEFAULT '',
	access_key_id_masked TEXT NOT NULL DEFAULT '',
	secret_access_key_masked TEXT NOT NULL DEFAULT '',
	sync_enabled INTEGER NOT NULL DEFAULT 0,
	sync_mode TEXT NOT NULL DEFAULT 'manual',
	sync_cron TEXT NOT NULL DEFAULT '',
	last_sync_at TEXT NOT NULL DEFAULT '',
	last_sync_status TEXT NOT NULL DEFAULT '',
	last_sync_summary TEXT NOT NULL DEFAULT '',
	cost_currency TEXT NOT NULL DEFAULT '',
	last_month_cost TEXT NOT NULL DEFAULT '',
	last_month_to_date_cost TEXT NOT NULL DEFAULT '',
	current_month_cost TEXT NOT NULL DEFAULT '',
	forecast_month_cost TEXT NOT NULL DEFAULT '',
	month_over_month_delta TEXT NOT NULL DEFAULT '',
	last_cost_sync_at TEXT NOT NULL DEFAULT '',
	last_cost_sync_status TEXT NOT NULL DEFAULT '',
	last_cost_sync_summary TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (platform_id) REFERENCES platforms(id)
);

CREATE TABLE IF NOT EXISTS cloud_account_syncs (
	id TEXT PRIMARY KEY,
	cloud_account_id TEXT NOT NULL,
	started_at TEXT NOT NULL,
	finished_at TEXT NOT NULL,
	status TEXT NOT NULL,
	discovered_assets INTEGER NOT NULL DEFAULT 0,
	created_assets INTEGER NOT NULL DEFAULT 0,
	updated_assets INTEGER NOT NULL DEFAULT 0,
	warnings_json TEXT NOT NULL DEFAULT '[]',
	breakdown_json TEXT NOT NULL DEFAULT '{}',
	summary TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (cloud_account_id) REFERENCES cloud_accounts(id)
);

CREATE TABLE IF NOT EXISTS cloud_account_cost_records (
	id TEXT PRIMARY KEY,
	cloud_account_id TEXT NOT NULL,
	period_start TEXT NOT NULL,
	period_end TEXT NOT NULL,
	granularity TEXT NOT NULL,
	dimension_type TEXT NOT NULL,
	dimension_name TEXT NOT NULL,
	currency TEXT NOT NULL DEFAULT '',
	amount TEXT NOT NULL DEFAULT '0.00',
	source TEXT NOT NULL DEFAULT 'aws_cost_explorer',
	synced_at TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (cloud_account_id) REFERENCES cloud_accounts(id)
);

CREATE TABLE IF NOT EXISTS assets (
	id TEXT PRIMARY KEY,
	platform_id TEXT NOT NULL DEFAULT '',
	platform_code TEXT NOT NULL DEFAULT '',
	platform_name TEXT NOT NULL DEFAULT '',
	cloud_account_id TEXT NOT NULL DEFAULT '',
	cloud_account_name TEXT NOT NULL DEFAULT '',
	account_id TEXT NOT NULL DEFAULT '',
	project_code TEXT NOT NULL DEFAULT 'public',
	category TEXT NOT NULL DEFAULT 'other',
	resource_type TEXT NOT NULL DEFAULT '',
	region TEXT NOT NULL DEFAULT '',
	environment TEXT NOT NULL DEFAULT 'prod',
	name TEXT NOT NULL DEFAULT '',
	endpoint TEXT NOT NULL DEFAULT '',
	owner TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	criticality TEXT NOT NULL DEFAULT 'medium',
	last_checked_at TEXT NOT NULL DEFAULT '',
	tags_csv TEXT NOT NULL DEFAULT '',
	notes TEXT NOT NULL DEFAULT '',
	specs_json TEXT NOT NULL DEFAULT '{}',
	source TEXT NOT NULL DEFAULT '',
	external_id TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS changes (
	id TEXT PRIMARY KEY,
	asset_id TEXT NOT NULL,
	title TEXT NOT NULL,
	category TEXT NOT NULL,
	executor TEXT NOT NULL,
	risk_level TEXT NOT NULL,
	window TEXT NOT NULL,
	status TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	rollback_plan TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS inspection_records (
	id TEXT PRIMARY KEY,
	asset_id TEXT NOT NULL,
	executor TEXT NOT NULL,
	result TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	checked_at TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS inspection_attachments (
	id TEXT PRIMARY KEY,
	inspection_id TEXT NOT NULL,
	asset_id TEXT NOT NULL,
	file_name TEXT NOT NULL,
	content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
	size_bytes INTEGER NOT NULL DEFAULT 0,
	uploader TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	data BLOB NOT NULL,
	created_at TEXT NOT NULL,
	FOREIGN KEY (inspection_id) REFERENCES inspection_records(id) ON DELETE CASCADE,
	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS probe_records (
	id TEXT PRIMARY KEY,
	asset_id TEXT NOT NULL,
	url TEXT NOT NULL,
	method TEXT NOT NULL DEFAULT 'GET',
	status TEXT NOT NULL,
	status_code INTEGER NOT NULL DEFAULT 0,
	latency_ms INTEGER NOT NULL DEFAULT 0,
	error TEXT NOT NULL DEFAULT '',
	checked_at TEXT NOT NULL,
	tls_expires_at TEXT NOT NULL DEFAULT '',
	cert_days_remaining INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS alert_records (
	id TEXT PRIMARY KEY,
	asset_id TEXT NOT NULL,
	source TEXT NOT NULL,
	severity TEXT NOT NULL,
	status TEXT NOT NULL,
	title TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	first_seen_at TEXT NOT NULL,
	last_seen_at TEXT NOT NULL,
	resolved_at TEXT NOT NULL DEFAULT '',
	resolved_by TEXT NOT NULL DEFAULT '',
	resolution TEXT NOT NULL DEFAULT '',
	event_count INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS environments (
	id TEXT PRIMARY KEY,
	code TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	tier TEXT NOT NULL DEFAULT '',
	owner TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tool_assets (
	id TEXT PRIMARY KEY,
	asset_id TEXT NOT NULL UNIQUE,
	environment TEXT NOT NULL,
	tool_type TEXT NOT NULL,
	login_policy TEXT NOT NULL DEFAULT 'sso',
	credential_policy TEXT NOT NULL DEFAULT 'none',
	approval_required INTEGER NOT NULL DEFAULT 0,
	webssh_enabled INTEGER NOT NULL DEFAULT 0,
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS app_users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	display_name TEXT NOT NULL,
	email TEXT NOT NULL DEFAULT '',
	phone TEXT NOT NULL DEFAULT '',
	role TEXT NOT NULL,
	team TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active',
	auth_source TEXT NOT NULL DEFAULT 'local',
	external_subject TEXT NOT NULL DEFAULT '',
	password_hash TEXT NOT NULL DEFAULT '',
	password_changed_at TEXT NOT NULL DEFAULT '',
	failed_login_count INTEGER NOT NULL DEFAULT 0,
	locked_until TEXT NOT NULL DEFAULT '',
	last_login_at TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	token_hash TEXT NOT NULL UNIQUE,
	expires_at TEXT NOT NULL,
	last_seen_at TEXT NOT NULL,
	revoked_at TEXT NOT NULL DEFAULT '',
	ip TEXT NOT NULL DEFAULT '',
	user_agent TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (user_id) REFERENCES app_users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS roles (
	id TEXT PRIMARY KEY,
	code TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	level INTEGER NOT NULL DEFAULT 100,
	status TEXT NOT NULL DEFAULT 'active',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
	id TEXT PRIMARY KEY,
	role TEXT NOT NULL,
	scope TEXT NOT NULL,
	action TEXT NOT NULL,
	environment TEXT NOT NULL DEFAULT '*',
	project_code TEXT NOT NULL DEFAULT '*',
	requires_approval INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS approval_flows (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	scope TEXT NOT NULL,
	environment TEXT NOT NULL DEFAULT '*',
	status TEXT NOT NULL DEFAULT 'active',
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS approval_flow_steps (
	id TEXT PRIMARY KEY,
	flow_id TEXT NOT NULL,
	step_order INTEGER NOT NULL,
	approver_role TEXT NOT NULL,
	approver_label TEXT NOT NULL DEFAULT '',
	required_action TEXT NOT NULL DEFAULT 'approved',
	timeout_minutes INTEGER NOT NULL DEFAULT 60,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (flow_id) REFERENCES approval_flows(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS approval_requests (
	id TEXT PRIMARY KEY,
	flow_id TEXT NOT NULL DEFAULT '',
	current_step_id TEXT NOT NULL DEFAULT '',
	requester TEXT NOT NULL,
	request_type TEXT NOT NULL,
	target_type TEXT NOT NULL,
	target_id TEXT NOT NULL DEFAULT '',
	environment TEXT NOT NULL,
	reason TEXT NOT NULL,
	permission_level TEXT NOT NULL DEFAULT '',
	duration_minutes INTEGER NOT NULL DEFAULT 30,
	status TEXT NOT NULL DEFAULT 'pending',
	approver TEXT NOT NULL DEFAULT '',
	decision_summary TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	decided_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS approval_tasks (
	id TEXT PRIMARY KEY,
	approval_id TEXT NOT NULL,
	flow_id TEXT NOT NULL DEFAULT '',
	step_id TEXT NOT NULL DEFAULT '',
	step_order INTEGER NOT NULL,
	approver_role TEXT NOT NULL,
	approver_label TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	approver TEXT NOT NULL DEFAULT '',
	decision_summary TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	decided_at TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (approval_id) REFERENCES approval_requests(id) ON DELETE CASCADE
);

	CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY,
		owner_type TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		kind TEXT NOT NULL,
		key_name TEXT NOT NULL DEFAULT '',
		encrypted_value TEXT NOT NULL DEFAULT '',
		masked_value TEXT NOT NULL DEFAULT '',
		environment TEXT NOT NULL DEFAULT '',
		project_code TEXT NOT NULL DEFAULT '',
		access_policy TEXT NOT NULL DEFAULT 'ops_only',
		status TEXT NOT NULL DEFAULT 'active',
		last_viewed_at TEXT NOT NULL DEFAULT '',
		last_viewed_by TEXT NOT NULL DEFAULT '',
		last_rotated_at TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(owner_type, owner_id, kind, key_name)
	);

	CREATE TABLE IF NOT EXISTS access_grants (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		action TEXT NOT NULL,
		target_type TEXT NOT NULL,
		target_id TEXT NOT NULL,
		environment TEXT NOT NULL DEFAULT '',
		source_approval_id TEXT NOT NULL DEFAULT '',
		temporary_credential TEXT NOT NULL DEFAULT '',
		temporary_credential_hash TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'active',
		expires_at TEXT NOT NULL,
		revoked_at TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS webssh_sessions (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		asset_id TEXT NOT NULL,
		access_grant_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		login_url TEXT NOT NULL DEFAULT '',
		ip TEXT NOT NULL DEFAULT '',
		user_agent TEXT NOT NULL DEFAULT '',
		close_reason TEXT NOT NULL DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		started_at TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		ended_at TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS audit_events (
		id TEXT PRIMARY KEY,
		actor TEXT NOT NULL DEFAULT '',
		actor_role TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		target_type TEXT NOT NULL DEFAULT '',
		target_id TEXT NOT NULL DEFAULT '',
		target_name TEXT NOT NULL DEFAULT '',
		outcome TEXT NOT NULL DEFAULT '',
		ip TEXT NOT NULL DEFAULT '',
		user_agent TEXT NOT NULL DEFAULT '',
		summary TEXT NOT NULL DEFAULT '',
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL
	);

CREATE INDEX IF NOT EXISTS idx_platforms_code ON platforms(code);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_platform_id ON cloud_accounts(platform_id);
CREATE INDEX IF NOT EXISTS idx_cloud_account_cost_records_account ON cloud_account_cost_records(cloud_account_id, period_start DESC);
CREATE INDEX IF NOT EXISTS idx_cloud_account_cost_records_dimension ON cloud_account_cost_records(dimension_type, dimension_name, period_start DESC);
CREATE INDEX IF NOT EXISTS idx_assets_updated_at ON assets(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_assets_source_external_id ON assets(source, external_id);
CREATE INDEX IF NOT EXISTS idx_changes_updated_at ON changes(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_changes_asset_id ON changes(asset_id);
CREATE INDEX IF NOT EXISTS idx_inspection_records_asset_id ON inspection_records(asset_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_inspection_records_checked_at ON inspection_records(checked_at DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inspection_attachments_inspection ON inspection_attachments(inspection_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inspection_attachments_asset ON inspection_attachments(asset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_probe_records_asset_id ON probe_records(asset_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_probe_records_checked_at ON probe_records(checked_at DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_records_asset_status ON alert_records(asset_id, status, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_records_status ON alert_records(status, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_tool_assets_environment ON tool_assets(environment, tool_type);
CREATE INDEX IF NOT EXISTS idx_app_users_role ON app_users(role);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token_hash ON user_sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_roles_code ON roles(code);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role, environment);
CREATE INDEX IF NOT EXISTS idx_approval_flows_scope ON approval_flows(scope, environment);
CREATE INDEX IF NOT EXISTS idx_approval_flow_steps_flow ON approval_flow_steps(flow_id, step_order);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_approval_tasks_approval ON approval_tasks(approval_id, step_order);
CREATE INDEX IF NOT EXISTS idx_approval_tasks_status_role ON approval_tasks(status, approver_role);
CREATE INDEX IF NOT EXISTS idx_credentials_owner ON credentials(owner_type, owner_id);
CREATE INDEX IF NOT EXISTS idx_credentials_status ON credentials(status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_access_grants_lookup ON access_grants(username, action, target_type, target_id, status, expires_at);
CREATE INDEX IF NOT EXISTS idx_webssh_sessions_user ON webssh_sessions(username, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events(actor, created_at DESC);
`

func (s *DBStore) migrateLegacySchema(ctx context.Context) error {
	if err := s.ensureAssetColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureCloudAccountColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureCloudAccountCostRecordSchema(ctx); err != nil {
		return err
	}
	if err := s.ensureUserColumns(ctx); err != nil {
		return err
	}
	if err := s.ensurePermissionSchema(ctx); err != nil {
		return err
	}
	if err := s.ensureApprovalColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureAccessGrantSchema(ctx); err != nil {
		return err
	}
	if err := s.ensureCredentialSchema(ctx); err != nil {
		return err
	}
	if err := s.migrateCloudAccountCredentials(ctx); err != nil {
		return err
	}
	if err := s.ensureAuditSchema(ctx); err != nil {
		return err
	}
	if err := s.ensureAlertSchema(ctx); err != nil {
		return err
	}
	if err := s.ensureInspectionAttachmentSchema(ctx); err != nil {
		return err
	}
	return nil
}

func (s *DBStore) ensureAssetColumns(ctx context.Context) error {
	columns, err := s.columnNames(ctx, "assets")
	if err != nil {
		return err
	}

	addIfMissing := func(name, ddl string) error {
		if columns[name] {
			return nil
		}
		_, err := s.db.ExecContext(ctx, ddl)
		return err
	}

	if err := addIfMissing("platform_id", `ALTER TABLE assets ADD COLUMN platform_id TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("platform_code", `ALTER TABLE assets ADD COLUMN platform_code TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("platform_name", `ALTER TABLE assets ADD COLUMN platform_name TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("cloud_account_id", `ALTER TABLE assets ADD COLUMN cloud_account_id TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("cloud_account_name", `ALTER TABLE assets ADD COLUMN cloud_account_name TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("account_id", `ALTER TABLE assets ADD COLUMN account_id TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("project_code", `ALTER TABLE assets ADD COLUMN project_code TEXT NOT NULL DEFAULT 'public'`); err != nil {
		return err
	}
	if err := addIfMissing("category", `ALTER TABLE assets ADD COLUMN category TEXT NOT NULL DEFAULT 'other'`); err != nil {
		return err
	}
	if err := addIfMissing("resource_type", `ALTER TABLE assets ADD COLUMN resource_type TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("source", `ALTER TABLE assets ADD COLUMN source TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("external_id", `ALTER TABLE assets ADD COLUMN external_id TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("specs_json", `ALTER TABLE assets ADD COLUMN specs_json TEXT NOT NULL DEFAULT '{}'`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_assets_cloud_account_id ON assets(cloud_account_id)`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_assets_project_code ON assets(project_code)`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_inspection_records_checked_at ON inspection_records(checked_at DESC, created_at DESC)`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_probe_records_checked_at ON probe_records(checked_at DESC, created_at DESC)`); err != nil {
		return err
	}
	if err := s.backfillAssetProjectCodes(ctx); err != nil {
		return err
	}

	if columns["account"] {
		if _, err := s.db.ExecContext(ctx, `UPDATE assets SET account_id = account WHERE account_id = ''`); err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) backfillAssetProjectCodes(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, platform_id, platform_code, platform_name, cloud_account_id, cloud_account_name, account_id,
		       project_code, category, resource_type, region, environment, name, endpoint, owner, status, criticality,
		       last_checked_at, tags_csv, notes, specs_json, source, external_id, created_at, updated_at
		FROM assets
		WHERE project_code = '' OR project_code = 'public' OR project_code IS NULL
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type projectUpdate struct {
		id      string
		project string
	}
	updates := []projectUpdate{}
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return err
		}
		project := inferProjectCode(asset)
		if project == "" {
			project = "public"
		}
		if asset.ProjectCode != project {
			updates = append(updates, projectUpdate{id: asset.ID, project: project})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	now := time.Now().Format(time.RFC3339)
	return s.withTx(ctx, func(tx *dialectTx) error {
		for _, update := range updates {
			if _, err := tx.ExecContext(ctx, `UPDATE assets SET project_code = ?, updated_at = ? WHERE id = ?`, update.project, now, update.id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *DBStore) ensureUserColumns(ctx context.Context) error {
	columns, err := s.columnNames(ctx, "app_users")
	if err != nil {
		return err
	}

	addIfMissing := func(name, ddl string) error {
		if columns[name] {
			return nil
		}
		_, err := s.db.ExecContext(ctx, ddl)
		return err
	}

	for name, ddl := range map[string]string{
		"email":               `ALTER TABLE app_users ADD COLUMN email TEXT NOT NULL DEFAULT ''`,
		"phone":               `ALTER TABLE app_users ADD COLUMN phone TEXT NOT NULL DEFAULT ''`,
		"auth_source":         `ALTER TABLE app_users ADD COLUMN auth_source TEXT NOT NULL DEFAULT 'local'`,
		"external_subject":    `ALTER TABLE app_users ADD COLUMN external_subject TEXT NOT NULL DEFAULT ''`,
		"password_hash":       `ALTER TABLE app_users ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''`,
		"password_changed_at": `ALTER TABLE app_users ADD COLUMN password_changed_at TEXT NOT NULL DEFAULT ''`,
		"failed_login_count":  `ALTER TABLE app_users ADD COLUMN failed_login_count INTEGER NOT NULL DEFAULT 0`,
		"locked_until":        `ALTER TABLE app_users ADD COLUMN locked_until TEXT NOT NULL DEFAULT ''`,
		"last_login_at":       `ALTER TABLE app_users ADD COLUMN last_login_at TEXT NOT NULL DEFAULT ''`,
	} {
		if err := addIfMissing(name, ddl); err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) ensurePermissionSchema(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS role_permissions (
			id TEXT PRIMARY KEY,
			role TEXT NOT NULL,
			scope TEXT NOT NULL,
			action TEXT NOT NULL,
			environment TEXT NOT NULL DEFAULT '*',
			project_code TEXT NOT NULL DEFAULT '*',
			requires_approval INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role, environment);
	`); err != nil {
		return err
	}
	return s.ensureColumns(ctx, "role_permissions", map[string]string{
		"project_code": `ALTER TABLE role_permissions ADD COLUMN project_code TEXT NOT NULL DEFAULT '*'`,
	})
}

func (s *DBStore) ensureApprovalColumns(ctx context.Context) error {
	columns, err := s.columnNames(ctx, "approval_requests")
	if err != nil {
		return err
	}

	addIfMissing := func(name, ddl string) error {
		if columns[name] {
			return nil
		}
		_, err := s.db.ExecContext(ctx, ddl)
		return err
	}
	if err := addIfMissing("flow_id", `ALTER TABLE approval_requests ADD COLUMN flow_id TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addIfMissing("current_step_id", `ALTER TABLE approval_requests ADD COLUMN current_step_id TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS approval_tasks (
			id TEXT PRIMARY KEY,
			approval_id TEXT NOT NULL,
			flow_id TEXT NOT NULL DEFAULT '',
			step_id TEXT NOT NULL DEFAULT '',
			step_order INTEGER NOT NULL,
			approver_role TEXT NOT NULL,
			approver_label TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			approver TEXT NOT NULL DEFAULT '',
			decision_summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			decided_at TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (approval_id) REFERENCES approval_requests(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_approval_tasks_approval ON approval_tasks(approval_id, step_order);
		CREATE INDEX IF NOT EXISTS idx_approval_tasks_status_role ON approval_tasks(status, approver_role);
	`)
	return err
}

func (s *DBStore) ensureAccessGrantSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS access_grants (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			action TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			environment TEXT NOT NULL DEFAULT '',
			source_approval_id TEXT NOT NULL DEFAULT '',
			temporary_credential TEXT NOT NULL DEFAULT '',
			temporary_credential_hash TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			expires_at TEXT NOT NULL,
			revoked_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS webssh_sessions (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			asset_id TEXT NOT NULL,
			access_grant_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			login_url TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			close_reason TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			ended_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_access_grants_lookup ON access_grants(username, action, target_type, target_id, status, expires_at);
		CREATE INDEX IF NOT EXISTS idx_webssh_sessions_user ON webssh_sessions(username, started_at DESC);
	`)
	if err != nil {
		return err
	}
	if err := s.ensureColumns(ctx, "access_grants", map[string]string{
		"temporary_credential_hash": `ALTER TABLE access_grants ADD COLUMN temporary_credential_hash TEXT NOT NULL DEFAULT ''`,
	}); err != nil {
		return err
	}
	return s.ensureColumns(ctx, "webssh_sessions", map[string]string{
		"ip":            `ALTER TABLE webssh_sessions ADD COLUMN ip TEXT NOT NULL DEFAULT ''`,
		"user_agent":    `ALTER TABLE webssh_sessions ADD COLUMN user_agent TEXT NOT NULL DEFAULT ''`,
		"close_reason":  `ALTER TABLE webssh_sessions ADD COLUMN close_reason TEXT NOT NULL DEFAULT ''`,
		"error_message": `ALTER TABLE webssh_sessions ADD COLUMN error_message TEXT NOT NULL DEFAULT ''`,
	})
}

func (s *DBStore) ensureCredentialSchema(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS credentials (
			id TEXT PRIMARY KEY,
			owner_type TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			key_name TEXT NOT NULL DEFAULT '',
			encrypted_value TEXT NOT NULL DEFAULT '',
			masked_value TEXT NOT NULL DEFAULT '',
			environment TEXT NOT NULL DEFAULT '',
			project_code TEXT NOT NULL DEFAULT '',
			access_policy TEXT NOT NULL DEFAULT 'ops_only',
			status TEXT NOT NULL DEFAULT 'active',
			last_viewed_at TEXT NOT NULL DEFAULT '',
			last_viewed_by TEXT NOT NULL DEFAULT '',
			last_rotated_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(owner_type, owner_id, kind, key_name)
		);
		CREATE INDEX IF NOT EXISTS idx_credentials_owner ON credentials(owner_type, owner_id);
		CREATE INDEX IF NOT EXISTS idx_credentials_status ON credentials(status, updated_at DESC);
	`); err != nil {
		return err
	}
	return s.ensureColumns(ctx, "credentials", map[string]string{
		"access_policy":   `ALTER TABLE credentials ADD COLUMN access_policy TEXT NOT NULL DEFAULT 'ops_only'`,
		"last_viewed_at":  `ALTER TABLE credentials ADD COLUMN last_viewed_at TEXT NOT NULL DEFAULT ''`,
		"last_viewed_by":  `ALTER TABLE credentials ADD COLUMN last_viewed_by TEXT NOT NULL DEFAULT ''`,
		"last_rotated_at": `ALTER TABLE credentials ADD COLUMN last_rotated_at TEXT NOT NULL DEFAULT ''`,
	})
}

func (s *DBStore) ensureAuditSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS audit_events (
			id TEXT PRIMARY KEY,
			actor TEXT NOT NULL DEFAULT '',
			actor_role TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL,
			target_type TEXT NOT NULL DEFAULT '',
			target_id TEXT NOT NULL DEFAULT '',
			target_name TEXT NOT NULL DEFAULT '',
			outcome TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events(actor, created_at DESC);
	`)
	return err
}

func (s *DBStore) ensureAlertSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS alert_records (
			id TEXT PRIMARY KEY,
			asset_id TEXT NOT NULL,
			source TEXT NOT NULL,
			severity TEXT NOT NULL,
			status TEXT NOT NULL,
			title TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			first_seen_at TEXT NOT NULL,
			last_seen_at TEXT NOT NULL,
			resolved_at TEXT NOT NULL DEFAULT '',
			resolved_by TEXT NOT NULL DEFAULT '',
			resolution TEXT NOT NULL DEFAULT '',
			event_count INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_alert_records_asset_status ON alert_records(asset_id, status, last_seen_at DESC);
		CREATE INDEX IF NOT EXISTS idx_alert_records_status ON alert_records(status, last_seen_at DESC);
	`)
	return err
}

func (s *DBStore) ensureInspectionAttachmentSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS inspection_attachments (
			id TEXT PRIMARY KEY,
			inspection_id TEXT NOT NULL,
			asset_id TEXT NOT NULL,
			file_name TEXT NOT NULL,
			content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
			size_bytes INTEGER NOT NULL DEFAULT 0,
			uploader TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			data BLOB NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (inspection_id) REFERENCES inspection_records(id) ON DELETE CASCADE,
			FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_inspection_attachments_inspection ON inspection_attachments(inspection_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_inspection_attachments_asset ON inspection_attachments(asset_id, created_at DESC);
	`)
	return err
}

func (s *DBStore) ensureColumns(ctx context.Context, table string, ddlByColumn map[string]string) error {
	columns, err := s.columnNames(ctx, table)
	if err != nil {
		return err
	}
	for name, ddl := range ddlByColumn {
		if columns[name] {
			continue
		}
		if _, err := s.db.ExecContext(ctx, ddl); err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) ensureCloudAccountColumns(ctx context.Context) error {
	columns, err := s.columnNames(ctx, "cloud_accounts")
	if err != nil {
		return err
	}

	if !columns["access_key_id_masked"] {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE cloud_accounts ADD COLUMN access_key_id_masked TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !columns["secret_access_key_masked"] {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE cloud_accounts ADD COLUMN secret_access_key_masked TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	costColumns := map[string]string{
		"cost_currency":           `ALTER TABLE cloud_accounts ADD COLUMN cost_currency TEXT NOT NULL DEFAULT ''`,
		"last_month_cost":         `ALTER TABLE cloud_accounts ADD COLUMN last_month_cost TEXT NOT NULL DEFAULT ''`,
		"last_month_to_date_cost": `ALTER TABLE cloud_accounts ADD COLUMN last_month_to_date_cost TEXT NOT NULL DEFAULT ''`,
		"current_month_cost":      `ALTER TABLE cloud_accounts ADD COLUMN current_month_cost TEXT NOT NULL DEFAULT ''`,
		"forecast_month_cost":     `ALTER TABLE cloud_accounts ADD COLUMN forecast_month_cost TEXT NOT NULL DEFAULT ''`,
		"month_over_month_delta":  `ALTER TABLE cloud_accounts ADD COLUMN month_over_month_delta TEXT NOT NULL DEFAULT ''`,
		"last_cost_sync_at":       `ALTER TABLE cloud_accounts ADD COLUMN last_cost_sync_at TEXT NOT NULL DEFAULT ''`,
		"last_cost_sync_status":   `ALTER TABLE cloud_accounts ADD COLUMN last_cost_sync_status TEXT NOT NULL DEFAULT ''`,
		"last_cost_sync_summary":  `ALTER TABLE cloud_accounts ADD COLUMN last_cost_sync_summary TEXT NOT NULL DEFAULT ''`,
	}
	for name, ddl := range costColumns {
		if !columns[name] {
			if _, err := s.db.ExecContext(ctx, ddl); err != nil {
				return err
			}
		}
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_cloud_account_syncs_account_id ON cloud_account_syncs(cloud_account_id, started_at DESC)`); err != nil {
		return err
	}
	return nil
}

func (s *DBStore) ensureCloudAccountCostRecordSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS cloud_account_cost_records (
			id TEXT PRIMARY KEY,
			cloud_account_id TEXT NOT NULL,
			period_start TEXT NOT NULL,
			period_end TEXT NOT NULL,
			granularity TEXT NOT NULL,
			dimension_type TEXT NOT NULL,
			dimension_name TEXT NOT NULL,
			currency TEXT NOT NULL DEFAULT '',
			amount TEXT NOT NULL DEFAULT '0.00',
			source TEXT NOT NULL DEFAULT 'aws_cost_explorer',
			synced_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (cloud_account_id) REFERENCES cloud_accounts(id)
		);
		CREATE INDEX IF NOT EXISTS idx_cloud_account_cost_records_account ON cloud_account_cost_records(cloud_account_id, period_start DESC);
		CREATE INDEX IF NOT EXISTS idx_cloud_account_cost_records_dimension ON cloud_account_cost_records(dimension_type, dimension_name, period_start DESC);
	`)
	return err
}
