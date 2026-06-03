package app

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func (s *Server) handleCreateAsset(w http.ResponseWriter, r *http.Request) {
	var asset model.Asset
	if err := decodeJSON(r.Body, &asset); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := s.store.CreateAsset(r.Context(), asset)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleExportAssets(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.DashboardData(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="opsledger-assets-%s.json"`, time.Now().Format("20060102-150405")))
	writeJSON(w, http.StatusOK, model.AssetImportRequest{Assets: data.Assets})
}

func (s *Server) handleImportAssets(w http.ResponseWriter, r *http.Request) {
	var req model.AssetImportRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Assets) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("assets is required"))
		return
	}
	result := model.AssetImportResult{
		CreatedIDs: []string{},
		Warnings:   []string{},
	}
	for index, asset := range req.Assets {
		asset.ID = ""
		created, err := s.store.CreateAsset(r.Context(), asset)
		if err != nil {
			result.SkippedAssets++
			result.Warnings = append(result.Warnings, fmt.Sprintf("assets[%d] %s", index, err.Error()))
			continue
		}
		result.ImportedAssets++
		result.CreatedIDs = append(result.CreatedIDs, created.ID)
	}
	status := http.StatusCreated
	if result.ImportedAssets == 0 {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, result)
}

func (s *Server) handleUpdateAsset(w http.ResponseWriter, r *http.Request) {
	var asset model.Asset
	if err := decodeJSON(r.Body, &asset); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := s.store.UpdateAsset(r.Context(), r.PathValue("id"), asset)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleBulkUpdateAssets(w http.ResponseWriter, r *http.Request) {
	var req model.AssetBulkUpdateRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.AssetIDs) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("asset_ids is required"))
		return
	}
	if !assetBulkUpdateHasPatch(req) {
		writeError(w, http.StatusBadRequest, errors.New("at least one update field is required"))
		return
	}

	result := model.AssetBulkUpdateResult{
		UpdatedIDs: []string{},
		Warnings:   []string{},
	}
	for index, id := range req.AssetIDs {
		asset, err := s.store.GetAsset(r.Context(), id)
		if err != nil {
			result.SkippedAssets++
			result.Warnings = append(result.Warnings, fmt.Sprintf("asset_ids[%d] %s", index, err.Error()))
			continue
		}
		applyAssetBulkPatch(&asset, req)
		updated, err := s.store.UpdateAsset(r.Context(), id, asset)
		if err != nil {
			result.SkippedAssets++
			result.Warnings = append(result.Warnings, fmt.Sprintf("asset_ids[%d] %s", index, err.Error()))
			continue
		}
		result.UpdatedAssets++
		result.UpdatedIDs = append(result.UpdatedIDs, updated.ID)
	}
	status := http.StatusOK
	if result.UpdatedAssets == 0 {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, result)
}

func assetBulkUpdateHasPatch(req model.AssetBulkUpdateRequest) bool {
	return req.ProjectCode != nil ||
		req.Category != nil ||
		req.ResourceType != nil ||
		req.Region != nil ||
		req.Environment != nil ||
		req.Owner != nil ||
		req.Status != nil ||
		req.Criticality != nil ||
		req.Tags != nil ||
		req.Notes != nil
}

func applyAssetBulkPatch(asset *model.Asset, req model.AssetBulkUpdateRequest) {
	if req.ProjectCode != nil {
		asset.ProjectCode = *req.ProjectCode
	}
	if req.Category != nil {
		asset.Category = *req.Category
	}
	if req.ResourceType != nil {
		asset.ResourceType = *req.ResourceType
	}
	if req.Region != nil {
		asset.Region = *req.Region
	}
	if req.Environment != nil {
		asset.Environment = *req.Environment
	}
	if req.Owner != nil {
		asset.Owner = *req.Owner
	}
	if req.Status != nil {
		asset.Status = *req.Status
	}
	if req.Criticality != nil {
		asset.Criticality = *req.Criticality
	}
	if req.Tags != nil {
		asset.Tags = *req.Tags
	}
	if req.Notes != nil {
		asset.Notes = *req.Notes
	}
}

func (s *Server) handleDeleteAsset(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteAsset(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
