package app

import (
	"errors"
	"net/http"

	"github.com/9904099/opsledger/internal/model"
)

func (s *Server) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListCredentials(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleUpsertCredential(w http.ResponseWriter, r *http.Request) {
	var req model.CredentialUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	item, err := s.store.UpsertCredential(r.Context(), req)
	if err != nil {
		s.audit(r.Context(), r, user, "credential.upsert", req.OwnerType, req.OwnerID, "", "failed", err.Error(), map[string]string{"kind": req.Kind, "key_name": req.KeyName})
		writeStoreError(w, err)
		return
	}
	s.audit(r.Context(), r, user, "credential.upsert", item.OwnerType, item.OwnerID, item.OwnerName, "success", "保存凭证项", map[string]string{
		"credential_id": item.ID,
		"kind":          item.Kind,
		"key_name":      item.KeyName,
	})
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleRevealCredential(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	result, err := s.store.RevealCredential(r.Context(), r.PathValue("id"), user)
	if err != nil {
		s.audit(r.Context(), r, user, "credential.reveal", "credential", r.PathValue("id"), "", "failed", err.Error(), nil)
		writeStoreError(w, err)
		return
	}
	s.audit(r.Context(), r, user, "credential.reveal", result.Credential.OwnerType, result.Credential.OwnerID, result.Credential.OwnerName, "success", "查看凭证明文", map[string]string{
		"credential_id": result.Credential.ID,
		"kind":          result.Credential.Kind,
		"key_name":      result.Credential.KeyName,
	})
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCopyCredential(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	item, err := s.store.RecordCredentialCopy(r.Context(), r.PathValue("id"), user)
	if err != nil {
		s.audit(r.Context(), r, user, "credential.copy", "credential", r.PathValue("id"), "", "failed", err.Error(), nil)
		writeStoreError(w, err)
		return
	}
	s.audit(r.Context(), r, user, "credential.copy", item.OwnerType, item.OwnerID, item.OwnerName, "success", "复制凭证项", map[string]string{
		"credential_id": item.ID,
		"kind":          item.Kind,
		"key_name":      item.KeyName,
	})
	writeJSON(w, http.StatusOK, item)
}
