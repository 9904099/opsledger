package app

import (
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func filterDashboardForUser(data model.DashboardData, user model.AppUser) model.DashboardData {
	if user.Role == "admin" || user.Role == "ops" {
		return data
	}

	now := time.Now().Format(time.RFC3339)
	grants := activeGrantsForUser(data.AccessGrants, user, now)
	grantedAssetIDs := grantedAssetIDs(grants)
	assetByID := map[string]model.Asset{}
	for _, asset := range data.Assets {
		assetByID[asset.ID] = asset
	}

	allowedAssetIDs := map[string]bool{}

	assets := make([]model.Asset, 0, len(data.Assets))
	for _, asset := range data.Assets {
		if assetVisibleForUser(asset, user, grantedAssetIDs) {
			assets = append(assets, asset)
			allowedAssetIDs[asset.ID] = true
		}
	}
	data.Assets = assets

	tools := make([]model.ToolAsset, 0, len(data.Tools))
	for _, tool := range data.Tools {
		if toolVisibleForUser(tool, user, assetByID, grantedAssetIDs) {
			tools = append(tools, tool)
			allowedAssetIDs[tool.AssetID] = true
		}
	}
	data.Tools = tools

	credentials := make([]model.CredentialItem, 0, len(data.Credentials))
	for _, credential := range data.Credentials {
		if credentialVisibleForUser(credential, user, grants, allowedAssetIDs) {
			credentials = append(credentials, credential)
		}
	}
	data.Credentials = credentials

	approvals := make([]model.ApprovalRequest, 0, len(data.Approvals))
	for _, approval := range data.Approvals {
		if user.Role == "auditor" || approval.Requester == user.Username || approvalHasPendingTaskForRole(approval, user.Role) {
			approvals = append(approvals, approval)
		}
	}
	data.Approvals = approvals

	changes := make([]model.ChangeRecord, 0, len(data.Changes))
	for _, change := range data.Changes {
		if allowedAssetIDs[change.AssetID] {
			changes = append(changes, change)
		}
	}
	data.Changes = changes

	inspections := make([]model.InspectionRecord, 0, len(data.Inspections))
	for _, inspection := range data.Inspections {
		if allowedAssetIDs[inspection.AssetID] {
			inspections = append(inspections, inspection)
		}
	}
	data.Inspections = inspections

	attachments := make([]model.InspectionAttachment, 0, len(data.Attachments))
	for _, attachment := range data.Attachments {
		if allowedAssetIDs[attachment.AssetID] {
			attachments = append(attachments, attachment)
		}
	}
	data.Attachments = attachments

	probes := make([]model.ProbeRecord, 0, len(data.Probes))
	for _, probe := range data.Probes {
		if allowedAssetIDs[probe.AssetID] {
			probes = append(probes, probe)
		}
	}
	data.Probes = probes

	alerts := make([]model.AlertRecord, 0, len(data.Alerts))
	for _, alert := range data.Alerts {
		if allowedAssetIDs[alert.AssetID] {
			alerts = append(alerts, alert)
		}
	}
	data.Alerts = alerts

	data.Users = nil
	data.Roles = nil
	data.Permissions = nil
	data.ApprovalFlows = nil
	data.CloudAccounts = nil
	data.CostRecords = nil
	data.RecentSyncs = nil
	if user.Role != "auditor" {
		data.AuditEvents = nil
	}
	data.AccessGrants = grants
	data.Summary = summarizeDashboard(data)
	return data
}

func activeGrantsForUser(items []model.AccessGrant, user model.AppUser, now string) []model.AccessGrant {
	grants := make([]model.AccessGrant, 0, len(items))
	for _, grant := range items {
		if grant.Username == user.Username && grant.Status == "active" && grant.ExpiresAt > now {
			grants = append(grants, grant)
		}
	}
	return grants
}

func grantedAssetIDs(grants []model.AccessGrant) map[string]bool {
	ids := map[string]bool{}
	for _, grant := range grants {
		if grant.TargetType == "asset" && grant.TargetID != "" {
			ids[grant.TargetID] = true
		}
	}
	return ids
}

func assetVisibleForUser(asset model.Asset, user model.AppUser, grantedAssetIDs map[string]bool) bool {
	if user.Role == "auditor" {
		return true
	}
	if grantedAssetIDs[asset.ID] {
		return true
	}
	if !environmentVisibleForUser(asset.Environment, user) {
		return false
	}
	return projectVisibleForUser(asset.ProjectCode, user)
}

func toolVisibleForUser(tool model.ToolAsset, user model.AppUser, assetByID map[string]model.Asset, grantedAssetIDs map[string]bool) bool {
	if user.Role == "auditor" {
		return true
	}
	if grantedAssetIDs[tool.AssetID] {
		return true
	}
	if tool.Environment == "global" {
		return true
	}
	if asset, ok := assetByID[tool.AssetID]; ok && !projectVisibleForUser(asset.ProjectCode, user) {
		return false
	}
	return environmentVisibleForUser(tool.Environment, user)
}

func credentialVisibleForUser(credential model.CredentialItem, user model.AppUser, grants []model.AccessGrant, allowedAssetIDs map[string]bool) bool {
	if user.Role == "auditor" {
		return false
	}
	for _, grant := range grants {
		if grant.Action != "credential" {
			continue
		}
		if grant.TargetType == "credential" && grant.TargetID == credential.ID {
			return true
		}
		if grant.TargetType == credential.OwnerType && grant.TargetID == credential.OwnerID {
			return true
		}
		if credential.OwnerType == "asset" && grant.TargetType == "tool" && grant.TargetID == credential.OwnerID {
			return true
		}
	}
	return credential.OwnerType == "asset" && allowedAssetIDs[credential.OwnerID] && credential.AccessPolicy == "viewable"
}

func environmentVisibleForUser(environment string, user model.AppUser) bool {
	if user.Role == "auditor" {
		return true
	}
	environment = strings.ToLower(strings.TrimSpace(environment))
	switch environment {
	case "", "global", "dev", "test", "staging", "stage", "pre", "preprod", "pre-prod", "local":
		return true
	default:
		return environment != "prod" && environment != "production"
	}
}

func projectVisibleForUser(projectCode string, user model.AppUser) bool {
	if user.Role == "auditor" {
		return true
	}
	projectCode = normalizeProjectScope(projectCode)
	if projectCode == "" {
		projectCode = "public"
	}
	for _, allowed := range projectScopeForUser(user) {
		if projectCode == allowed {
			return true
		}
	}
	return false
}

func projectScopeForUser(user model.AppUser) []string {
	role := strings.ToLower(strings.TrimSpace(user.Role))
	if role != "developer" && role != "lead" && role != "viewer" {
		return []string{"public"}
	}
	return []string{"public"}
}

func normalizeProjectScope(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "", "none":
		return ""
	case "common", "global", "shared", "public-resource", "公共", "公共资源":
		return "public"
	case "ent":
		return "enterprise"
	case "proxmox":
		return "pve"
	default:
		return value
	}
}

func summarizeDashboard(data model.DashboardData) model.Summary {
	summary := model.Summary{
		AssetsByPlatform:   map[string]int{},
		AssetsByCategory:   map[string]int{},
		AssetsByEnv:        map[string]int{},
		ChangesByRiskLevel: map[string]int{},
	}
	summary.TotalAssets = len(data.Assets)
	summary.ToolAssets = len(data.Tools)
	for _, asset := range data.Assets {
		summary.AssetsByPlatform[asset.PlatformName]++
		summary.AssetsByCategory[asset.Category]++
		summary.AssetsByEnv[asset.Environment]++
		switch asset.Status {
		case "active":
			summary.ActiveAssets++
		case "maintenance":
			summary.MaintenanceAssets++
		}
		if asset.Criticality == "high" {
			summary.CriticalAssets++
		}
	}
	latestProbe := map[string]model.ProbeRecord{}
	for _, probe := range data.Probes {
		current, ok := latestProbe[probe.AssetID]
		if !ok || probe.CheckedAt > current.CheckedAt {
			latestProbe[probe.AssetID] = probe
		}
	}
	for _, probe := range latestProbe {
		if probe.Status != "up" {
			summary.ProbeAlerts++
		}
	}
	for _, alert := range data.Alerts {
		if alert.Status == "open" || alert.Status == "acknowledged" {
			summary.OpenAlerts++
		}
	}
	for _, approval := range data.Approvals {
		if approval.Status == "pending" {
			summary.PendingApprovals++
		}
	}
	for _, change := range data.Changes {
		summary.ChangesByRiskLevel[change.RiskLevel]++
		switch change.Status {
		case "planned":
			summary.PlannedChanges++
		case "in_progress":
			summary.InProgressChanges++
		case "completed":
			summary.CompletedChanges++
		}
	}
	return summary
}
