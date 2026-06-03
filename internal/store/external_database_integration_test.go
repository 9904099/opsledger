package store

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/9904099/opsledger/internal/model"
)

func TestExternalDatabaseIntegration(t *testing.T) {
	cases := []struct {
		name   string
		driver string
		dsnEnv string
	}{
		{name: "postgres", driver: "postgres", dsnEnv: "OPSLEDGER_TEST_POSTGRES_DSN"},
		{name: "mysql", driver: "mysql", dsnEnv: "OPSLEDGER_TEST_MYSQL_DSN"},
	}

	ran := false
	for _, tc := range cases {
		dsn := strings.TrimSpace(os.Getenv(tc.dsnEnv))
		if dsn == "" {
			continue
		}
		ran = true
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPSLEDGER_DEV_SEED_USERS", "1")
			t.Setenv("OPSLEDGER_DEV_SEED_PASSWORD", "opsledger-test-password")
			t.Setenv("OPSLEDGER_CREDENTIAL_KEY", "opsledger-integration-test-key")
			resetExternalDatabase(t, tc.driver, dsn)
			t.Cleanup(func() {
				resetExternalDatabase(t, tc.driver, dsn)
			})

			store, err := NewStore(DatabaseConfig{Driver: tc.driver, DSN: dsn})
			if err != nil {
				t.Fatalf("new %s store: %v", tc.driver, err)
			}
			t.Cleanup(func() { _ = store.Close() })

			exerciseStoreCriticalPath(t, store)
		})
	}

	if !ran {
		t.Skip("set OPSLEDGER_TEST_POSTGRES_DSN or OPSLEDGER_TEST_MYSQL_DSN to run external database integration")
	}
}

func exerciseStoreCriticalPath(t *testing.T, store *DBStore) {
	t.Helper()
	ctx := context.Background()

	data, err := store.DashboardData(ctx)
	if err != nil {
		t.Fatalf("dashboard after init: %v", err)
	}
	if !containsPlatform(data.Platforms, "aws") || !containsPlatform(data.Platforms, "cloudflare") {
		t.Fatalf("seed platforms missing: %#v", data.Platforms)
	}
	if _, _, _, err := store.AuthenticateUser(ctx, model.LoginRequest{Username: "admin", Password: "opsledger-test-password"}, "127.0.0.1", "integration-test"); err != nil {
		t.Fatalf("authenticate seeded admin: %v", err)
	}

	account, err := store.CreateCloudAccount(ctx, model.CloudAccountUpsertRequest{
		PlatformCode:    "aws",
		Name:            "integration-aws",
		AccountID:       "123456789012",
		DefaultRegion:   "ap-southeast-1",
		Environment:     "dev",
		Owner:           "Ops",
		Criticality:     "medium",
		AccessKeyID:     "TESTACCESSKEY0002",
		SecretAccessKey: "integration-secret",
		SyncEnabled:     true,
		SyncMode:        "interval",
		SyncCron:        "6h",
	})
	if err != nil {
		t.Fatalf("create cloud account: %v", err)
	}
	if account.AccessKeyIDMasked == "" || strings.Contains(account.AccessKeyIDMasked, "ACCESS") {
		t.Fatalf("cloud account access key should be masked: %#v", account)
	}
	accessKey, secretKey, err := store.GetCloudAccountSecrets(ctx, account.ID)
	if err != nil {
		t.Fatalf("get cloud account secrets: %v", err)
	}
	if accessKey != "TESTACCESSKEY0002" || secretKey != "integration-secret" {
		t.Fatalf("cloud account secrets mismatch: %q / %q", accessKey, secretKey)
	}

	now := "2026-06-02T10:00:00+08:00"
	if err := store.SetCloudAccountCostResult(ctx, account.ID, model.CloudAccountCostResult{
		Currency:            "USD",
		LastMonthCost:       "100.00",
		LastMonthToDateCost: "50.00",
		CurrentMonthCost:    "66.00",
		ForecastMonthCost:   "198.00",
		MonthOverMonthDelta: "16.00",
		FinishedAt:          now,
		Summary:             "integration cost",
	}); err != nil {
		t.Fatalf("set cloud account cost: %v", err)
	}
	if err := store.RecordCloudAccountSync(ctx, model.CloudAccountSyncRecord{
		CloudAccountID:   account.ID,
		StartedAt:        now,
		FinishedAt:       now,
		Status:           "success",
		DiscoveredAssets: 1,
		CreatedAssets:    1,
		UpdatedAssets:    0,
		StaleAssets:      0,
		Warnings:         []string{"integration warning"},
		Breakdown:        map[string]int{"EC2": 1},
	}); err != nil {
		t.Fatalf("record cloud account sync: %v", err)
	}

	platform := mustPlatform(t, store, "aws")
	asset, err := store.CreateAsset(ctx, model.Asset{
		PlatformID:       platform.ID,
		PlatformCode:     platform.Code,
		PlatformName:     platform.Name,
		CloudAccountID:   account.ID,
		CloudAccountName: account.Name,
		AccountID:        account.AccountID,
		ProjectCode:      "business",
		Category:         "compute",
		ResourceType:     "EC2",
		Region:           "ap-southeast-1",
		Environment:      "dev",
		Name:             "integration-ec2",
		Endpoint:         "192.0.2.10",
		Owner:            "Ops",
		Status:           "active",
		Criticality:      "medium",
		LastCheckedAt:    now,
		Tags:             []string{"Project=business", "Environment=dev"},
		Notes:            "external database integration asset",
		Specs:            map[string]string{"instance_type": "t3.small"},
		Source:           "aws",
		ExternalID:       "aws:ec2:ap-southeast-1:i-integration",
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if _, _, err := store.UpsertAssetBySource(ctx, asset); err != nil {
		t.Fatalf("upsert asset by source: %v", err)
	}

	if _, err := store.CreateChange(ctx, model.ChangeRecord{
		AssetID:      asset.ID,
		Title:        "integration change",
		Category:     "config",
		Executor:     "Ops",
		RiskLevel:    "low",
		Window:       "2026-06-02 10:00-11:00",
		Status:       "planned",
		Summary:      "integration change",
		RollbackPlan: "rollback",
	}); err != nil {
		t.Fatalf("create change: %v", err)
	}

	inspection, err := store.CreateInspection(ctx, model.InspectionRecord{
		AssetID:   asset.ID,
		Executor:  "auto",
		Result:    "pass",
		Summary:   "integration inspection",
		CheckedAt: now,
	})
	if err != nil {
		t.Fatalf("create inspection: %v", err)
	}
	attachment, err := store.CreateInspectionAttachment(ctx, model.InspectionAttachmentCreateRequest{
		InspectionID: inspection.ID,
		FileName:     "integration.txt",
		ContentType:  "text/plain",
		Data:         []byte("external-db-attachment"),
		Uploader:     "admin",
		Description:  "integration",
	})
	if err != nil {
		t.Fatalf("create inspection attachment: %v", err)
	}
	loadedAttachment, payload, err := store.GetInspectionAttachment(ctx, attachment.ID)
	if err != nil {
		t.Fatalf("get inspection attachment: %v", err)
	}
	if loadedAttachment.SizeBytes != int64(len(payload)) || !bytes.Equal(payload, []byte("external-db-attachment")) {
		t.Fatalf("attachment payload mismatch: %#v %q", loadedAttachment, string(payload))
	}

	credential, err := store.UpsertCredential(ctx, model.CredentialUpsertRequest{
		OwnerType:    "asset",
		OwnerID:      asset.ID,
		Kind:         "password",
		KeyName:      "ssh",
		Value:        "asset-secret",
		Environment:  "dev",
		ProjectCode:  "business",
		AccessPolicy: "approval_required",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("upsert credential: %v", err)
	}
	developer := model.AppUser{Username: "developer", Role: "developer"}
	if _, err := store.RevealCredential(ctx, credential.ID, developer); !errors.Is(err, ErrForbidden) {
		t.Fatalf("developer reveal before approval error = %v, want ErrForbidden", err)
	}
	credentialApproval, err := store.CreateApproval(ctx, model.ApprovalCreateRequest{
		Requester:       "developer",
		RequestType:     "credential",
		TargetType:      "asset",
		TargetID:        asset.ID,
		Environment:     "dev",
		Reason:          "integration credential",
		PermissionLevel: "read",
		DurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create credential approval: %v", err)
	}
	if _, err := store.DecideApproval(ctx, credentialApproval.ID, model.ApprovalDecisionRequest{
		Approver:        "lead",
		Status:          "approved",
		DecisionSummary: "ok",
	}, model.AppUser{Username: "lead", Role: "lead"}); err != nil {
		t.Fatalf("approve credential request: %v", err)
	}
	revealed, err := store.RevealCredential(ctx, credential.ID, developer)
	if err != nil {
		t.Fatalf("developer reveal after approval: %v", err)
	}
	if revealed.Value != "asset-secret" {
		t.Fatalf("revealed credential = %q, want asset-secret", revealed.Value)
	}

	websshApproval, err := store.CreateApproval(ctx, model.ApprovalCreateRequest{
		Requester:       "developer",
		RequestType:     "webssh",
		TargetType:      "asset",
		TargetID:        asset.ID,
		Environment:     "dev",
		Reason:          "integration webssh",
		PermissionLevel: "connect",
		DurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create webssh approval: %v", err)
	}
	if _, err := store.DecideApproval(ctx, websshApproval.ID, model.ApprovalDecisionRequest{
		Approver:        "lead",
		Status:          "approved",
		DecisionSummary: "ok",
	}, model.AppUser{Username: "lead", Role: "lead"}); err != nil {
		t.Fatalf("approve webssh request: %v", err)
	}
	session, err := store.OpenWebSSH(ctx, developer, asset.ID, "127.0.0.1", "integration-test")
	if err != nil {
		t.Fatalf("open webssh: %v", err)
	}
	if session.Status != "active" || strings.Contains(session.LoginURL, "token=") {
		t.Fatalf("webssh session mismatch: %#v", session)
	}
	if _, err := store.ValidateWebSSHSession(ctx, developer, session.ID, asset.ID); err != nil {
		t.Fatalf("validate webssh session: %v", err)
	}
	if err := store.CloseWebSSHSession(ctx, developer, session.ID, "closed", "integration done", ""); err != nil {
		t.Fatalf("close webssh session: %v", err)
	}

	cloudflare := mustPlatform(t, store, "cloudflare")
	dnsAsset, err := store.CreateAsset(ctx, model.Asset{
		PlatformID:       cloudflare.ID,
		PlatformCode:     cloudflare.Code,
		PlatformName:     cloudflare.Name,
		CloudAccountName: "integration-cloudflare",
		AccountID:        "integration-cloudflare",
		ProjectCode:      "cloud",
		Category:         "network",
		ResourceType:     "DNS Record",
		Region:           "global",
		Environment:      "prod",
		Name:             "www.integration.example",
		Endpoint:         "https://www.integration.example",
		Owner:            "Ops",
		Status:           "active",
		Criticality:      "medium",
		LastCheckedAt:    now,
		Specs:            map[string]string{"type": "A"},
		Source:           "cloudflare",
		ExternalID:       "cf:dns:integration",
	})
	if err != nil {
		t.Fatalf("create dns asset: %v", err)
	}
	probeAssets, err := store.ListProbeAssets(ctx)
	if err != nil {
		t.Fatalf("list probe assets: %v", err)
	}
	if !slices.ContainsFunc(probeAssets, func(item model.Asset) bool { return item.ID == dnsAsset.ID }) {
		t.Fatalf("probe assets missing dns asset %s: %#v", dnsAsset.ID, probeAssets)
	}

	dashboard, err := store.DashboardData(ctx)
	if err != nil {
		t.Fatalf("dashboard after writes: %v", err)
	}
	if dashboard.Summary.TotalAssets < 2 || dashboard.Summary.PendingApprovals != 0 {
		t.Fatalf("dashboard summary mismatch: %#v", dashboard.Summary)
	}
	if len(dashboard.Attachments) == 0 || len(dashboard.Credentials) == 0 || len(dashboard.AccessGrants) < 2 {
		t.Fatalf("dashboard linked data incomplete: attachments=%d credentials=%d grants=%d", len(dashboard.Attachments), len(dashboard.Credentials), len(dashboard.AccessGrants))
	}
}

func containsPlatform(platforms []model.Platform, code string) bool {
	return slices.ContainsFunc(platforms, func(item model.Platform) bool {
		return item.Code == code
	})
}

func resetExternalDatabase(t *testing.T, driver string, dsn string) {
	t.Helper()
	dialect, sqlDriver, err := normalizeDatabaseDialect(driver)
	if err != nil {
		t.Fatalf("normalize driver %s: %v", driver, err)
	}
	if dialect == dialectMySQL {
		dsn = normalizeMySQLDSN(dsn)
	}
	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		t.Fatalf("open reset connection: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping reset connection: %v", err)
	}
	switch dialect {
	case dialectPostgres:
		resetPostgresSchema(t, db)
	case dialectMySQL:
		resetMySQLSchema(t, db)
	default:
		t.Fatalf("external integration only supports postgres/mysql, got %s", dialect)
	}
}

func resetPostgresSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = current_schema()
	`)
	if err != nil {
		t.Fatalf("list postgres tables: %v", err)
	}
	defer rows.Close()
	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("scan postgres table: %v", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("postgres table rows: %v", err)
	}
	if len(tables) == 0 {
		return
	}
	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		quoted = append(quoted, quoteIdentifier(table))
	}
	if _, err := db.Exec(`DROP TABLE IF EXISTS ` + strings.Join(quoted, ", ") + ` CASCADE`); err != nil {
		t.Fatalf("drop postgres tables: %v", err)
	}
}

func resetMySQLSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
	`)
	if err != nil {
		t.Fatalf("list mysql tables: %v", err)
	}
	defer rows.Close()
	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("scan mysql table: %v", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("mysql table rows: %v", err)
	}
	if len(tables) == 0 {
		return
	}
	if _, err := db.Exec(`SET FOREIGN_KEY_CHECKS=0`); err != nil {
		t.Fatalf("disable mysql foreign key checks: %v", err)
	}
	for _, table := range tables {
		if _, err := db.Exec("DROP TABLE IF EXISTS `" + strings.ReplaceAll(table, "`", "``") + "`"); err != nil {
			t.Fatalf("drop mysql table %s: %v", table, err)
		}
	}
	if _, err := db.Exec(`SET FOREIGN_KEY_CHECKS=1`); err != nil {
		t.Fatalf("enable mysql foreign key checks: %v", err)
	}
}
