package app

import (
	"errors"
	"net/http"

	"github.com/9904099/opsledger/internal/model"
)

func (s *Server) handleCreateApprovalFlow(w http.ResponseWriter, r *http.Request) {
	var req model.ApprovalFlowUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := s.store.CreateApprovalFlow(r.Context(), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateApprovalFlow(w http.ResponseWriter, r *http.Request) {
	var req model.ApprovalFlowUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := s.store.UpdateApprovalFlow(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleCreateApproval(w http.ResponseWriter, r *http.Request) {
	var req model.ApprovalCreateRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	req.Requester = user.Username
	created, err := s.store.CreateApproval(r.Context(), req)
	if err != nil {
		s.audit(r.Context(), r, user, "approval.create", req.TargetType, req.TargetID, "", "failed", err.Error(), map[string]string{
			"request_type": req.RequestType,
			"environment":  req.Environment,
		})
		writeStoreError(w, err)
		return
	}
	s.audit(r.Context(), r, user, "approval.create", created.TargetType, created.TargetID, created.TargetName, "success", "提交审批申请", map[string]string{
		"approval_id":  created.ID,
		"request_type": created.RequestType,
		"environment":  created.Environment,
		"status":       created.Status,
	})
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDecideApproval(w http.ResponseWriter, r *http.Request) {
	var req model.ApprovalDecisionRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	req.Approver = user.Username
	updated, err := s.store.DecideApproval(r.Context(), r.PathValue("id"), req, user)
	if err != nil {
		s.audit(r.Context(), r, user, "approval.decide", "approval", r.PathValue("id"), "", "failed", err.Error(), map[string]string{"decision": req.Status})
		writeStoreError(w, err)
		return
	}
	s.audit(r.Context(), r, user, "approval.decide", updated.TargetType, updated.TargetID, updated.TargetName, "success", "处理审批申请", map[string]string{
		"approval_id":  updated.ID,
		"request_type": updated.RequestType,
		"decision":     req.Status,
		"status":       updated.Status,
	})
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListAuditEvents(r.Context(), 100)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func approvalHasPendingTaskForRole(approval model.ApprovalRequest, role string) bool {
	for _, task := range approval.Tasks {
		if task.Status == "pending" && task.ApproverRole == role {
			return true
		}
	}
	return false
}
