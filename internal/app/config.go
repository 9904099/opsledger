package app

import (
	"net/http"

	"github.com/9904099/opsledger/internal/model"
)

func (s *Server) handleListPlatforms(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListPlatforms(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateTool(w http.ResponseWriter, r *http.Request) {
	var req model.ToolAssetUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := s.store.CreateTool(r.Context(), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateTool(w http.ResponseWriter, r *http.Request) {
	var req model.ToolAssetUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := s.store.UpdateTool(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req model.AppUserUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := s.store.CreateUser(r.Context(), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	var req model.AppUserUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := s.store.UpdateUser(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var req model.RoleDefinitionUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := s.store.CreateRole(r.Context(), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateRole(w http.ResponseWriter, r *http.Request) {
	var req model.RoleDefinitionUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := s.store.UpdateRole(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleCreatePermission(w http.ResponseWriter, r *http.Request) {
	var req model.RolePermissionUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	created, err := s.store.CreatePermission(r.Context(), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdatePermission(w http.ResponseWriter, r *http.Request) {
	var req model.RolePermissionUpsertRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := s.store.UpdatePermission(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeletePermission(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeletePermission(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (s *Server) handleCreateChange(w http.ResponseWriter, r *http.Request) {
	var change model.ChangeRecord
	if err := decodeJSON(r.Body, &change); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := s.store.CreateChange(r.Context(), change)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateChange(w http.ResponseWriter, r *http.Request) {
	var change model.ChangeRecord
	if err := decodeJSON(r.Body, &change); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := s.store.UpdateChange(r.Context(), r.PathValue("id"), change)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteChange(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteChange(r.Context(), r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
