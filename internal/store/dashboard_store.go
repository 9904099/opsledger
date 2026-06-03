package store

import (
	"context"
	"github.com/9904099/opsledger/internal/model"
	"time"
)

func (s *DBStore) DashboardData(ctx context.Context) (model.DashboardData, error) {
	platforms, err := s.ListPlatforms(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	cloudAccounts, err := s.ListCloudAccounts(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	costRecords, err := s.ListCloudAccountCostRecords(ctx, 2000)
	if err != nil {
		return model.DashboardData{}, err
	}

	environments, err := s.ListEnvironments(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	tools, err := s.ListTools(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	users, err := s.ListUsers(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	roles, err := s.ListRoles(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	permissions, err := s.ListPermissions(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	approvalFlows, err := s.ListApprovalFlows(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	approvals, err := s.ListApprovals(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	accessGrants, err := s.ListAccessGrants(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	credentials, err := s.ListCredentials(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	assets, err := s.listAssets(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	changes, err := s.listChanges(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	inspections, err := s.listRecentInspections(ctx, 500)
	if err != nil {
		return model.DashboardData{}, err
	}

	attachments, err := s.ListInspectionAttachments(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	probes, err := s.listProbes(ctx, 500)
	if err != nil {
		return model.DashboardData{}, err
	}

	alerts, err := s.ListAlerts(ctx)
	if err != nil {
		return model.DashboardData{}, err
	}

	recentSyncs, err := s.listRecentSyncs(ctx, 10)
	if err != nil {
		return model.DashboardData{}, err
	}

	auditEvents, err := s.ListAuditEvents(ctx, 50)
	if err != nil {
		return model.DashboardData{}, err
	}

	return model.DashboardData{
		Platforms:     platforms,
		CloudAccounts: cloudAccounts,
		CostRecords:   costRecords,
		Assets:        assets,
		Environments:  environments,
		Tools:         tools,
		Users:         users,
		Roles:         roles,
		Permissions:   permissions,
		ApprovalFlows: approvalFlows,
		Approvals:     approvals,
		AccessGrants:  accessGrants,
		Credentials:   credentials,
		Changes:       changes,
		Inspections:   inspections,
		Attachments:   attachments,
		Probes:        probes,
		Alerts:        alerts,
		RecentSyncs:   recentSyncs,
		AuditEvents:   auditEvents,
		Summary:       summarize(assets, changes, probes, alerts, tools, approvals),
		GeneratedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

func (s *DBStore) ListPlatforms(ctx context.Context) ([]model.Platform, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, name, description, enabled, created_at, updated_at
		FROM platforms
		ORDER BY code ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Platform
	for rows.Next() {
		var item model.Platform
		var enabled int
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func summarize(assets []model.Asset, changes []model.ChangeRecord, probes []model.ProbeRecord, alerts []model.AlertRecord, tools []model.ToolAsset, approvals []model.ApprovalRequest) model.Summary {
	summary := model.Summary{
		AssetsByPlatform:   map[string]int{},
		AssetsByCategory:   map[string]int{},
		AssetsByEnv:        map[string]int{},
		ChangesByRiskLevel: map[string]int{},
	}

	summary.TotalAssets = len(assets)
	summary.ToolAssets = len(tools)
	for _, asset := range assets {
		summary.AssetsByPlatform[asset.PlatformName]++
		summary.AssetsByCategory[asset.Category]++
		summary.AssetsByEnv[asset.Environment]++
		if asset.Status == "active" {
			summary.ActiveAssets++
		}
		if asset.Status == "maintenance" {
			summary.MaintenanceAssets++
		}
		if asset.Criticality == "high" {
			summary.CriticalAssets++
		}
	}

	for _, change := range changes {
		summary.ChangesByRiskLevel[change.RiskLevel]++
		switch change.Status {
		case "planned":
			summary.PlannedChanges++
		case "in_progress":
			summary.InProgressChanges++
		case "done":
			summary.CompletedChanges++
		}
	}

	for _, approval := range approvals {
		if approval.Status == "pending" {
			summary.PendingApprovals++
		}
	}

	latestProbes := map[string]model.ProbeRecord{}
	for _, probe := range probes {
		if existing, ok := latestProbes[probe.AssetID]; !ok || probe.CheckedAt > existing.CheckedAt {
			latestProbes[probe.AssetID] = probe
		}
	}
	for _, probe := range latestProbes {
		if probe.Status != "up" {
			summary.ProbeAlerts++
		}
	}
	for _, alert := range alerts {
		if alert.Status == "open" || alert.Status == "acknowledged" {
			summary.OpenAlerts++
		}
	}

	return summary
}
