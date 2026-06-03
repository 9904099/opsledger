package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9904099/opsledger/internal/model"
)

func mustPlatform(t *testing.T, store *DBStore, code string) model.Platform {
	t.Helper()
	platform, err := store.getPlatformByCode(context.Background(), code)
	if err != nil {
		t.Fatalf("get platform %s: %v", code, err)
	}
	return platform
}

func findApprovalForTest(ctx context.Context, store *DBStore, id string) (model.ApprovalRequest, error) {
	items, err := store.ListApprovals(ctx)
	if err != nil {
		return model.ApprovalRequest{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return model.ApprovalRequest{}, os.ErrNotExist
}

func TestNewStoreDatabaseConfig(t *testing.T) {
	sqliteStore, err := NewStore(DatabaseConfig{Driver: "sqlite", Path: filepath.Join(t.TempDir(), "opsledger.db")})
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	_ = sqliteStore.Close()

	defaultStore, err := NewStore(DatabaseConfig{Path: filepath.Join(t.TempDir(), "default.db")})
	if err != nil {
		t.Fatalf("new default store: %v", err)
	}
	_ = defaultStore.Close()

	if _, err := NewStore(DatabaseConfig{Driver: "postgres"}); err == nil || !strings.Contains(err.Error(), "dsn is required") {
		t.Fatalf("postgres should require dsn, got %v", err)
	}
	if _, err := NewStore(DatabaseConfig{Driver: "mysql"}); err == nil || !strings.Contains(err.Error(), "dsn is required") {
		t.Fatalf("mysql should require dsn, got %v", err)
	}
	if _, err := NewStore(DatabaseConfig{Driver: "oracle"}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported driver error mismatch: %v", err)
	}
}

func TestDatabaseDialectHelpers(t *testing.T) {
	query := "SELECT '?' AS literal, id FROM assets WHERE owner = ? AND name = ? -- ? in comment\nAND notes <> '?'"
	got := rewriteSQLPlaceholders(dialectPostgres, query)
	want := "SELECT '?' AS literal, id FROM assets WHERE owner = $1 AND name = $2 -- ? in comment\nAND notes <> '?'"
	if got != want {
		t.Fatalf("postgres placeholder rewrite mismatch:\n got: %s\nwant: %s", got, want)
	}
	if got := rewriteSQLPlaceholders(dialectSQLite, "WHERE id = ?"); got != "WHERE id = ?" {
		t.Fatalf("sqlite placeholders should not change: %s", got)
	}
	if got := (&DBStore{dialect: dialectPostgres}).schemaSQL(); !strings.Contains(got, "data BYTEA NOT NULL") {
		t.Fatalf("postgres schema should use BYTEA for attachment data")
	}
	if got := (&DBStore{dialect: dialectSQLite}).schemaSQL(); !strings.Contains(got, "data BLOB NOT NULL") {
		t.Fatalf("sqlite schema should use BLOB for attachment data")
	}
	mysqlSchema := (&DBStore{dialect: dialectMySQL}).schemaSQL()
	for _, fragment := range []string{"data LONGBLOB NOT NULL", "metadata_json JSON NOT NULL DEFAULT (JSON_OBJECT())", "CREATE INDEX idx_assets_updated_at ON assets(updated_at DESC)"} {
		if !strings.Contains(mysqlSchema, fragment) {
			t.Fatalf("mysql schema missing %q", fragment)
		}
	}
	mysqlUpsert := rewriteSQLPlaceholders(dialectMySQL, `
		INSERT INTO platforms (id, code, name, description, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT(code) DO UPDATE SET name = excluded.name, description = excluded.description, updated_at = excluded.updated_at
	`)
	if !strings.Contains(mysqlUpsert, "ON DUPLICATE KEY UPDATE") {
		t.Fatalf("mysql upsert should use ON DUPLICATE KEY UPDATE: %s", mysqlUpsert)
	}
	if got := (&DBStore{dialect: dialectPostgres}).dnsRecordTypePredicate(); !strings.Contains(got, "::jsonb") {
		t.Fatalf("postgres dns predicate should use jsonb, got %s", got)
	}
	if got := (&DBStore{dialect: dialectMySQL}).dnsRecordTypePredicate(); !strings.Contains(got, "JSON_EXTRACT") {
		t.Fatalf("mysql dns predicate should use JSON_EXTRACT, got %s", got)
	}
	if got := (&DBStore{dialect: dialectSQLite}).dnsRecordTypePredicate(); !strings.Contains(got, "json_extract") {
		t.Fatalf("sqlite dns predicate should use json_extract, got %s", got)
	}
	if got := normalizeMySQLDSN("user:pass@tcp(localhost:3306)/opsledger"); !strings.Contains(got, "parseTime=true") || !strings.Contains(got, "utf8mb4") {
		t.Fatalf("mysql dsn should get safe defaults: %s", got)
	}
}

func TestDBStoreCRUDAndCascadeDelete(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	platform := mustPlatform(t, store, "aws")
	asset, err := store.CreateAsset(ctx, model.Asset{
		PlatformID:       platform.ID,
		PlatformCode:     platform.Code,
		PlatformName:     platform.Name,
		CloudAccountName: "ops-prod",
		AccountID:        "ops-prod",
		Category:         "network",
		Region:           "ap-southeast-1",
		Environment:      "prod",
		ResourceType:     "ALB",
		Name:             "edge-public-alb",
		Endpoint:         "alb.example.com",
		Owner:            "SRE",
		Status:           "active",
		Criticality:      "high",
		LastCheckedAt:    "2026-05-29",
		Tags:             []string{" ingress ", "prod", "ingress"},
		Notes:            "公网入口。",
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}

	if got, want := len(asset.Tags), 2; got != want {
		t.Fatalf("asset tags len = %d, want %d", got, want)
	}

	change, err := store.CreateChange(ctx, model.ChangeRecord{
		AssetID:      asset.ID,
		Title:        "ALB 监听器扩容",
		Category:     "capacity",
		Executor:     "Ops",
		RiskLevel:    "medium",
		Window:       "2026-05-29 22:00-23:00",
		Status:       "planned",
		Summary:      "新增 443 监听器策略。",
		RollbackPlan: "回切旧监听器配置。",
	})
	if err != nil {
		t.Fatalf("create change: %v", err)
	}

	dashboard, err := store.DashboardData(ctx)
	if err != nil {
		t.Fatalf("dashboard data: %v", err)
	}
	if dashboard.Summary.TotalAssets < 1 {
		t.Fatalf("summary total assets = %d, want >= 1", dashboard.Summary.TotalAssets)
	}
	if change.AssetID != asset.ID {
		t.Fatalf("change asset id = %s, want %s", change.AssetID, asset.ID)
	}

	if err := store.DeleteAsset(ctx, asset.ID); err != nil {
		t.Fatalf("delete asset: %v", err)
	}

	dashboard, err = store.DashboardData(ctx)
	if err != nil {
		t.Fatalf("dashboard data after delete: %v", err)
	}
	for _, item := range dashboard.Changes {
		if item.AssetID == asset.ID {
			t.Fatalf("found dangling change %s for deleted asset %s", item.ID, asset.ID)
		}
	}
}

func TestCloudAccountCostRecordsUpsertAndList(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	account, err := store.CreateCloudAccount(ctx, model.CloudAccountUpsertRequest{
		PlatformCode:    "aws",
		Name:            "cost-test",
		AccountID:       "123456789012",
		DefaultRegion:   "ap-southeast-1",
		Environment:     "dev",
		Owner:           "Ops",
		Criticality:     "medium",
		AccessKeyID:     "TESTACCESSKEYCOST",
		SecretAccessKey: "secret",
	})
	if err != nil {
		t.Fatalf("create cloud account: %v", err)
	}

	record := model.CloudAccountCostRecord{
		CloudAccountID: account.ID,
		PeriodStart:    "2026-06-01",
		PeriodEnd:      "2026-06-02",
		Granularity:    "daily",
		DimensionType:  "total",
		DimensionName:  "total",
		Currency:       "USD",
		Amount:         "12.34",
		SyncedAt:       "2026-06-02T10:00:00Z",
	}
	if err := store.UpsertCloudAccountCostRecords(ctx, []model.CloudAccountCostRecord{record}); err != nil {
		t.Fatalf("upsert cost record: %v", err)
	}
	record.Amount = "13.37"
	if err := store.UpsertCloudAccountCostRecords(ctx, []model.CloudAccountCostRecord{record}); err != nil {
		t.Fatalf("upsert cost record again: %v", err)
	}

	items, err := store.ListCloudAccountCostRecords(ctx, 10)
	if err != nil {
		t.Fatalf("list cost records: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("cost record count = %d, want 1", len(items))
	}
	if items[0].Amount != "13.37" || items[0].CloudAccountID != account.ID {
		t.Fatalf("cost record mismatch: %#v", items[0])
	}

	dashboard, err := store.DashboardData(ctx)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if len(dashboard.CostRecords) != 1 || dashboard.CostRecords[0].Amount != "13.37" {
		t.Fatalf("dashboard cost records mismatch: %#v", dashboard.CostRecords)
	}
}

func TestDBStoreAlertLifecycle(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	platform := mustPlatform(t, store, "cloudflare")
	asset, err := store.CreateAsset(ctx, model.Asset{
		PlatformID:       platform.ID,
		PlatformCode:     platform.Code,
		PlatformName:     platform.Name,
		CloudAccountName: "cloudflare-example",
		AccountID:        "cloudflare-example",
		Category:         "network",
		ResourceType:     "DNS Record",
		Name:             "api.example.com",
		Endpoint:         "https://api.example.com",
		Environment:      "prod",
		Owner:            "SRE",
		Status:           "active",
		Criticality:      "high",
		LastCheckedAt:    "2026-06-02",
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}

	alert, err := store.UpsertAlert(ctx, model.AlertUpsertRequest{
		AssetID:  asset.ID,
		Source:   "probe",
		Severity: "critical",
		Title:    "拨测异常",
		Summary:  "HTTP 500",
		SeenAt:   "2026-06-02T10:00:00+08:00",
	})
	if err != nil {
		t.Fatalf("upsert alert: %v", err)
	}
	alert, err = store.UpsertAlert(ctx, model.AlertUpsertRequest{
		AssetID:  asset.ID,
		Source:   "probe",
		Severity: "critical",
		Title:    "拨测异常",
		Summary:  "HTTP 503",
		SeenAt:   "2026-06-02T10:05:00+08:00",
	})
	if err != nil {
		t.Fatalf("upsert alert again: %v", err)
	}
	if alert.EventCount != 2 || alert.Summary != "HTTP 503" {
		t.Fatalf("alert after second upsert = %+v", alert)
	}

	dashboard, err := store.DashboardData(ctx)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.Summary.OpenAlerts != 1 {
		t.Fatalf("open alerts = %d, want 1", dashboard.Summary.OpenAlerts)
	}

	resolved, err := store.ResolveAlert(ctx, alert.ID, model.AlertResolveRequest{Resolver: "ops", Resolution: "已恢复"})
	if err != nil {
		t.Fatalf("resolve alert: %v", err)
	}
	if resolved.Status != "resolved" || resolved.ResolvedBy != "ops" {
		t.Fatalf("resolved alert = %+v", resolved)
	}
}

func TestDBStoreInspectionAttachmentLifecycle(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	platform := mustPlatform(t, store, "aws")
	asset, err := store.CreateAsset(ctx, model.Asset{
		PlatformID:       platform.ID,
		PlatformCode:     platform.Code,
		PlatformName:     platform.Name,
		CloudAccountName: "manual",
		AccountID:        "manual",
		ProjectCode:      "public",
		Category:         "manual",
		ResourceType:     "Manual Service",
		Region:           "global",
		Environment:      "dev",
		Name:             "inspection-asset",
		Owner:            "SRE",
		Status:           "active",
		Criticality:      "medium",
		LastCheckedAt:    "2026-06-02",
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}
	inspection, err := store.CreateInspection(ctx, model.InspectionRecord{
		AssetID:   asset.ID,
		Executor:  "auto-probe",
		Result:    "failed",
		Summary:   "probe failed",
		CheckedAt: "2026-06-02T10:00:00+08:00",
	})
	if err != nil {
		t.Fatalf("create inspection: %v", err)
	}
	attachment, err := store.CreateInspectionAttachment(ctx, model.InspectionAttachmentCreateRequest{
		InspectionID: inspection.ID,
		FileName:     "../probe.log",
		ContentType:  "text/plain",
		Data:         []byte("probe output"),
		Uploader:     "ops",
		Description:  "拨测日志",
	})
	if err != nil {
		t.Fatalf("create attachment: %v", err)
	}
	if attachment.FileName != "probe.log" || attachment.AssetID != asset.ID || attachment.SizeBytes != int64(len("probe output")) {
		t.Fatalf("attachment metadata mismatch: %+v", attachment)
	}

	items, err := store.ListInspectionAttachments(ctx)
	if err != nil {
		t.Fatalf("list attachments: %v", err)
	}
	if len(items) != 1 || items[0].ID != attachment.ID {
		t.Fatalf("attachments = %#v", items)
	}
	downloaded, data, err := store.GetInspectionAttachment(ctx, attachment.ID)
	if err != nil {
		t.Fatalf("get attachment: %v", err)
	}
	if downloaded.ID != attachment.ID || string(data) != "probe output" {
		t.Fatalf("downloaded attachment = %+v data=%q", downloaded, string(data))
	}
}

func TestDBStorePersistsUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opsledger.db")
	store, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()
	platform := mustPlatform(t, store, "aliyun")
	asset, err := store.CreateAsset(ctx, model.Asset{
		PlatformID:       platform.ID,
		PlatformCode:     platform.Code,
		PlatformName:     platform.Name,
		CloudAccountName: "sandbox",
		AccountID:        "sandbox",
		Category:         "compute",
		Region:           "cn-hangzhou",
		Environment:      "dev",
		ResourceType:     "ECS",
		Name:             "ops-dev-ecs",
		Owner:            "DevOps",
		Status:           "maintenance",
		Criticality:      "low",
		LastCheckedAt:    "2026-05-29",
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}

	asset.Owner = "Platform"
	asset.Status = "active"
	if _, err := store.UpdateAsset(ctx, asset.ID, asset); err != nil {
		t.Fatalf("update asset: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reloaded, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	defer reloaded.Close()

	dashboard, err := reloaded.DashboardData(ctx)
	if err != nil {
		t.Fatalf("reload dashboard: %v", err)
	}

	found := false
	for _, item := range dashboard.Assets {
		if item.ID == asset.ID {
			found = true
			if item.Owner != "Platform" || item.Status != "active" {
				t.Fatalf("reloaded asset = %+v", item)
			}
		}
	}
	if !found {
		t.Fatalf("updated asset %s not found after reload", asset.ID)
	}
}

func TestDBStoreEnvironmentSeedIsGeneric(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	envs, err := store.ListEnvironments(ctx)
	if err != nil {
		t.Fatalf("list environments: %v", err)
	}

	got := map[string]string{}
	order := []string{}
	for _, env := range envs {
		got[env.Code] = env.Status
		order = append(order, env.Code)
	}
	for code, want := range map[string]string{
		"dev":     "active",
		"prod":    "guarded",
		"local":   "active",
		"test":    "reserved",
		"staging": "reserved",
	} {
		if got[code] != want {
			t.Fatalf("environment %s status = %q, want %q; order=%v", code, got[code], want, order)
		}
	}
}

func TestDBStoreUpsertAssetBySource(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	platform := mustPlatform(t, store, "aws")
	created, isCreated, err := store.UpsertAssetBySource(ctx, model.Asset{
		PlatformID:       platform.ID,
		PlatformCode:     platform.Code,
		PlatformName:     platform.Name,
		CloudAccountName: "dev-account",
		AccountID:        "123456789012",
		Category:         "compute",
		Region:           "ap-southeast-1",
		Environment:      "prod",
		ResourceType:     "EC2",
		Name:             "web-01",
		Endpoint:         "192.0.2.10",
		Owner:            "Ops",
		Status:           "active",
		Criticality:      "medium",
		LastCheckedAt:    "2026-05-29",
		Source:           "aws",
		ExternalID:       "aws:ec2:ap-southeast-1:i-123",
	})
	if err != nil {
		t.Fatalf("upsert create: %v", err)
	}
	if !isCreated {
		t.Fatalf("expected create on first upsert")
	}

	created.Endpoint = "192.0.2.11"
	created.Name = "web-01-renamed"
	updated, isCreated, err := store.UpsertAssetBySource(ctx, created)
	if err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if isCreated {
		t.Fatalf("expected update on second upsert")
	}
	if updated.ID != created.ID {
		t.Fatalf("updated id = %s, want %s", updated.ID, created.ID)
	}

	dashboard, err := store.DashboardData(ctx)
	if err != nil {
		t.Fatalf("dashboard data: %v", err)
	}

	matches := 0
	for _, item := range dashboard.Assets {
		if item.Source == "aws" && item.ExternalID == "aws:ec2:ap-southeast-1:i-123" {
			matches++
			if item.Endpoint != "192.0.2.11" || item.Name != "web-01-renamed" {
				t.Fatalf("unexpected updated asset: %+v", item)
			}
		}
	}
	if matches != 1 {
		t.Fatalf("expected exactly one imported asset, got %d", matches)
	}
}

func TestDBStoreMarkAssetsStaleBySource(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	platform := mustPlatform(t, store, "aws")
	base := model.Asset{
		PlatformID:       platform.ID,
		PlatformCode:     platform.Code,
		PlatformName:     platform.Name,
		CloudAccountID:   "cloud-account-1",
		CloudAccountName: "business",
		AccountID:        "123456789012",
		Category:         "compute",
		Region:           "ap-southeast-1",
		Environment:      "prod",
		ResourceType:     "EC2",
		Owner:            "Ops",
		Status:           "active",
		Criticality:      "medium",
		LastCheckedAt:    "2026-06-01",
		Source:           "aws",
	}

	active := base
	active.Name = "active-ec2"
	active.ExternalID = "aws:ec2:ap-southeast-1:i-active"
	if _, _, err := store.UpsertAssetBySource(ctx, active); err != nil {
		t.Fatalf("upsert active: %v", err)
	}
	stale := base
	stale.Name = "missing-ec2"
	stale.ExternalID = "aws:ec2:ap-southeast-1:i-missing"
	if _, _, err := store.UpsertAssetBySource(ctx, stale); err != nil {
		t.Fatalf("upsert missing: %v", err)
	}
	otherRegion := base
	otherRegion.Name = "other-region-ec2"
	otherRegion.Region = "us-east-1"
	otherRegion.ExternalID = "aws:ec2:us-east-1:i-other"
	if _, _, err := store.UpsertAssetBySource(ctx, otherRegion); err != nil {
		t.Fatalf("upsert other region: %v", err)
	}

	count, err := store.MarkAssetsStaleBySource(ctx, "cloud-account-1", "aws", []string{active.ExternalID}, []string{"ap-southeast-1"}, "2026-06-02")
	if err != nil {
		t.Fatalf("mark stale: %v", err)
	}
	if count != 1 {
		t.Fatalf("stale count = %d, want 1", count)
	}

	dashboard, err := store.DashboardData(ctx)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	statusByExternalID := map[string]string{}
	for _, item := range dashboard.Assets {
		statusByExternalID[item.ExternalID] = item.Status
	}
	if statusByExternalID[active.ExternalID] != "active" {
		t.Fatalf("active status = %q, want active", statusByExternalID[active.ExternalID])
	}
	if statusByExternalID[stale.ExternalID] != "stale" {
		t.Fatalf("stale status = %q, want stale", statusByExternalID[stale.ExternalID])
	}
	if statusByExternalID[otherRegion.ExternalID] != "active" {
		t.Fatalf("other region status = %q, want active", statusByExternalID[otherRegion.ExternalID])
	}
}

func TestCloudAccountCredentialsAreStoredAsCredentialItems(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	account, err := store.CreateCloudAccount(ctx, model.CloudAccountUpsertRequest{
		PlatformCode:    "aws",
		Name:            "credential-test",
		AccountID:       "123456789012",
		DefaultRegion:   "ap-southeast-1",
		Environment:     "dev",
		Owner:           "Ops",
		Criticality:     "medium",
		AccessKeyID:     "TESTACCESSKEY0001",
		SecretAccessKey: "super-secret-access-key",
		SyncEnabled:     true,
		SyncMode:        "manual",
		SyncCron:        "",
	})
	if err != nil {
		t.Fatalf("create cloud account: %v", err)
	}

	accessKeyID, secretAccessKey, err := store.GetCloudAccountSecrets(ctx, account.ID)
	if err != nil {
		t.Fatalf("get cloud account secrets: %v", err)
	}
	if accessKeyID != "TESTACCESSKEY0001" || secretAccessKey != "super-secret-access-key" {
		t.Fatalf("unexpected secrets: %q %q", accessKeyID, secretAccessKey)
	}

	var rawAccessKey string
	var rawSecretKey string
	if err := store.db.QueryRowContext(ctx, `
		SELECT access_key_id, secret_access_key
		FROM cloud_accounts
		WHERE id = ?
	`, account.ID).Scan(&rawAccessKey, &rawSecretKey); err != nil {
		t.Fatalf("query raw cloud account secrets: %v", err)
	}
	if rawAccessKey != "" || rawSecretKey != "" {
		t.Fatalf("cloud_accounts should not store raw secrets, got %q %q", rawAccessKey, rawSecretKey)
	}

	credentials, err := store.ListCredentials(ctx)
	if err != nil {
		t.Fatalf("list credentials: %v", err)
	}
	if len(credentials) != 2 {
		t.Fatalf("credentials len = %d, want 2: %#v", len(credentials), credentials)
	}

	var accessKeyCredential model.CredentialItem
	for _, credential := range credentials {
		if credential.Kind == "access_key_id" {
			accessKeyCredential = credential
		}
	}
	if accessKeyCredential.ID == "" {
		t.Fatalf("access key credential not found: %#v", credentials)
	}
	if accessKeyCredential.MaskedValue != "TEST****0001" {
		t.Fatalf("unexpected masked value: %q", accessKeyCredential.MaskedValue)
	}

	if _, err := store.RevealCredential(ctx, accessKeyCredential.ID, model.AppUser{Username: "developer", Role: "developer"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("developer reveal error = %v, want ErrForbidden", err)
	}
	revealed, err := store.RevealCredential(ctx, accessKeyCredential.ID, model.AppUser{Username: "ops", Role: "ops"})
	if err != nil {
		t.Fatalf("ops reveal credential: %v", err)
	}
	if revealed.Value != "TESTACCESSKEY0001" {
		t.Fatalf("revealed value = %q", revealed.Value)
	}
	if revealed.Credential.LastViewedBy != "ops" || revealed.Credential.LastViewedAt == "" {
		t.Fatalf("view audit fields not updated: %#v", revealed.Credential)
	}
}

func TestGenericCredentialUpsertAndCopyAudit(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	tool, err := store.CreateTool(ctx, model.ToolAssetUpsertRequest{
		Environment:      "global",
		ToolType:         "ci",
		Name:             "Jenkins",
		Endpoint:         "https://jenkins.example.com",
		Owner:            "Ops",
		Status:           "active",
		Criticality:      "medium",
		LoginPolicy:      "shared_credential",
		CredentialPolicy: "approval_required",
		Description:      "ci",
	})
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}

	credential, err := store.UpsertCredential(ctx, model.CredentialUpsertRequest{
		OwnerType:    "asset",
		OwnerID:      tool.AssetID,
		Kind:         "password",
		KeyName:      "admin",
		Value:        "jenkins-admin-password",
		AccessPolicy: "approval_required",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("upsert credential: %v", err)
	}
	if credential.OwnerType != "asset" || credential.OwnerID != tool.AssetID {
		t.Fatalf("credential owner mismatch: %#v", credential)
	}
	if credential.MaskedValue == "" || credential.MaskedValue == "jenkins-admin-password" {
		t.Fatalf("credential should be masked, got %q", credential.MaskedValue)
	}

	revealed, err := store.RevealCredential(ctx, credential.ID, model.AppUser{Username: "admin", Role: "admin"})
	if err != nil {
		t.Fatalf("reveal credential: %v", err)
	}
	if revealed.Value != "jenkins-admin-password" {
		t.Fatalf("revealed value = %q", revealed.Value)
	}

	copied, err := store.RecordCredentialCopy(ctx, credential.ID, model.AppUser{Username: "ops", Role: "ops"})
	if err != nil {
		t.Fatalf("record credential copy: %v", err)
	}
	if copied.LastViewedBy != "ops" || copied.LastViewedAt == "" {
		t.Fatalf("copy audit fields not updated: %#v", copied)
	}
}

func TestDeveloperCanRevealCredentialWithGrant(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	tool, err := store.CreateTool(ctx, model.ToolAssetUpsertRequest{
		Environment:      "dev",
		ToolType:         "deploy",
		Name:             "Deploy Console",
		Endpoint:         "https://deploy.example.com",
		Owner:            "Ops",
		Status:           "active",
		Criticality:      "medium",
		LoginPolicy:      "shared_credential",
		CredentialPolicy: "approval_required",
	})
	if err != nil {
		t.Fatalf("create tool: %v", err)
	}
	credential, err := store.UpsertCredential(ctx, model.CredentialUpsertRequest{
		OwnerType:    "asset",
		OwnerID:      tool.AssetID,
		Kind:         "api_token",
		KeyName:      "deploy",
		Value:        "deploy-token",
		AccessPolicy: "approval_required",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("upsert credential: %v", err)
	}
	developer := model.AppUser{Username: "developer", Role: "developer"}
	if _, err := store.RevealCredential(ctx, credential.ID, developer); !errors.Is(err, ErrForbidden) {
		t.Fatalf("developer reveal without grant error = %v, want ErrForbidden", err)
	}
	approval, err := store.CreateApproval(ctx, model.ApprovalCreateRequest{
		Requester:       "developer",
		RequestType:     "credential",
		TargetType:      "asset",
		TargetID:        tool.AssetID,
		Environment:     "dev",
		Reason:          "debug deploy",
		PermissionLevel: "read",
		DurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if _, err := store.DecideApproval(ctx, approval.ID, model.ApprovalDecisionRequest{
		Approver:        "lead",
		Status:          "approved",
		DecisionSummary: "ok",
	}, model.AppUser{Username: "lead", Role: "lead"}); err != nil {
		t.Fatalf("decide approval: %v", err)
	}
	revealed, err := store.RevealCredential(ctx, credential.ID, developer)
	if err != nil {
		t.Fatalf("developer reveal with grant: %v", err)
	}
	if revealed.Value != "deploy-token" {
		t.Fatalf("revealed value = %q", revealed.Value)
	}
}

func TestApprovalFlowStepsAndWebSSHGrant(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	platform := mustPlatform(t, store, "aws")
	asset, err := store.CreateAsset(ctx, model.Asset{
		PlatformID:       platform.ID,
		PlatformCode:     platform.Code,
		PlatformName:     platform.Name,
		CloudAccountName: "example-dev",
		AccountID:        "123456789012",
		ProjectCode:      "business",
		Category:         "compute",
		ResourceType:     "EC2",
		Region:           "ap-southeast-1",
		Environment:      "dev",
		Name:             "dev-ec2",
		Endpoint:         "192.0.2.8",
		Owner:            "Ops",
		Status:           "active",
		Criticality:      "medium",
		LastCheckedAt:    "2026-06-02",
		Source:           "aws",
		ExternalID:       "aws:ec2:ap-southeast-1:i-webssh",
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}

	if _, err := store.OpenWebSSH(ctx, model.AppUser{Username: "developer", Role: "developer"}, asset.ID, "127.0.0.1", "test"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("open webssh without grant error = %v, want ErrForbidden", err)
	}

	approval, err := store.CreateApproval(ctx, model.ApprovalCreateRequest{
		Requester:       "developer",
		RequestType:     "webssh",
		TargetType:      "asset",
		TargetID:        asset.ID,
		Environment:     "dev",
		Reason:          "debug dev vm",
		PermissionLevel: "connect",
		DurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if approval.Status != "pending" || len(approval.Tasks) != 1 || approval.Tasks[0].ApproverRole != "lead" || approval.Tasks[0].Status != "pending" {
		t.Fatalf("unexpected approval tasks: %#v", approval)
	}
	if _, err := store.DecideApproval(ctx, approval.ID, model.ApprovalDecisionRequest{
		Approver:        "developer",
		Status:          "approved",
		DecisionSummary: "self approve",
	}, model.AppUser{Username: "developer", Role: "developer"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("developer self approval error = %v, want ErrForbidden", err)
	}

	approved, err := store.DecideApproval(ctx, approval.ID, model.ApprovalDecisionRequest{
		Approver:        "lead",
		Status:          "approved",
		DecisionSummary: "ok",
	}, model.AppUser{Username: "lead", Role: "lead"})
	if err != nil {
		t.Fatalf("lead approve: %v", err)
	}
	if approved.Status != "approved" || approved.DecidedAt == "" {
		t.Fatalf("approved request mismatch: %#v", approved)
	}

	grants, err := store.ListAccessGrants(ctx)
	if err != nil {
		t.Fatalf("list grants: %v", err)
	}
	if len(grants) != 1 {
		t.Fatalf("grant count = %d, want 1: %#v", len(grants), grants)
	}
	grant := grants[0]
	if grant.Action != "webssh" || grant.TargetID != asset.ID || grant.TemporaryCredentialHash == "" || grant.TemporaryCredential != "" {
		t.Fatalf("unexpected grant: %#v", grant)
	}

	session, err := store.OpenWebSSH(ctx, model.AppUser{Username: "developer", Role: "developer"}, asset.ID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("open webssh with grant: %v", err)
	}
	if session.Status != "active" || session.AccessGrantID != grant.ID {
		t.Fatalf("unexpected session: %#v", session)
	}
	if strings.Contains(session.LoginURL, grant.TemporaryCredentialHash) || strings.Contains(session.LoginURL, "token=") {
		t.Fatalf("login url should not contain temporary credential or token query: %s", session.LoginURL)
	}
	if _, err := store.ValidateWebSSHSession(ctx, model.AppUser{Username: "developer", Role: "developer"}, session.ID, asset.ID); err != nil {
		t.Fatalf("validate webssh session: %v", err)
	}
	if _, err := store.ValidateWebSSHSession(ctx, model.AppUser{Username: "other", Role: "developer"}, session.ID, asset.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other user validate error = %v, want ErrForbidden", err)
	}
	if err := store.CloseWebSSHSession(ctx, model.AppUser{Username: "developer", Role: "developer"}, session.ID, "closed", "done", ""); err != nil {
		t.Fatalf("close session: %v", err)
	}
	if _, err := store.ValidateWebSSHSession(ctx, model.AppUser{Username: "developer", Role: "developer"}, session.ID, asset.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("closed session validate error = %v, want ErrForbidden", err)
	}
}

func TestApprovalFlowMultiStepRequiresCurrentApprover(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	flow, err := store.CreateApprovalFlow(ctx, model.ApprovalFlowUpsertRequest{
		Name:        "prod credential multi-step test",
		Scope:       "credential",
		Environment: "local",
		Status:      "active",
		Steps: []model.ApprovalFlowStepRequest{
			{ApproverRole: "ops", ApproverLabel: "Ops Engineer", RequiredAction: "approved", TimeoutMinutes: 30},
			{ApproverRole: "admin", ApproverLabel: "Platform Admin", RequiredAction: "approved", TimeoutMinutes: 60},
		},
	})
	if err != nil {
		t.Fatalf("create flow: %v", err)
	}
	if len(flow.Steps) != 2 {
		t.Fatalf("flow steps = %d, want 2", len(flow.Steps))
	}

	approval, err := store.CreateApproval(ctx, model.ApprovalCreateRequest{
		Requester:       "developer",
		RequestType:     "credential",
		TargetType:      "asset",
		TargetID:        "asset-prod-credential",
		Environment:     "local",
		Reason:          "inspect prod credential",
		PermissionLevel: "read",
		DurationMinutes: 15,
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if approval.FlowID != flow.ID || len(approval.Tasks) != 2 || approval.Tasks[0].Status != "pending" || approval.Tasks[1].Status != "waiting" {
		t.Fatalf("approval should use custom multi-step flow: %#v", approval)
	}
	if _, err := store.DecideApproval(ctx, approval.ID, model.ApprovalDecisionRequest{
		Approver:        "admin",
		Status:          "approved",
		DecisionSummary: "skip ops",
	}, model.AppUser{Username: "admin", Role: "admin"}); err != nil {
		t.Fatalf("admin should be allowed as break-glass approver: %v", err)
	}
	afterAdmin, err := findApprovalForTest(ctx, store, approval.ID)
	if err != nil {
		t.Fatalf("reload approval: %v", err)
	}
	if afterAdmin.Status != "pending" || afterAdmin.Tasks[0].Status != "approved" || afterAdmin.Tasks[1].Status != "pending" {
		t.Fatalf("after first approval mismatch: %#v", afterAdmin)
	}
	if _, err := store.DecideApproval(ctx, approval.ID, model.ApprovalDecisionRequest{
		Approver:        "ops",
		Status:          "approved",
		DecisionSummary: "wrong step",
	}, model.AppUser{Username: "ops", Role: "ops"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ops on admin step error = %v, want ErrForbidden", err)
	}
	final, err := store.DecideApproval(ctx, approval.ID, model.ApprovalDecisionRequest{
		Approver:        "admin",
		Status:          "approved",
		DecisionSummary: "done",
	}, model.AppUser{Username: "admin", Role: "admin"})
	if err != nil {
		t.Fatalf("admin final approve: %v", err)
	}
	if final.Status != "approved" {
		t.Fatalf("final status = %q, want approved", final.Status)
	}
}

func TestRolePermissionProjectScopePersists(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	permission, err := store.CreatePermission(ctx, model.RolePermissionUpsertRequest{
		Role:             "developer",
		Scope:            "credential",
		Action:           "view",
		Environment:      "dev",
		ProjectCode:      "business",
		RequiresApproval: true,
	})
	if err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if permission.ProjectCode != "business" {
		t.Fatalf("project_code = %q, want business", permission.ProjectCode)
	}
	permissions, err := store.ListPermissionsForRole(ctx, "developer")
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	found := false
	for _, item := range permissions {
		if item.ID == permission.ID {
			found = true
			if item.ProjectCode != "business" {
				t.Fatalf("listed project_code = %q", item.ProjectCode)
			}
		}
	}
	if !found {
		t.Fatalf("permission %s not listed", permission.ID)
	}
}
