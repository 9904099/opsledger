package app

import (
	"embed"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

//go:embed assets/*
var assetsFS embed.FS

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /webssh/session/{session}", s.handleWebSSHSessionPage)
	mux.Handle("GET /webssh/ws/{session}", websocket.Handler(s.handleWebSSHWebSocket))
	mux.Handle("GET /static/", http.StripPrefix("/static/", s.static))
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/setup", s.handleSetupStatus)
	mux.HandleFunc("POST /api/setup", s.handleSetupInitialAdmin)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleCurrentUser)
	mux.HandleFunc("GET /api/bootstrap", s.handleBootstrap)
	mux.HandleFunc("GET /api/platforms", s.handleListPlatforms)
	mux.HandleFunc("GET /api/cloud-accounts", s.handleListCloudAccounts)
	mux.HandleFunc("POST /api/cloud-accounts", s.handleCreateCloudAccount)
	mux.HandleFunc("PUT /api/cloud-accounts/{id}", s.handleUpdateCloudAccount)
	mux.HandleFunc("POST /api/cloud-accounts/sync", s.handleCloudAccountSync)
	mux.HandleFunc("POST /api/cloud-accounts/{id}/cost-sync", s.handleCloudAccountCostSync)
	mux.HandleFunc("POST /api/tools", s.handleCreateTool)
	mux.HandleFunc("PUT /api/tools/{id}", s.handleUpdateTool)
	mux.HandleFunc("POST /api/users", s.handleCreateUser)
	mux.HandleFunc("PUT /api/users/{id}", s.handleUpdateUser)
	mux.HandleFunc("POST /api/roles", s.handleCreateRole)
	mux.HandleFunc("PUT /api/roles/{id}", s.handleUpdateRole)
	mux.HandleFunc("POST /api/permissions", s.handleCreatePermission)
	mux.HandleFunc("PUT /api/permissions/{id}", s.handleUpdatePermission)
	mux.HandleFunc("DELETE /api/permissions/{id}", s.handleDeletePermission)
	mux.HandleFunc("POST /api/approval-flows", s.handleCreateApprovalFlow)
	mux.HandleFunc("PUT /api/approval-flows/{id}", s.handleUpdateApprovalFlow)
	mux.HandleFunc("POST /api/approvals", s.handleCreateApproval)
	mux.HandleFunc("POST /api/approvals/{id}/decision", s.handleDecideApproval)
	mux.HandleFunc("GET /api/audit-events", s.handleListAuditEvents)
	mux.HandleFunc("GET /api/credentials", s.handleListCredentials)
	mux.HandleFunc("POST /api/credentials", s.handleUpsertCredential)
	mux.HandleFunc("POST /api/credentials/{id}/reveal", s.handleRevealCredential)
	mux.HandleFunc("POST /api/credentials/{id}/copy", s.handleCopyCredential)
	mux.HandleFunc("POST /api/webssh/open", s.handleOpenWebSSH)
	mux.HandleFunc("GET /api/assets/export", s.handleExportAssets)
	mux.HandleFunc("POST /api/assets/import", s.handleImportAssets)
	mux.HandleFunc("POST /api/assets/bulk-update", s.handleBulkUpdateAssets)
	mux.HandleFunc("POST /api/assets", s.handleCreateAsset)
	mux.HandleFunc("PUT /api/assets/{id}", s.handleUpdateAsset)
	mux.HandleFunc("DELETE /api/assets/{id}", s.handleDeleteAsset)
	mux.HandleFunc("POST /api/assets/{id}/probe", s.handleProbeAsset)
	mux.HandleFunc("POST /api/changes", s.handleCreateChange)
	mux.HandleFunc("PUT /api/changes/{id}", s.handleUpdateChange)
	mux.HandleFunc("DELETE /api/changes/{id}", s.handleDeleteChange)
	mux.HandleFunc("POST /api/inspections", s.handleCreateInspection)
	mux.HandleFunc("POST /api/inspections/{id}/attachments", s.handleCreateInspectionAttachment)
	mux.HandleFunc("GET /api/inspection-attachments/{id}/download", s.handleDownloadInspectionAttachment)
	mux.HandleFunc("POST /api/alerts/{id}/resolve", s.handleResolveAlert)
	return requestLogger(s.authMiddleware(mux))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ensureCSRFCookie(w, r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(s.index)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.DashboardData(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if user, ok := currentUser(r.Context()); ok {
		data = filterDashboardForUser(data, user)
	}
	writeJSON(w, http.StatusOK, data)
}
