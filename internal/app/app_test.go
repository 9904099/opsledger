package app

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func TestFilterDashboardForDeveloperDataScope(t *testing.T) {
	now := time.Now().UTC()
	data := model.DashboardData{
		Assets: []model.Asset{
			{ID: "asset-business-dev", ProjectCode: "business", Environment: "dev", Name: "business dev", Status: "active"},
			{ID: "asset-business-prod", ProjectCode: "business", Environment: "prod", Name: "business prod", Status: "active"},
			{ID: "asset-cloud-dev", ProjectCode: "cloud", Environment: "dev", Name: "cloud dev", Status: "active"},
			{ID: "asset-public-test", ProjectCode: "public", Environment: "test", Name: "public test", Status: "active"},
		},
		Tools: []model.ToolAsset{
			{ID: "tool-global", AssetID: "asset-tool-global", Environment: "global", ToolType: "tool", AssetName: "global tool"},
			{ID: "tool-prod", AssetID: "asset-business-prod", Environment: "prod", ToolType: "business", AssetName: "prod app"},
			{ID: "tool-dev-cloud", AssetID: "asset-cloud-dev", Environment: "dev", ToolType: "business", AssetName: "cloud app"},
		},
		AccessGrants: []model.AccessGrant{
			{ID: "grant-old", Username: "developer", TargetType: "asset", TargetID: "asset-cloud-dev", Status: "expired", ExpiresAt: now.Add(-time.Hour).Format(time.RFC3339)},
		},
		Approvals: []model.ApprovalRequest{
			{ID: "approval-own", Requester: "developer", Status: "pending"},
			{ID: "approval-other", Requester: "other", Status: "pending"},
		},
		Changes: []model.ChangeRecord{
			{ID: "change-dev", AssetID: "asset-business-dev"},
			{ID: "change-prod", AssetID: "asset-business-prod"},
		},
		Probes: []model.ProbeRecord{
			{ID: "probe-dev", AssetID: "asset-business-dev", Status: "up", CheckedAt: now.Format(time.RFC3339)},
			{ID: "probe-prod", AssetID: "asset-business-prod", Status: "down", CheckedAt: now.Format(time.RFC3339)},
		},
		Users:         []model.AppUser{{Username: "admin"}},
		CloudAccounts: []model.CloudAccount{{ID: "cloud-account"}},
		AuditEvents:   []model.AuditEvent{{ID: "audit"}},
	}

	filtered := filterDashboardForUser(data, model.AppUser{
		Username: "developer",
		Role:     "developer",
		Team:     "Engineering",
	})

	assertAssetIDs(t, filtered.Assets, []string{"asset-public-test"})
	assertToolIDs(t, filtered.Tools, []string{"tool-global"})
	if len(filtered.AccessGrants) != 0 {
		t.Fatalf("developer should not receive expired grants, got %d", len(filtered.AccessGrants))
	}
	if len(filtered.CloudAccounts) != 0 || len(filtered.Users) != 0 || len(filtered.AuditEvents) != 0 {
		t.Fatalf("developer received sensitive admin data: cloud_accounts=%d users=%d audit_events=%d", len(filtered.CloudAccounts), len(filtered.Users), len(filtered.AuditEvents))
	}
	if got := len(filtered.Approvals); got != 1 || filtered.Approvals[0].ID != "approval-own" {
		t.Fatalf("developer approvals mismatch: got %#v", filtered.Approvals)
	}
	if got := len(filtered.Changes); got != 0 {
		t.Fatalf("developer changes mismatch: got %#v", filtered.Changes)
	}
	if got := filtered.Summary.TotalAssets; got != 1 {
		t.Fatalf("summary should be recalculated after filtering, got total_assets=%d", got)
	}
}

func TestShouldAutoSyncAccount(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	account := model.CloudAccount{
		SyncEnabled: true,
		SyncMode:    "auto",
		SyncCron:    "2h",
		LastSyncAt:  now.Add(-3 * time.Hour).Format(time.RFC3339),
	}
	if !shouldAutoSyncAccount(account, now) {
		t.Fatalf("expected account to sync after interval")
	}
	account.LastSyncAt = now.Add(-30 * time.Minute).Format(time.RFC3339)
	if shouldAutoSyncAccount(account, now) {
		t.Fatalf("did not expect account to sync before interval")
	}
	account.SyncMode = "manual"
	if shouldAutoSyncAccount(account, now) {
		t.Fatalf("manual mode should not auto sync")
	}
	account.SyncMode = "auto"
	account.SyncEnabled = false
	if shouldAutoSyncAccount(account, now) {
		t.Fatalf("disabled account should not auto sync")
	}
}

func TestSyncInterval(t *testing.T) {
	cases := []struct {
		name     string
		mode     string
		expr     string
		expected time.Duration
	}{
		{name: "duration", mode: "interval", expr: "30m", expected: 30 * time.Minute},
		{name: "hour cron", mode: "cron", expr: "0 */6 * * *", expected: 6 * time.Hour},
		{name: "minute cron", mode: "cron", expr: "*/15 * * * *", expected: 15 * time.Minute},
		{name: "daily cron", mode: "cron", expr: "0 2 * * *", expected: 24 * time.Hour},
		{name: "manual", mode: "manual", expr: "1h", expected: 0},
		{name: "default auto", mode: "auto", expr: "", expected: 6 * time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := syncInterval(tc.mode, tc.expr); got != tc.expected {
				t.Fatalf("syncInterval(%q, %q) = %s, want %s", tc.mode, tc.expr, got, tc.expected)
			}
		})
	}
}

func TestFilterDashboardIncludesGrantedProdAsset(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).Format(time.RFC3339)
	data := model.DashboardData{
		Assets: []model.Asset{
			{ID: "asset-business-prod", ProjectCode: "business", Environment: "prod", Name: "prod", Status: "active"},
			{ID: "asset-cloud-prod", ProjectCode: "cloud", Environment: "prod", Name: "cloud prod", Status: "active"},
		},
		Tools: []model.ToolAsset{
			{ID: "tool-prod", AssetID: "asset-business-prod", Environment: "prod", ToolType: "business"},
		},
		AccessGrants: []model.AccessGrant{
			{ID: "grant-prod", Username: "developer", TargetType: "asset", TargetID: "asset-business-prod", Status: "active", ExpiresAt: expiresAt},
		},
	}

	filtered := filterDashboardForUser(data, model.AppUser{
		Username: "developer",
		Role:     "developer",
		Team:     "Engineering",
	})

	assertAssetIDs(t, filtered.Assets, []string{"asset-business-prod"})
	assertToolIDs(t, filtered.Tools, []string{"tool-prod"})
	if got := len(filtered.AccessGrants); got != 1 || filtered.AccessGrants[0].ID != "grant-prod" {
		t.Fatalf("developer grant mismatch: %#v", filtered.AccessGrants)
	}
}

func TestFilterDashboardAuditorReadOnlyFullAssets(t *testing.T) {
	data := model.DashboardData{
		Assets: []model.Asset{
			{ID: "asset-prod", ProjectCode: "cloud", Environment: "prod", Status: "active"},
		},
		Tools:         []model.ToolAsset{{ID: "tool-prod", AssetID: "asset-prod", Environment: "prod"}},
		CloudAccounts: []model.CloudAccount{{ID: "cloud-account"}},
		AuditEvents:   []model.AuditEvent{{ID: "audit"}},
	}

	filtered := filterDashboardForUser(data, model.AppUser{Username: "auditor", Role: "auditor"})

	assertAssetIDs(t, filtered.Assets, []string{"asset-prod"})
	assertToolIDs(t, filtered.Tools, []string{"tool-prod"})
	if len(filtered.CloudAccounts) != 0 {
		t.Fatalf("auditor should not receive config internals via bootstrap")
	}
	if len(filtered.AuditEvents) != 1 {
		t.Fatalf("auditor should receive audit events via bootstrap")
	}
}

func TestHTTPFirstSetupCreatesInitialAdmin(t *testing.T) {
	server, err := NewServer(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	csrf := fetchCSRFToken(t, handler)
	status := apiRequest(t, handler, http.MethodGet, "/api/setup", "", "", nil)
	if status.Code != http.StatusOK {
		t.Fatalf("setup status = %d, want 200; body=%s", status.Code, status.Body.String())
	}
	var setupStatus model.SetupStatusResponse
	if err := json.Unmarshal(status.Body.Bytes(), &setupStatus); err != nil {
		t.Fatalf("decode setup status: %v", err)
	}
	if !setupStatus.Required || !setupStatus.DatabaseReady || setupStatus.UserCount != 0 {
		t.Fatalf("unexpected setup status: %#v", setupStatus)
	}

	created := apiRequest(t, handler, http.MethodPost, "/api/setup", csrf, "", map[string]any{
		"username":         "admin",
		"display_name":     "Platform Admin",
		"email":            "admin@example.com",
		"password":         "strong-admin-password",
		"confirm_password": "strong-admin-password",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("setup create status = %d, want 201; body=%s", created.Code, created.Body.String())
	}
	var login model.LoginResponse
	if err := json.Unmarshal(created.Body.Bytes(), &login); err != nil {
		t.Fatalf("decode setup login: %v", err)
	}
	if login.User.Username != "admin" || login.User.Role != "admin" {
		t.Fatalf("unexpected setup user: %#v", login.User)
	}
	session := ""
	for _, cookie := range created.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			session = cookie.Value
		}
	}
	if session == "" {
		t.Fatalf("setup should create a login session")
	}

	repeated := apiRequest(t, handler, http.MethodPost, "/api/setup", csrf, "", map[string]any{
		"username":         "admin2",
		"display_name":     "Admin 2",
		"password":         "another-strong-password",
		"confirm_password": "another-strong-password",
	})
	if repeated.Code != http.StatusBadRequest {
		t.Fatalf("repeat setup status = %d, want 400; body=%s", repeated.Code, repeated.Body.String())
	}
	bootstrap := apiRequest(t, handler, http.MethodGet, "/api/bootstrap", "", session, nil)
	if bootstrap.Code != http.StatusOK {
		t.Fatalf("bootstrap with setup session status = %d, want 200; body=%s", bootstrap.Code, bootstrap.Body.String())
	}
}

func TestHTTPRBACAndCSRF(t *testing.T) {
	t.Setenv("OPSLEDGER_DEV_SEED_USERS", "1")
	t.Setenv("OPSLEDGER_DEV_SEED_PASSWORD", "opsledger-test-password")
	server, err := NewServer(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	csrf := fetchCSRFToken(t, handler)
	developerSession := loginForTest(t, handler, csrf, "developer", "opsledger-test-password")

	forbidden := apiRequest(t, handler, http.MethodPost, "/api/tools", csrf, developerSession, map[string]any{
		"environment":       "dev",
		"tool_type":         "ops",
		"name":              "forbidden tool",
		"endpoint":          "https://forbidden.example.com",
		"owner":             "Ops",
		"status":            "active",
		"criticality":       "medium",
		"login_policy":      "sso",
		"credential_policy": "none",
	})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("developer POST /api/tools status = %d, want 403; body=%s", forbidden.Code, forbidden.Body.String())
	}

	approval := apiRequest(t, handler, http.MethodPost, "/api/approvals", csrf, developerSession, map[string]any{
		"requester":        "spoofed-admin",
		"request_type":     "webssh",
		"target_type":      "asset",
		"target_id":        "asset-http-test",
		"environment":      "dev",
		"reason":           "debug dev host",
		"permission_level": "connect",
		"duration_minutes": 30,
	})
	if approval.Code != http.StatusCreated {
		t.Fatalf("developer POST /api/approvals status = %d, want 201; body=%s", approval.Code, approval.Body.String())
	}
	var created model.ApprovalRequest
	if err := json.Unmarshal(approval.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode approval: %v", err)
	}
	if created.Requester != "developer" {
		t.Fatalf("approval requester = %q, want developer", created.Requester)
	}

	noCSRF := apiRequest(t, handler, http.MethodPost, "/api/approvals", "", developerSession, map[string]any{
		"request_type": "webssh",
		"target_type":  "asset",
		"target_id":    "asset-http-test",
		"environment":  "dev",
		"reason":       "missing csrf",
	})
	if noCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d, want 403", noCSRF.Code)
	}

	events, err := server.store.ListAuditEvents(context.Background(), 20)
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	hasForbidden := false
	for _, event := range events {
		if event.Action == "auth.forbidden" && event.Actor == "developer" && event.TargetID == "/api/tools" {
			hasForbidden = true
		}
	}
	if !hasForbidden {
		t.Fatalf("expected auth.forbidden audit event for developer /api/tools, got %#v", events)
	}
}

func TestLogoutKeepsCSRFForImmediateRelogin(t *testing.T) {
	t.Setenv("OPSLEDGER_DEV_SEED_USERS", "1")
	t.Setenv("OPSLEDGER_DEV_SEED_PASSWORD", "opsledger-test-password")
	server, err := NewServer(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	csrf := fetchCSRFToken(t, handler)
	session := loginForTest(t, handler, csrf, "admin", "opsledger-test-password")

	logout := apiRequest(t, handler, http.MethodPost, "/api/auth/logout", csrf, session, map[string]any{})
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want 200; body=%s", logout.Code, logout.Body.String())
	}
	for _, cookie := range logout.Result().Cookies() {
		if cookie.Name == csrfCookieName && cookie.MaxAge < 0 {
			t.Fatalf("logout should not clear csrf cookie")
		}
	}

	relogin := apiRequest(t, handler, http.MethodPost, "/api/auth/login", csrf, "", map[string]any{
		"username": "admin",
		"password": "opsledger-test-password",
	})
	if relogin.Code != http.StatusOK {
		t.Fatalf("relogin without page refresh status = %d, want 200; body=%s", relogin.Code, relogin.Body.String())
	}
}

func TestHTTPAdminCanUseConfigAPI(t *testing.T) {
	t.Setenv("OPSLEDGER_DEV_SEED_USERS", "1")
	t.Setenv("OPSLEDGER_DEV_SEED_PASSWORD", "opsledger-test-password")
	server, err := NewServer(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	csrf := fetchCSRFToken(t, handler)
	adminSession := loginForTest(t, handler, csrf, "admin", "opsledger-test-password")

	response := apiRequest(t, handler, http.MethodPost, "/api/tools", csrf, adminSession, map[string]any{
		"environment":       "dev",
		"tool_type":         "ops",
		"name":              "admin tool",
		"endpoint":          "https://admin-tool.example.com",
		"owner":             "Ops",
		"status":            "active",
		"criticality":       "medium",
		"login_policy":      "sso",
		"credential_policy": "none",
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("admin POST /api/tools status = %d, want 201; body=%s", response.Code, response.Body.String())
	}
}

func TestHTTPAssetExportImportRBAC(t *testing.T) {
	t.Setenv("OPSLEDGER_DEV_SEED_USERS", "1")
	t.Setenv("OPSLEDGER_DEV_SEED_PASSWORD", "opsledger-test-password")
	server, err := NewServer(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	csrf := fetchCSRFToken(t, handler)
	adminSession := loginForTest(t, handler, csrf, "admin", "opsledger-test-password")
	developerSession := loginForTest(t, handler, csrf, "developer", "opsledger-test-password")

	importResponse := apiRequest(t, handler, http.MethodPost, "/api/assets/import", csrf, adminSession, map[string]any{
		"assets": []map[string]any{
			{
				"platform_code":      "manual",
				"platform_name":      "Manual",
				"cloud_account_name": "manual",
				"account_id":         "manual",
				"project_code":       "public",
				"category":           "tool",
				"resource_type":      "Manual Service",
				"region":             "global",
				"environment":        "dev",
				"name":               "imported-manual-service",
				"endpoint":           "https://manual.example.com",
				"owner":              "Ops",
				"status":             "active",
				"criticality":        "medium",
				"last_checked_at":    "2026-06-02",
				"tags":               []string{"manual", "import"},
				"notes":              "import test",
			},
		},
	})
	if importResponse.Code != http.StatusCreated {
		t.Fatalf("admin POST /api/assets/import status = %d, want 201; body=%s", importResponse.Code, importResponse.Body.String())
	}
	var importResult model.AssetImportResult
	if err := json.Unmarshal(importResponse.Body.Bytes(), &importResult); err != nil {
		t.Fatalf("decode import result: %v", err)
	}
	if importResult.ImportedAssets != 1 || len(importResult.CreatedIDs) != 1 {
		t.Fatalf("import result mismatch: %#v", importResult)
	}

	exportResponse := apiRequest(t, handler, http.MethodGet, "/api/assets/export", csrf, adminSession, nil)
	if exportResponse.Code != http.StatusOK {
		t.Fatalf("admin GET /api/assets/export status = %d, want 200; body=%s", exportResponse.Code, exportResponse.Body.String())
	}
	var exported model.AssetImportRequest
	if err := json.Unmarshal(exportResponse.Body.Bytes(), &exported); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	found := false
	for _, asset := range exported.Assets {
		if asset.ID == importResult.CreatedIDs[0] && asset.Name == "imported-manual-service" {
			found = true
		}
	}
	if !found {
		t.Fatalf("imported asset %s not found in export", importResult.CreatedIDs[0])
	}

	developerImport := apiRequest(t, handler, http.MethodPost, "/api/assets/import", csrf, developerSession, map[string]any{"assets": []any{}})
	if developerImport.Code != http.StatusForbidden {
		t.Fatalf("developer POST /api/assets/import status = %d, want 403", developerImport.Code)
	}
	developerExport := apiRequest(t, handler, http.MethodGet, "/api/assets/export", csrf, developerSession, nil)
	if developerExport.Code != http.StatusForbidden {
		t.Fatalf("developer GET /api/assets/export status = %d, want 403", developerExport.Code)
	}
}

func TestHTTPAssetBulkUpdateRBAC(t *testing.T) {
	t.Setenv("OPSLEDGER_DEV_SEED_USERS", "1")
	t.Setenv("OPSLEDGER_DEV_SEED_PASSWORD", "opsledger-test-password")
	server, err := NewServer(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	csrf := fetchCSRFToken(t, handler)
	adminSession := loginForTest(t, handler, csrf, "admin", "opsledger-test-password")
	developerSession := loginForTest(t, handler, csrf, "developer", "opsledger-test-password")

	first := createAssetByAPI(t, handler, csrf, adminSession, "bulk-a")
	second := createAssetByAPI(t, handler, csrf, adminSession, "bulk-b")

	response := apiRequest(t, handler, http.MethodPost, "/api/assets/bulk-update", csrf, adminSession, map[string]any{
		"asset_ids":    []string{first.ID, second.ID},
		"project_code": "business",
		"environment":  "local",
		"status":       "maintenance",
		"criticality":  "low",
		"owner":        "SRE",
		"tags":         []string{"bulk", "updated"},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("admin bulk update status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	var result model.AssetBulkUpdateResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode bulk result: %v", err)
	}
	if result.UpdatedAssets != 2 || len(result.UpdatedIDs) != 2 {
		t.Fatalf("bulk update result mismatch: %#v", result)
	}

	exportResponse := apiRequest(t, handler, http.MethodGet, "/api/assets/export", csrf, adminSession, nil)
	if exportResponse.Code != http.StatusOK {
		t.Fatalf("export status = %d: %s", exportResponse.Code, exportResponse.Body.String())
	}
	var exported model.AssetImportRequest
	if err := json.Unmarshal(exportResponse.Body.Bytes(), &exported); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	seen := 0
	for _, asset := range exported.Assets {
		if asset.ID != first.ID && asset.ID != second.ID {
			continue
		}
		seen++
		if asset.ProjectCode != "business" || asset.Environment != "local" || asset.Status != "maintenance" || asset.Criticality != "low" || asset.Owner != "SRE" {
			t.Fatalf("asset not patched correctly: %#v", asset)
		}
		if asset.Category != "manual" || asset.ResourceType != "Manual Service" {
			t.Fatalf("unspecified fields should be preserved: %#v", asset)
		}
		assertStringSet(t, asset.Tags, []string{"bulk", "updated"})
	}
	if seen != 2 {
		t.Fatalf("seen patched assets = %d, want 2", seen)
	}

	developerResponse := apiRequest(t, handler, http.MethodPost, "/api/assets/bulk-update", csrf, developerSession, map[string]any{
		"asset_ids": []string{first.ID},
		"status":    "offline",
	})
	if developerResponse.Code != http.StatusForbidden {
		t.Fatalf("developer bulk update status = %d, want 403", developerResponse.Code)
	}
}

func TestHTTPAlertResolveRBAC(t *testing.T) {
	t.Setenv("OPSLEDGER_DEV_SEED_USERS", "1")
	t.Setenv("OPSLEDGER_DEV_SEED_PASSWORD", "opsledger-test-password")
	server, err := NewServer(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	csrf := fetchCSRFToken(t, handler)
	adminSession := loginForTest(t, handler, csrf, "admin", "opsledger-test-password")
	developerSession := loginForTest(t, handler, csrf, "developer", "opsledger-test-password")

	asset := createAssetByAPI(t, handler, csrf, adminSession, "alert-target")
	alert, err := server.store.UpsertAlert(context.Background(), model.AlertUpsertRequest{
		AssetID:  asset.ID,
		Source:   "probe",
		Severity: "critical",
		Title:    "拨测异常",
		Summary:  "HTTP 500",
	})
	if err != nil {
		t.Fatalf("upsert alert: %v", err)
	}

	developerResolve := apiRequest(t, handler, http.MethodPost, "/api/alerts/"+alert.ID+"/resolve", csrf, developerSession, map[string]any{
		"resolution": "越权处理",
	})
	if developerResolve.Code != http.StatusForbidden {
		t.Fatalf("developer resolve alert status = %d, want 403", developerResolve.Code)
	}

	adminResolve := apiRequest(t, handler, http.MethodPost, "/api/alerts/"+alert.ID+"/resolve", csrf, adminSession, map[string]any{
		"resolution": "已处理",
	})
	if adminResolve.Code != http.StatusOK {
		t.Fatalf("admin resolve alert status = %d, want 200; body=%s", adminResolve.Code, adminResolve.Body.String())
	}
	var resolved model.AlertRecord
	if err := json.Unmarshal(adminResolve.Body.Bytes(), &resolved); err != nil {
		t.Fatalf("decode alert: %v", err)
	}
	if resolved.Status != "resolved" || resolved.ResolvedBy != "admin" {
		t.Fatalf("resolved alert = %+v", resolved)
	}
}

func TestHTTPInspectionAttachmentRBAC(t *testing.T) {
	t.Setenv("OPSLEDGER_DEV_SEED_USERS", "1")
	t.Setenv("OPSLEDGER_DEV_SEED_PASSWORD", "opsledger-test-password")
	server, err := NewServer(filepath.Join(t.TempDir(), "opsledger.db"))
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	csrf := fetchCSRFToken(t, handler)
	adminSession := loginForTest(t, handler, csrf, "admin", "opsledger-test-password")
	developerSession := loginForTest(t, handler, csrf, "developer", "opsledger-test-password")

	asset := createAssetByAPI(t, handler, csrf, adminSession, "attachment-target")
	inspection, err := server.store.CreateInspection(context.Background(), model.InspectionRecord{
		AssetID:   asset.ID,
		Executor:  "auto-probe",
		Result:    "failed",
		Summary:   "probe failed",
		CheckedAt: "2026-06-02T10:00:00+08:00",
	})
	if err != nil {
		t.Fatalf("create inspection: %v", err)
	}

	developerUpload := multipartRequest(t, handler, http.MethodPost, "/api/inspections/"+inspection.ID+"/attachments", csrf, developerSession, "probe.log", "probe output")
	if developerUpload.Code != http.StatusForbidden {
		t.Fatalf("developer upload status = %d, want 403; body=%s", developerUpload.Code, developerUpload.Body.String())
	}

	adminUpload := multipartRequest(t, handler, http.MethodPost, "/api/inspections/"+inspection.ID+"/attachments", csrf, adminSession, "probe.log", "probe output")
	if adminUpload.Code != http.StatusCreated {
		t.Fatalf("admin upload status = %d, want 201; body=%s", adminUpload.Code, adminUpload.Body.String())
	}
	var attachment model.InspectionAttachment
	if err := json.Unmarshal(adminUpload.Body.Bytes(), &attachment); err != nil {
		t.Fatalf("decode attachment: %v", err)
	}
	if attachment.Uploader != "admin" || attachment.AssetID != asset.ID {
		t.Fatalf("attachment mismatch: %+v", attachment)
	}

	adminDownload := apiRequest(t, handler, http.MethodGet, "/api/inspection-attachments/"+attachment.ID+"/download", csrf, adminSession, nil)
	if adminDownload.Code != http.StatusOK || adminDownload.Body.String() != "probe output" {
		t.Fatalf("admin download status=%d body=%q", adminDownload.Code, adminDownload.Body.String())
	}
	developerDownload := apiRequest(t, handler, http.MethodGet, "/api/inspection-attachments/"+attachment.ID+"/download", csrf, developerSession, nil)
	if developerDownload.Code != http.StatusOK || developerDownload.Body.String() != "probe output" {
		t.Fatalf("developer download status=%d body=%q", developerDownload.Code, developerDownload.Body.String())
	}
}

func createAssetByAPI(t *testing.T, handler http.Handler, csrf string, session string, name string) model.Asset {
	t.Helper()
	response := apiRequest(t, handler, http.MethodPost, "/api/assets", csrf, session, map[string]any{
		"platform_code":      "manual",
		"platform_name":      "Manual",
		"cloud_account_name": "manual",
		"account_id":         "manual",
		"project_code":       "public",
		"category":           "manual",
		"resource_type":      "Manual Service",
		"region":             "global",
		"environment":        "dev",
		"name":               name,
		"endpoint":           "https://" + name + ".example.com",
		"owner":              "Ops",
		"status":             "active",
		"criticality":        "medium",
		"last_checked_at":    "2026-06-02",
		"tags":               []string{"manual"},
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("create asset %s status = %d, want 201; body=%s", name, response.Code, response.Body.String())
	}
	var asset model.Asset
	if err := json.Unmarshal(response.Body.Bytes(), &asset); err != nil {
		t.Fatalf("decode asset: %v", err)
	}
	return asset
}

func assertAssetIDs(t *testing.T, assets []model.Asset, want []string) {
	t.Helper()
	got := make([]string, 0, len(assets))
	for _, asset := range assets {
		got = append(got, asset.ID)
	}
	assertStringSet(t, got, want)
}

func fetchCSRFToken(t *testing.T, handler http.Handler) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET / status = %d", response.Code)
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == csrfCookieName {
			return cookie.Value
		}
	}
	t.Fatalf("csrf cookie not set")
	return ""
}

func loginForTest(t *testing.T, handler http.Handler, csrf string, username string, password string) string {
	t.Helper()
	response := apiRequest(t, handler, http.MethodPost, "/api/auth/login", csrf, "", map[string]any{
		"username": username,
		"password": password,
	})
	if response.Code != http.StatusOK {
		t.Fatalf("login %s status = %d, body=%s", username, response.Code, response.Body.String())
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie.Value
		}
	}
	t.Fatalf("session cookie not set for %s", username)
	return ""
}

func apiRequest(t *testing.T, handler http.Handler, method string, path string, csrf string, session string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if payload != nil {
		if err := json.NewEncoder(body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, body)
	request.Header.Set("Content-Type", "application/json")
	if csrf != "" {
		request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrf})
		request.Header.Set(csrfHeaderName, csrf)
	}
	if session != "" {
		request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session})
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func multipartRequest(t *testing.T, handler http.Handler, method string, path string, csrf string, session string, filename string, content string) *httptest.ResponseRecorder {
	t.Helper()
	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	request := httptest.NewRequest(method, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if csrf != "" {
		request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrf})
		request.Header.Set(csrfHeaderName, csrf)
	}
	if session != "" {
		request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session})
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertToolIDs(t *testing.T, tools []model.ToolAsset, want []string) {
	t.Helper()
	got := make([]string, 0, len(tools))
	for _, tool := range tools {
		got = append(got, tool.ID)
	}
	assertStringSet(t, got, want)
}

func assertStringSet(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", got, want)
	}
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("missing %q: got %v want %v", item, got, want)
		}
	}
}
