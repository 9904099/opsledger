package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/9904099/opsledger/internal/model"
)

func (s *Server) handleListCloudAccounts(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListCloudAccounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateCloudAccount(w http.ResponseWriter, r *http.Request) {
	var req model.CloudAccountUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	item, err := s.store.CreateCloudAccount(r.Context(), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateCloudAccount(w http.ResponseWriter, r *http.Request) {
	var req model.CloudAccountUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	item, err := s.store.UpdateCloudAccount(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleCloudAccountSync(w http.ResponseWriter, r *http.Request) {
	var req model.CloudAccountSyncRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	account, err := s.store.GetCloudAccount(r.Context(), req.CloudAccountID)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	result, err := s.syncCloudAccount(r.Context(), account, req)
	user, _ := currentUser(r.Context())
	if err != nil {
		s.audit(r.Context(), r, user, "cloud_account.sync", "cloud_account", account.ID, account.Name, "failed", err.Error(), map[string]string{"platform": account.PlatformCode})
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.audit(r.Context(), r, user, "cloud_account.sync", "cloud_account", account.ID, account.Name, "success", formatCloudAccountSyncSummary(result), map[string]string{
		"platform":          account.PlatformCode,
		"discovered_assets": fmt.Sprintf("%d", result.DiscoveredAssets),
		"created_assets":    fmt.Sprintf("%d", result.CreatedAssets),
		"updated_assets":    fmt.Sprintf("%d", result.UpdatedAssets),
		"stale_assets":      fmt.Sprintf("%d", result.StaleAssets),
	})
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) syncCloudAccount(ctx context.Context, account model.CloudAccount, req model.CloudAccountSyncRequest) (model.CloudAccountSyncResult, error) {
	if strings.TrimSpace(req.CloudAccountID) == "" {
		req.CloudAccountID = account.ID
	}
	switch account.PlatformCode {
	case "aws":
		return s.awsImporter.SyncCloudAccount(ctx, req)
	case "cloudflare":
		return s.cloudflareImporter.SyncCloudAccount(ctx, req)
	case "pve":
		return s.pveImporter.SyncCloudAccount(ctx, req)
	default:
		return model.CloudAccountSyncResult{}, errors.New("unsupported cloud platform: " + account.PlatformName)
	}
}

func formatCloudAccountSyncSummary(result model.CloudAccountSyncResult) string {
	return fmt.Sprintf("同步完成：发现 %d 条，新增 %d 条，更新 %d 条，标记 stale %d 条", result.DiscoveredAssets, result.CreatedAssets, result.UpdatedAssets, result.StaleAssets)
}

func (s *Server) handleCloudAccountCostSync(w http.ResponseWriter, r *http.Request) {
	account, err := s.store.GetCloudAccount(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}

	var result model.CloudAccountCostResult
	switch account.PlatformCode {
	case "aws":
		result, err = s.awsImporter.SyncCloudAccountCost(r.Context(), account.ID)
	case "cloudflare":
		err = errors.New("Cloudflare 当前未接入费用 API")
	default:
		err = errors.New("unsupported cloud platform: " + account.PlatformName)
	}
	user, _ := currentUser(r.Context())
	if err != nil {
		s.audit(r.Context(), r, user, "cloud_account.cost_sync", "cloud_account", account.ID, account.Name, "failed", err.Error(), map[string]string{"platform": account.PlatformCode})
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.audit(r.Context(), r, user, "cloud_account.cost_sync", "cloud_account", account.ID, account.Name, "success", result.Summary, map[string]string{
		"platform":               account.PlatformCode,
		"current_month_cost":     result.CurrentMonthCost,
		"forecast_month_cost":    result.ForecastMonthCost,
		"month_over_month_delta": result.MonthOverMonthDelta,
	})
	writeJSON(w, http.StatusOK, result)
}
