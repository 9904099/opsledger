package app

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/9904099/opsledger/internal/model"
	"github.com/9904099/opsledger/internal/store"
)

const maxInspectionAttachmentUploadSize = 10 << 20

func (s *Server) handleProbeAsset(w http.ResponseWriter, r *http.Request) {
	asset, err := s.store.GetAsset(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}

	probe, err := probeHTTP(r.Context(), asset)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	record, err := s.store.CreateProbe(r.Context(), probe)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.reconcileProbeAlert(r.Context(), asset, record, "manual-probe")
	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) handleResolveAlert(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	var req model.AlertResolveRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.Resolver = user.Username
	alert, err := s.store.ResolveAlert(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.audit(r.Context(), r, user, "alert.resolve", "alert", alert.ID, alert.Title, "success", "告警已处理", map[string]string{
		"asset_id": alert.AssetID,
		"source":   alert.Source,
		"severity": alert.Severity,
	})
	writeJSON(w, http.StatusOK, alert)
}

func (s *Server) handleCreateInspection(w http.ResponseWriter, r *http.Request) {
	var req model.InspectionRecordCreateRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	record, err := s.store.CreateInspection(r.Context(), model.InspectionRecord{
		AssetID:   req.AssetID,
		Executor:  req.Executor,
		Result:    req.Result,
		Summary:   req.Summary,
		CheckedAt: req.CheckedAt,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) handleCreateInspectionAttachment(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseMultipartForm(maxInspectionAttachmentUploadSize + (1 << 20)); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	limited := io.LimitReader(file, maxInspectionAttachmentUploadSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(data) > maxInspectionAttachmentUploadSize {
		writeError(w, http.StatusBadRequest, fmt.Errorf("attachment exceeds %d bytes", maxInspectionAttachmentUploadSize))
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(header.Filename)))
	}
	if contentType == "" && len(data) > 0 {
		contentType = http.DetectContentType(data)
	}
	attachment, err := s.store.CreateInspectionAttachment(r.Context(), model.InspectionAttachmentCreateRequest{
		InspectionID: r.PathValue("id"),
		FileName:     header.Filename,
		ContentType:  contentType,
		Data:         data,
		Uploader:     user.Username,
		Description:  r.FormValue("description"),
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.audit(r.Context(), r, user, "inspection.attachment.upload", "inspection", attachment.InspectionID, attachment.FileName, "success", "上传巡检附件", map[string]string{
		"asset_id": attachment.AssetID,
		"size":     fmt.Sprintf("%d", attachment.SizeBytes),
	})
	writeJSON(w, http.StatusCreated, attachment)
}

func (s *Server) handleDownloadInspectionAttachment(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	attachment, data, err := s.store.GetInspectionAttachment(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !canAccessAttachment(r.Context(), s.store, user, attachment) {
		writeError(w, http.StatusForbidden, store.ErrForbidden)
		return
	}

	filename := filepath.Base(attachment.FileName)
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		filename = "inspection-attachment"
	}
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func canAccessAttachment(ctx context.Context, st store.Store, user model.AppUser, attachment model.InspectionAttachment) bool {
	if user.Role == "admin" || user.Role == "ops" || user.Role == "auditor" {
		return true
	}
	data, err := st.DashboardData(ctx)
	if err != nil {
		return false
	}
	filtered := filterDashboardForUser(data, user)
	for _, asset := range filtered.Assets {
		if asset.ID == attachment.AssetID {
			return true
		}
	}
	return false
}
