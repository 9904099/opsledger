package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/9904099/opsledger/internal/model"
)

func (s *DBStore) ListEnvironments(ctx context.Context) ([]model.Environment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, name, tier, owner, description, status, created_at, updated_at
		FROM environments
		ORDER BY CASE code WHEN 'dev' THEN 1 WHEN 'prod' THEN 2 WHEN 'test' THEN 3 WHEN 'staging' THEN 4 WHEN 'local' THEN 5 ELSE 9 END, code ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Environment{}
	for rows.Next() {
		var item model.Environment
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Tier, &item.Owner, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) ListTools(ctx context.Context) ([]model.ToolAsset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.asset_id, t.environment, t.tool_type, t.login_policy, t.credential_policy,
		       t.approval_required, t.webssh_enabled, t.description, t.created_at, t.updated_at,
		       a.name, a.endpoint, a.owner, a.status, a.criticality, a.tags_csv
		FROM tool_assets t
		JOIN assets a ON a.id = t.asset_id
		ORDER BY t.environment ASC, t.tool_type ASC, a.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ToolAsset{}
	for rows.Next() {
		item, err := scanTool(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) CreateTool(ctx context.Context, req model.ToolAssetUpsertRequest) (model.ToolAsset, error) {
	req = normalizeToolRequest(req)
	if err := validateToolRequest(req); err != nil {
		return model.ToolAsset{}, err
	}
	now := time.Now().Format(time.RFC3339)
	asset := model.Asset{
		PlatformCode:  "tool",
		PlatformName:  "Tool",
		Category:      "tool",
		ResourceType:  req.ToolType,
		Environment:   req.Environment,
		Name:          req.Name,
		Endpoint:      req.Endpoint,
		Owner:         req.Owner,
		Status:        req.Status,
		Criticality:   req.Criticality,
		LastCheckedAt: time.Now().Format("2006-01-02"),
		Tags:          req.Tags,
		Notes:         req.Description,
		Specs:         map[string]string{"tool_type": req.ToolType, "login_policy": req.LoginPolicy, "credential_policy": req.CredentialPolicy},
		Source:        "manual-tool",
		ExternalID:    strings.ToLower(req.Environment + ":" + req.ToolType + ":" + req.Name),
	}
	createdAsset, err := s.CreateAsset(ctx, asset)
	if err != nil {
		return model.ToolAsset{}, err
	}
	id := newID("tool")
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO tool_assets (
			id, asset_id, environment, tool_type, login_policy, credential_policy,
			approval_required, webssh_enabled, description, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, createdAsset.ID, req.Environment, req.ToolType, req.LoginPolicy, req.CredentialPolicy,
		boolToInt(req.ApprovalRequired), boolToInt(req.WebSSHEnabled), req.Description, now, now)
	if err != nil {
		return model.ToolAsset{}, err
	}
	return s.getTool(ctx, id)
}

func (s *DBStore) UpdateTool(ctx context.Context, id string, req model.ToolAssetUpsertRequest) (model.ToolAsset, error) {
	req = normalizeToolRequest(req)
	if err := validateToolRequest(req); err != nil {
		return model.ToolAsset{}, err
	}
	existing, err := s.getTool(ctx, id)
	if err != nil {
		return model.ToolAsset{}, err
	}
	asset, err := s.getAsset(ctx, existing.AssetID)
	if err != nil {
		return model.ToolAsset{}, err
	}
	asset.PlatformCode = "tool"
	asset.PlatformName = "Tool"
	asset.Category = "tool"
	asset.ResourceType = req.ToolType
	asset.Environment = req.Environment
	asset.Name = req.Name
	asset.Endpoint = req.Endpoint
	asset.Owner = req.Owner
	asset.Status = req.Status
	asset.Criticality = req.Criticality
	asset.Tags = req.Tags
	asset.Notes = req.Description
	asset.Specs = map[string]string{"tool_type": req.ToolType, "login_policy": req.LoginPolicy, "credential_policy": req.CredentialPolicy}
	if _, err := s.UpdateAsset(ctx, asset.ID, asset); err != nil {
		return model.ToolAsset{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE tool_assets
		SET environment = ?, tool_type = ?, login_policy = ?, credential_policy = ?,
			approval_required = ?, webssh_enabled = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, req.Environment, req.ToolType, req.LoginPolicy, req.CredentialPolicy,
		boolToInt(req.ApprovalRequired), boolToInt(req.WebSSHEnabled), req.Description, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return model.ToolAsset{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return model.ToolAsset{}, err
	}
	return s.getTool(ctx, id)
}

func (s *DBStore) ListUsers(ctx context.Context) ([]model.AppUser, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, display_name, email, phone, role, team, status, auth_source, external_subject,
		       password_changed_at, failed_login_count, locked_until, last_login_at, created_at, updated_at
		FROM app_users
		ORDER BY username ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.AppUser{}
	for rows.Next() {
		var item model.AppUser
		if err := rows.Scan(
			&item.ID, &item.Username, &item.DisplayName, &item.Email, &item.Phone, &item.Role, &item.Team,
			&item.Status, &item.AuthSource, &item.ExternalSubject, &item.PasswordChangedAt,
			&item.FailedLoginCount, &item.LockedUntil, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_users`).Scan(&count)
	return count, err
}

func (s *DBStore) CreateInitialAdmin(ctx context.Context, req model.SetupAdminRequest) (model.AppUser, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.ConfirmPassword = strings.TrimSpace(req.ConfirmPassword)
	switch {
	case req.Username == "":
		return model.AppUser{}, errors.New("username is required")
	case req.DisplayName == "":
		return model.AppUser{}, errors.New("display_name is required")
	case req.Password == "":
		return model.AppUser{}, errors.New("password is required")
	case len(req.Password) < 12:
		return model.AppUser{}, errors.New("password must be at least 12 characters")
	case req.Password != req.ConfirmPassword:
		return model.AppUser{}, errors.New("password confirmation does not match")
	}

	var created model.AppUser
	if err := s.withTx(ctx, func(tx *dialectTx) error {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_users`).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("setup has already been completed")
		}
		now := time.Now().Format(time.RFC3339)
		hashed, err := hashPassword(req.Password)
		if err != nil {
			return err
		}
		created = model.AppUser{
			ID:                "usr-initial-admin",
			Username:          req.Username,
			DisplayName:       req.DisplayName,
			Email:             req.Email,
			Role:              "admin",
			Team:              "Platform",
			Status:            "active",
			AuthSource:        "local",
			PasswordChangedAt: now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO app_users (
				id, username, display_name, email, phone, role, team, status, auth_source, external_subject,
				password_hash, password_changed_at, failed_login_count, locked_until, last_login_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, '', 'admin', 'Platform', 'active', 'local', '', ?, ?, 0, '', '', ?, ?)
		`, created.ID, created.Username, created.DisplayName, created.Email, hashed, created.PasswordChangedAt, created.CreatedAt, created.UpdatedAt)
		return err
	}); err != nil {
		return model.AppUser{}, err
	}
	return created, nil
}

func (s *DBStore) CreateUser(ctx context.Context, req model.AppUserUpsertRequest) (model.AppUser, error) {
	req = normalizeUserRequest(req)
	if err := validateUserRequest(req); err != nil {
		return model.AppUser{}, err
	}
	now := time.Now().Format(time.RFC3339)
	id := newID("usr")
	passwordHash := ""
	passwordChangedAt := ""
	if req.Password != "" {
		hashed, err := hashPassword(req.Password)
		if err != nil {
			return model.AppUser{}, err
		}
		passwordHash = hashed
		passwordChangedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO app_users (
			id, username, display_name, email, phone, role, team, status, auth_source, external_subject,
			password_hash, password_changed_at, failed_login_count, locked_until, last_login_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'local', '', ?, ?, 0, '', '', ?, ?)
	`, id, req.Username, req.DisplayName, req.Email, req.Phone, req.Role, req.Team, req.Status, passwordHash, passwordChangedAt, now, now)
	if err != nil {
		return model.AppUser{}, err
	}
	return s.getUser(ctx, id)
}

func (s *DBStore) UpdateUser(ctx context.Context, id string, req model.AppUserUpsertRequest) (model.AppUser, error) {
	req = normalizeUserRequest(req)
	if err := validateUserRequest(req); err != nil {
		return model.AppUser{}, err
	}
	var result sql.Result
	var err error
	if req.Password != "" {
		hashed, hashErr := hashPassword(req.Password)
		if hashErr != nil {
			return model.AppUser{}, hashErr
		}
		result, err = s.db.ExecContext(ctx, `
			UPDATE app_users
			SET username = ?, display_name = ?, email = ?, phone = ?, role = ?, team = ?, status = ?,
				password_hash = ?, password_changed_at = ?, updated_at = ?
			WHERE id = ?
		`, req.Username, req.DisplayName, req.Email, req.Phone, req.Role, req.Team, req.Status, hashed, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339), id)
	} else {
		result, err = s.db.ExecContext(ctx, `
		UPDATE app_users
		SET username = ?, display_name = ?, email = ?, phone = ?, role = ?, team = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, req.Username, req.DisplayName, req.Email, req.Phone, req.Role, req.Team, req.Status, time.Now().Format(time.RFC3339), id)
	}
	if err != nil {
		return model.AppUser{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return model.AppUser{}, err
	}
	return s.getUser(ctx, id)
}

func (s *DBStore) AuthenticateUser(ctx context.Context, req model.LoginRequest, ip string, userAgent string) (model.AppUser, string, model.UserSession, error) {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return model.AppUser{}, "", model.UserSession{}, errors.New("username and password are required")
	}
	user, passwordHash, err := s.getUserWithPassword(ctx, req.Username)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.AppUser{}, "", model.UserSession{}, errors.New("invalid username or password")
		}
		return model.AppUser{}, "", model.UserSession{}, err
	}
	if user.Status != "active" {
		return model.AppUser{}, "", model.UserSession{}, errors.New("user is not active")
	}
	if user.LockedUntil != "" {
		lockedUntil, parseErr := time.Parse(time.RFC3339, user.LockedUntil)
		if parseErr == nil && lockedUntil.After(time.Now()) {
			return model.AppUser{}, "", model.UserSession{}, errors.New("user is locked")
		}
	}
	if passwordHash == "" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		_ = s.recordLoginFailure(ctx, user.ID, user.FailedLoginCount+1)
		return model.AppUser{}, "", model.UserSession{}, errors.New("invalid username or password")
	}

	token, err := randomToken(32)
	if err != nil {
		return model.AppUser{}, "", model.UserSession{}, err
	}
	now := time.Now()
	session := model.UserSession{
		ID:         newID("sess"),
		UserID:     user.ID,
		TokenHash:  hashSessionToken(token),
		ExpiresAt:  now.Add(8 * time.Hour).Format(time.RFC3339),
		LastSeenAt: now.Format(time.RFC3339),
		IP:         ip,
		UserAgent:  userAgent,
		CreatedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at, last_seen_at, revoked_at, ip, user_agent, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?)
	`, session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.LastSeenAt, session.IP, session.UserAgent, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return model.AppUser{}, "", model.UserSession{}, err
	}
	nowText := now.Format(time.RFC3339)
	_, _ = s.db.ExecContext(ctx, `UPDATE app_users SET failed_login_count = 0, locked_until = '', last_login_at = ?, updated_at = ? WHERE id = ?`, nowText, nowText, user.ID)
	user.FailedLoginCount = 0
	user.LockedUntil = ""
	user.LastLoginAt = nowText
	user.UpdatedAt = nowText
	return user, token, session, nil
}

func (s *DBStore) CurrentUserBySessionToken(ctx context.Context, token string) (model.AppUser, model.UserSession, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return model.AppUser{}, model.UserSession{}, os.ErrNotExist
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT us.id, us.user_id, us.token_hash, us.expires_at, us.last_seen_at, us.revoked_at,
		       us.ip, us.user_agent, us.created_at, us.updated_at,
		       u.id, u.username, u.display_name, u.email, u.phone, u.role, u.team, u.status, u.auth_source,
		       u.external_subject, u.password_changed_at, u.failed_login_count, u.locked_until,
		       u.last_login_at, u.created_at, u.updated_at
		FROM user_sessions us
		JOIN app_users u ON u.id = us.user_id
		WHERE us.token_hash = ?
	`, hashSessionToken(token))
	var session model.UserSession
	var user model.AppUser
	err := row.Scan(
		&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.LastSeenAt, &session.RevokedAt,
		&session.IP, &session.UserAgent, &session.CreatedAt, &session.UpdatedAt,
		&user.ID, &user.Username, &user.DisplayName, &user.Email, &user.Phone, &user.Role, &user.Team, &user.Status,
		&user.AuthSource, &user.ExternalSubject, &user.PasswordChangedAt, &user.FailedLoginCount, &user.LockedUntil,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AppUser{}, model.UserSession{}, os.ErrNotExist
		}
		return model.AppUser{}, model.UserSession{}, err
	}
	if user.Status != "active" || session.RevokedAt != "" {
		return model.AppUser{}, model.UserSession{}, os.ErrNotExist
	}
	expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil || expiresAt.Before(time.Now()) {
		return model.AppUser{}, model.UserSession{}, os.ErrNotExist
	}
	now := time.Now().Format(time.RFC3339)
	_, _ = s.db.ExecContext(ctx, `UPDATE user_sessions SET last_seen_at = ?, updated_at = ? WHERE id = ?`, now, now, session.ID)
	session.LastSeenAt = now
	session.UpdatedAt = now
	return user, session, nil
}

func (s *DBStore) RevokeSession(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE user_sessions SET revoked_at = ?, updated_at = ? WHERE token_hash = ?`, now, now, hashSessionToken(token))
	return err
}

func (s *DBStore) ListRoles(ctx context.Context) ([]model.RoleDefinition, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, name, description, level, status, created_at, updated_at
		FROM roles
		ORDER BY level ASC, code ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.RoleDefinition{}
	for rows.Next() {
		var item model.RoleDefinition
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Level, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) CreateRole(ctx context.Context, req model.RoleDefinitionUpsertRequest) (model.RoleDefinition, error) {
	req = normalizeRoleRequest(req)
	if err := validateRoleRequest(req); err != nil {
		return model.RoleDefinition{}, err
	}
	now := time.Now().Format(time.RFC3339)
	id := newID("role")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO roles (id, code, name, description, level, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.Code, req.Name, req.Description, req.Level, req.Status, now, now)
	if err != nil {
		return model.RoleDefinition{}, err
	}
	return s.getRole(ctx, id)
}

func (s *DBStore) UpdateRole(ctx context.Context, id string, req model.RoleDefinitionUpsertRequest) (model.RoleDefinition, error) {
	req = normalizeRoleRequest(req)
	if err := validateRoleRequest(req); err != nil {
		return model.RoleDefinition{}, err
	}
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		UPDATE roles
		SET code = ?, name = ?, description = ?, level = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, req.Code, req.Name, req.Description, req.Level, req.Status, now, id)
	if err != nil {
		return model.RoleDefinition{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return model.RoleDefinition{}, err
	}
	return s.getRole(ctx, id)
}

func (s *DBStore) getRole(ctx context.Context, id string) (model.RoleDefinition, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, code, name, description, level, status, created_at, updated_at
		FROM roles
		WHERE id = ?
	`, id)
	var item model.RoleDefinition
	if err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Level, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RoleDefinition{}, os.ErrNotExist
		}
		return model.RoleDefinition{}, err
	}
	return item, nil
}

func (s *DBStore) ListPermissionsForRole(ctx context.Context, role string) ([]model.RolePermission, error) {
	role = strings.TrimSpace(role)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, role, scope, action, environment, project_code, requires_approval, created_at, updated_at
		FROM role_permissions
		WHERE role = ? OR role = '*'
		ORDER BY role ASC, environment ASC, scope ASC, action ASC
	`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.RolePermission{}
	for rows.Next() {
		var item model.RolePermission
		var requiresApproval int
		if err := rows.Scan(&item.ID, &item.Role, &item.Scope, &item.Action, &item.Environment, &item.ProjectCode, &requiresApproval, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.RequiresApproval = requiresApproval == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) ListPermissions(ctx context.Context) ([]model.RolePermission, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, role, scope, action, environment, project_code, requires_approval, created_at, updated_at
		FROM role_permissions
		ORDER BY role ASC, environment ASC, scope ASC, action ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.RolePermission{}
	for rows.Next() {
		var item model.RolePermission
		var requiresApproval int
		if err := rows.Scan(&item.ID, &item.Role, &item.Scope, &item.Action, &item.Environment, &item.ProjectCode, &requiresApproval, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.RequiresApproval = requiresApproval == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) CreatePermission(ctx context.Context, req model.RolePermissionUpsertRequest) (model.RolePermission, error) {
	req = normalizePermissionRequest(req)
	if err := validatePermissionRequest(req); err != nil {
		return model.RolePermission{}, err
	}
	now := time.Now().Format(time.RFC3339)
	id := newID("perm")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO role_permissions (id, role, scope, action, environment, project_code, requires_approval, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.Role, req.Scope, req.Action, req.Environment, req.ProjectCode, boolToInt(req.RequiresApproval), now, now)
	if err != nil {
		return model.RolePermission{}, err
	}
	return s.getPermission(ctx, id)
}

func (s *DBStore) UpdatePermission(ctx context.Context, id string, req model.RolePermissionUpsertRequest) (model.RolePermission, error) {
	req = normalizePermissionRequest(req)
	if err := validatePermissionRequest(req); err != nil {
		return model.RolePermission{}, err
	}
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		UPDATE role_permissions
		SET role = ?, scope = ?, action = ?, environment = ?, project_code = ?, requires_approval = ?, updated_at = ?
		WHERE id = ?
	`, req.Role, req.Scope, req.Action, req.Environment, req.ProjectCode, boolToInt(req.RequiresApproval), now, id)
	if err != nil {
		return model.RolePermission{}, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return model.RolePermission{}, err
	}
	return s.getPermission(ctx, id)
}

func (s *DBStore) DeletePermission(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM role_permissions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return ensureRowsAffected(result)
}

func (s *DBStore) getPermission(ctx context.Context, id string) (model.RolePermission, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, role, scope, action, environment, project_code, requires_approval, created_at, updated_at
		FROM role_permissions
		WHERE id = ?
	`, id)
	var item model.RolePermission
	var requiresApproval int
	if err := row.Scan(&item.ID, &item.Role, &item.Scope, &item.Action, &item.Environment, &item.ProjectCode, &requiresApproval, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RolePermission{}, os.ErrNotExist
		}
		return model.RolePermission{}, err
	}
	item.RequiresApproval = requiresApproval == 1
	return item, nil
}

func (s *DBStore) getTool(ctx context.Context, id string) (model.ToolAsset, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT t.id, t.asset_id, t.environment, t.tool_type, t.login_policy, t.credential_policy,
		       t.approval_required, t.webssh_enabled, t.description, t.created_at, t.updated_at,
		       a.name, a.endpoint, a.owner, a.status, a.criticality, a.tags_csv
		FROM tool_assets t
		JOIN assets a ON a.id = t.asset_id
		WHERE t.id = ?
	`, id)
	item, err := scanTool(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ToolAsset{}, os.ErrNotExist
		}
		return model.ToolAsset{}, err
	}
	return item, nil
}

func (s *DBStore) getUser(ctx context.Context, id string) (model.AppUser, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, phone, role, team, status, auth_source, external_subject,
		       password_changed_at, failed_login_count, locked_until, last_login_at, created_at, updated_at
		FROM app_users
		WHERE id = ?
	`, id)
	var item model.AppUser
	if err := row.Scan(
		&item.ID, &item.Username, &item.DisplayName, &item.Email, &item.Phone, &item.Role, &item.Team,
		&item.Status, &item.AuthSource, &item.ExternalSubject, &item.PasswordChangedAt,
		&item.FailedLoginCount, &item.LockedUntil, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AppUser{}, os.ErrNotExist
		}
		return model.AppUser{}, err
	}
	return item, nil
}

func (s *DBStore) getUserWithPassword(ctx context.Context, username string) (model.AppUser, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, phone, role, team, status, auth_source, external_subject,
		       password_hash, password_changed_at, failed_login_count, locked_until, last_login_at, created_at, updated_at
		FROM app_users
		WHERE username = ?
	`, username)
	var item model.AppUser
	var passwordHash string
	if err := row.Scan(
		&item.ID, &item.Username, &item.DisplayName, &item.Email, &item.Phone, &item.Role, &item.Team,
		&item.Status, &item.AuthSource, &item.ExternalSubject, &passwordHash, &item.PasswordChangedAt,
		&item.FailedLoginCount, &item.LockedUntil, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AppUser{}, "", os.ErrNotExist
		}
		return model.AppUser{}, "", err
	}
	return item, passwordHash, nil
}

func (s *DBStore) recordLoginFailure(ctx context.Context, userID string, failedCount int) error {
	now := time.Now()
	lockedUntil := ""
	if failedCount >= 5 {
		lockedUntil = now.Add(15 * time.Minute).Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE app_users
		SET failed_login_count = ?, locked_until = ?, updated_at = ?
		WHERE id = ?
	`, failedCount, lockedUntil, now.Format(time.RFC3339), userID)
	return err
}
