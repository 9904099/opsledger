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

func (s *DBStore) seedPlatforms(ctx context.Context) error {
	now := time.Now().Format(time.RFC3339)
	platforms := []struct {
		Code string
		Name string
		Desc string
	}{
		{Code: "aws", Name: "AWS", Desc: "Amazon Web Services"},
		{Code: "cloudflare", Name: "Cloudflare", Desc: "Cloudflare edge network and DNS"},
		{Code: "pve", Name: "PVE", Desc: "Proxmox Virtual Environment"},
		{Code: "aliyun", Name: "Aliyun", Desc: "Alibaba Cloud"},
		{Code: "tencent", Name: "Tencent Cloud", Desc: "Tencent Cloud"},
	}

	for _, item := range platforms {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO platforms (id, code, name, description, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, ?, ?)
			ON CONFLICT(code) DO UPDATE SET name = excluded.name, description = excluded.description, updated_at = excluded.updated_at
		`, "platform-"+item.Code, item.Code, item.Name, item.Desc, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) seedEnvironments(ctx context.Context) error {
	now := time.Now().Format(time.RFC3339)
	items := []model.Environment{
		{ID: "env-dev", Code: "dev", Name: "Development", Tier: "dev", Owner: "Ops", Description: "Shared development environment.", Status: "active"},
		{ID: "env-prod", Code: "prod", Name: "Production", Tier: "prod", Owner: "Ops", Description: "Production environment. Sensitive operations should require approval.", Status: "guarded"},
		{ID: "env-local", Code: "local", Name: "Local / On-prem", Tier: "local", Owner: "Ops", Description: "Local datacenter, lab, or on-premises environment.", Status: "active"},
		{ID: "env-test", Code: "test", Name: "Test", Tier: "test", Owner: "Ops", Description: "Reserved test environment.", Status: "reserved"},
		{ID: "env-staging", Code: "staging", Name: "Staging", Tier: "staging", Owner: "Ops", Description: "Reserved staging environment.", Status: "reserved"},
	}
	for _, item := range items {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO environments (id, code, name, tier, owner, description, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(code) DO UPDATE SET name = excluded.name, tier = excluded.tier, owner = excluded.owner,
				description = excluded.description, status = excluded.status, updated_at = excluded.updated_at
		`, item.ID, item.Code, item.Name, item.Tier, item.Owner, item.Description, item.Status, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) seedUsersAndPermissions(ctx context.Context, seedDevUsers bool) error {
	now := time.Now().Format(time.RFC3339)
	if seedDevUsers {
		users := []model.AppUser{
			{ID: "usr-admin", Username: "admin", DisplayName: "Platform Admin", Role: "admin", Team: "Platform", Status: "active"},
			{ID: "usr-dev", Username: "developer", DisplayName: "Developer", Role: "developer", Team: "Engineering", Status: "active"},
			{ID: "usr-lead", Username: "lead", DisplayName: "Development Lead", Role: "lead", Team: "Engineering", Status: "active"},
			{ID: "usr-ops", Username: "ops", DisplayName: "Ops Engineer", Role: "ops", Team: "Operations", Status: "active"},
			{ID: "usr-auditor", Username: "auditor", DisplayName: "Auditor", Role: "auditor", Team: "Audit", Status: "active"},
		}
		for _, user := range users {
			devPassword := devSeedPassword(user.Username)
			if devPassword == "" {
				return errors.New("OPSLEDGER_DEV_SEED_USERS=1 requires OPSLEDGER_DEV_SEED_PASSWORD or OPSLEDGER_DEV_PASSWORD_<USERNAME>")
			}
			hashed, err := hashPassword(devPassword)
			if err != nil {
				return err
			}
			_, err = s.db.ExecContext(ctx, `
				INSERT INTO app_users (
					id, username, display_name, email, phone, role, team, status, auth_source, external_subject,
					password_hash, password_changed_at, failed_login_count, locked_until, last_login_at, created_at, updated_at
				)
				VALUES (?, ?, ?, '', '', ?, ?, ?, 'local', '', ?, ?, 0, '', '', ?, ?)
				ON CONFLICT(username) DO UPDATE SET display_name = excluded.display_name, role = excluded.role,
					team = excluded.team, auth_source = excluded.auth_source,
					password_hash = CASE WHEN app_users.password_hash = '' THEN excluded.password_hash ELSE app_users.password_hash END,
					password_changed_at = CASE WHEN app_users.password_hash = '' THEN excluded.password_changed_at ELSE app_users.password_changed_at END,
					updated_at = excluded.updated_at
			`, user.ID, user.Username, user.DisplayName, user.Role, user.Team, user.Status, hashed, now, now, now)
			if err != nil {
				return err
			}
		}
	}

	permissions := []model.RolePermission{
		{ID: "perm-dev-tool-dev", Role: "developer", Scope: "tool", Action: "view", Environment: "dev", RequiresApproval: false},
		{ID: "perm-dev-tool-test", Role: "developer", Scope: "tool", Action: "view", Environment: "test", RequiresApproval: false},
		{ID: "perm-dev-credential-prod", Role: "developer", Scope: "credential", Action: "view", Environment: "prod", RequiresApproval: true},
		{ID: "perm-dev-webssh-prod", Role: "developer", Scope: "webssh", Action: "connect", Environment: "prod", RequiresApproval: true},
		{ID: "perm-lead-approval-dev", Role: "lead", Scope: "approval", Action: "decide", Environment: "dev", RequiresApproval: false},
		{ID: "perm-lead-approval-test", Role: "lead", Scope: "approval", Action: "decide", Environment: "test", RequiresApproval: false},
		{ID: "perm-ops-all", Role: "ops", Scope: "*", Action: "*", Environment: "*", RequiresApproval: false},
		{ID: "perm-admin-all", Role: "admin", Scope: "*", Action: "*", Environment: "*", RequiresApproval: false},
	}
	for _, permission := range permissions {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO role_permissions (id, role, scope, action, environment, project_code, requires_approval, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET role = excluded.role, scope = excluded.scope, action = excluded.action,
				environment = excluded.environment, project_code = excluded.project_code, requires_approval = excluded.requires_approval, updated_at = excluded.updated_at
		`, permission.ID, permission.Role, permission.Scope, permission.Action, permission.Environment, defaultString(permission.ProjectCode, "*"), boolToInt(permission.RequiresApproval), now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func devSeedPassword(username string) string {
	envName := "OPSLEDGER_DEV_PASSWORD_" + strings.ToUpper(strings.ReplaceAll(username, "-", "_"))
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv("OPSLEDGER_DEV_SEED_PASSWORD"))
}

func (s *DBStore) seedApprovalFlows(ctx context.Context) error {
	now := time.Now().Format(time.RFC3339)
	roles := []model.RoleDefinition{
		{ID: "role-admin", Code: "admin", Name: "Platform Admin", Description: "Manage system settings, permissions, and high-risk approvals.", Level: 10, Status: "active"},
		{ID: "role-ops", Code: "ops", Name: "Ops Engineer", Description: "Maintain cloud accounts, assets, tools, and approvals.", Level: 20, Status: "active"},
		{ID: "role-lead", Code: "lead", Name: "Development Lead", Description: "Review team requests in development and test environments.", Level: 30, Status: "active"},
		{ID: "role-developer", Code: "developer", Name: "Developer", Description: "Use environment entries and request credentials or WebSSH.", Level: 40, Status: "active"},
		{ID: "role-viewer", Code: "viewer", Name: "Viewer", Description: "Read ledger and runtime status.", Level: 80, Status: "active"},
		{ID: "role-auditor", Code: "auditor", Name: "Auditor", Description: "Review approvals, audit events, and change history.", Level: 90, Status: "active"},
	}
	for _, role := range roles {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO roles (id, code, name, description, level, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(code) DO UPDATE SET name = excluded.name, description = excluded.description,
				level = excluded.level, status = excluded.status, updated_at = excluded.updated_at
		`, role.ID, role.Code, role.Name, role.Description, role.Level, role.Status, now, now)
		if err != nil {
			return err
		}
	}

	flowCount := 0
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM approval_flows`).Scan(&flowCount); err != nil {
		return err
	}
	if flowCount > 0 {
		return nil
	}

	defaults := []model.ApprovalFlowUpsertRequest{
		{
			Name:        "Production Credential Access Approval",
			Scope:       "credential",
			Environment: "prod",
			Status:      "active",
			Description: "Production credential access is reviewed by Operations and confirmed by Platform Admin.",
			Steps: []model.ApprovalFlowStepRequest{
				{ApproverRole: "ops", ApproverLabel: "Ops Engineer", RequiredAction: "approved", TimeoutMinutes: 60},
				{ApproverRole: "admin", ApproverLabel: "Platform Admin", RequiredAction: "approved", TimeoutMinutes: 120},
			},
		},
		{
			Name:        "Development WebSSH Approval",
			Scope:       "webssh",
			Environment: "dev",
			Status:      "active",
			Description: "Development WebSSH access is reviewed by the Development Lead.",
			Steps: []model.ApprovalFlowStepRequest{
				{ApproverRole: "lead", ApproverLabel: "Development Lead", RequiredAction: "approved", TimeoutMinutes: 60},
			},
		},
	}
	for _, flow := range defaults {
		if _, err := s.CreateApprovalFlow(ctx, flow); err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) seedDefaultTools(ctx context.Context) error {
	defaultTools := []model.ToolAssetUpsertRequest{
		{
			Environment:      "dev",
			ToolType:         "ops",
			Name:             "OpsLedger Local",
			Endpoint:         "http://127.0.0.1:18090/",
			Owner:            "Ops",
			Status:           "active",
			Criticality:      "medium",
			Tags:             []string{"opsledger", "local", "dev"},
			Description:      "Local OpsLedger development console.",
			LoginPolicy:      "shared_credential",
			CredentialPolicy: "viewable",
		},
		{
			Environment:      "dev",
			ToolType:         "git",
			Name:             "Local Git Service",
			Endpoint:         "http://127.0.0.1:3000",
			Owner:            "Ops",
			Status:           "active",
			Criticality:      "medium",
			Tags:             []string{"git", "local", "dev"},
			Description:      "Example local Git service endpoint.",
			LoginPolicy:      "shared_credential",
			CredentialPolicy: "viewable",
		},
		{
			Environment:      "dev",
			ToolType:         "registry",
			Name:             "Local Container Registry",
			Endpoint:         "http://127.0.0.1:5000/v2/",
			Owner:            "Ops",
			Status:           "active",
			Criticality:      "medium",
			Tags:             []string{"registry", "docker", "local"},
			Description:      "Example local container registry.",
			LoginPolicy:      "shared_credential",
			CredentialPolicy: "viewable",
		},
		{
			Environment:      "prod",
			ToolType:         "cluster",
			Name:             "Production Cluster Console",
			Endpoint:         "https://cluster.example.com",
			Owner:            "Ops",
			Status:           "active",
			Criticality:      "high",
			Tags:             []string{"cluster", "prod", "ops"},
			Description:      "Example production cluster management console.",
			LoginPolicy:      "sso",
			CredentialPolicy: "approval_required",
			ApprovalRequired: true,
		},
		{
			Environment:      "prod",
			ToolType:         "monitor",
			Name:             "Production Monitoring",
			Endpoint:         "https://monitoring.example.com",
			Owner:            "Ops",
			Status:           "active",
			Criticality:      "high",
			Tags:             []string{"monitoring", "prod", "ops"},
			Description:      "Example production monitoring console.",
			LoginPolicy:      "sso",
			CredentialPolicy: "approval_required",
			ApprovalRequired: true,
		},
		{
			Environment:      "global",
			ToolType:         "dns",
			Name:             "DNS Provider Console",
			Endpoint:         "https://dns.example.com",
			Owner:            "Ops",
			Status:           "active",
			Criticality:      "high",
			Tags:             []string{"dns", "global", "example"},
			Description:      "Example DNS provider console.",
			LoginPolicy:      "sso",
			CredentialPolicy: "approval_required",
			ApprovalRequired: true,
		},
		{
			Environment:      "global",
			ToolType:         "docs",
			Name:             "Architecture Docs",
			Endpoint:         "https://docs.example.com",
			Owner:            "Ops",
			Status:           "active",
			Criticality:      "medium",
			Tags:             []string{"docs", "global", "example"},
			Description:      "Example architecture documentation portal.",
			LoginPolicy:      "sso",
			CredentialPolicy: "none",
		},
	}

	for _, req := range defaultTools {
		if exists, err := s.toolExists(ctx, req.Environment, req.ToolType, req.Endpoint); err != nil {
			return err
		} else if exists {
			continue
		}
		if _, err := s.CreateTool(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) toolExists(ctx context.Context, environment, toolType, endpoint string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM tool_assets t
		JOIN assets a ON a.id = t.asset_id
		WHERE t.environment = ? AND t.tool_type = ? AND a.endpoint = ?
	`, strings.TrimSpace(environment), strings.TrimSpace(toolType), strings.TrimSpace(endpoint)).Scan(&count)
	return count > 0, err
}

func (s *DBStore) backfillLegacyAssetRelations(ctx context.Context) error {
	assetColumns, err := s.columnNames(ctx, "assets")
	if err != nil {
		return err
	}
	if !assetColumns["cloud_account_id"] || !assetColumns["platform_code"] || !assetColumns["account_id"] {
		return nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT platform_code, platform_name, account_id, environment, owner, criticality
		FROM assets
		WHERE COALESCE(cloud_account_id, '') = ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type legacyGroup struct {
		platformCode string
		platformName string
		accountID    string
		environment  string
		owner        string
		criticality  string
	}
	var groups []legacyGroup
	for rows.Next() {
		var g legacyGroup
		if err := rows.Scan(&g.platformCode, &g.platformName, &g.accountID, &g.environment, &g.owner, &g.criticality); err != nil {
			return err
		}
		if strings.TrimSpace(g.platformCode) == "" {
			continue
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, group := range groups {
		platform, err := s.getPlatformByCode(ctx, group.platformCode)
		if err != nil {
			continue
		}

		accountName := group.accountID
		if accountName == "" {
			accountName = group.platformName + " imported"
		}

		cloudAccountID, err := s.ensureLegacyCloudAccount(ctx, platform, accountName, group.accountID, group.environment, group.owner, group.criticality)
		if err != nil {
			return err
		}

		if _, err := s.db.ExecContext(ctx, `
			UPDATE assets
			SET platform_id = ?, cloud_account_id = ?, cloud_account_name = ?
			WHERE platform_code = ? AND COALESCE(account_id, '') = ? AND COALESCE(cloud_account_id, '') = ''
		`, platform.ID, cloudAccountID, accountName, group.platformCode, group.accountID); err != nil {
			return err
		}
	}

	return nil
}

func (s *DBStore) ensureLegacyCloudAccount(ctx context.Context, platform model.Platform, name, accountID, environment, owner, criticality string) (string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM cloud_accounts
		WHERE platform_id = ? AND name = ? AND account_id = ?
		LIMIT 1
	`, platform.ID, name, accountID)

	var id string
	if err := row.Scan(&id); err == nil {
		return id, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	now := time.Now().Format(time.RFC3339)
	id = newID("acct")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cloud_accounts (
			id, platform_id, name, account_id, default_region, environment, owner, criticality,
			access_key_id, secret_access_key, access_key_id_masked, secret_access_key_masked,
			sync_enabled, sync_mode, sync_cron, last_sync_at, last_sync_status, last_sync_summary,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, '', ?, ?, ?, '', '', '', '', 0, 'manual', '', '', '', '', ?, ?)
	`, id, platform.ID, name, accountID, defaultString(environment, "prod"), defaultString(owner, "Ops"), defaultString(criticality, "medium"), now, now)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *DBStore) getPlatformByCode(ctx context.Context, code string) (model.Platform, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, code, name, description, enabled, created_at, updated_at
		FROM platforms
		WHERE code = ?
	`, strings.ToLower(strings.TrimSpace(code)))

	var item model.Platform
	var enabled int
	if err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Platform{}, os.ErrNotExist
		}
		return model.Platform{}, err
	}
	item.Enabled = enabled == 1
	return item, nil
}
