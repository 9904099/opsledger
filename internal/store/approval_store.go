package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func (s *DBStore) ListApprovalFlows(ctx context.Context) ([]model.ApprovalFlow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, scope, environment, status, description, created_at, updated_at
		FROM approval_flows
		ORDER BY environment ASC, scope ASC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ApprovalFlow{}
	for rows.Next() {
		var item model.ApprovalFlow
		if err := rows.Scan(&item.ID, &item.Name, &item.Scope, &item.Environment, &item.Status, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		steps, err := s.listApprovalFlowSteps(ctx, items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].Steps = steps
	}
	return items, nil
}

func (s *DBStore) CreateApprovalFlow(ctx context.Context, req model.ApprovalFlowUpsertRequest) (model.ApprovalFlow, error) {
	req = normalizeApprovalFlowRequest(req)
	if err := validateApprovalFlowRequest(req); err != nil {
		return model.ApprovalFlow{}, err
	}
	now := time.Now().Format(time.RFC3339)
	id := newID("flow")
	if err := s.withTx(ctx, func(tx *dialectTx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO approval_flows (id, name, scope, environment, status, description, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, id, req.Name, req.Scope, req.Environment, req.Status, req.Description, now, now); err != nil {
			return err
		}
		return insertApprovalFlowSteps(ctx, tx, id, req.Steps, now)
	}); err != nil {
		return model.ApprovalFlow{}, err
	}
	return s.getApprovalFlow(ctx, id)
}

func (s *DBStore) UpdateApprovalFlow(ctx context.Context, id string, req model.ApprovalFlowUpsertRequest) (model.ApprovalFlow, error) {
	req = normalizeApprovalFlowRequest(req)
	if err := validateApprovalFlowRequest(req); err != nil {
		return model.ApprovalFlow{}, err
	}
	now := time.Now().Format(time.RFC3339)
	if err := s.withTx(ctx, func(tx *dialectTx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE approval_flows
			SET name = ?, scope = ?, environment = ?, status = ?, description = ?, updated_at = ?
			WHERE id = ?
		`, req.Name, req.Scope, req.Environment, req.Status, req.Description, now, id)
		if err != nil {
			return err
		}
		if err := ensureRowsAffected(result); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM approval_flow_steps WHERE flow_id = ?`, id); err != nil {
			return err
		}
		return insertApprovalFlowSteps(ctx, tx, id, req.Steps, now)
	}); err != nil {
		return model.ApprovalFlow{}, err
	}
	return s.getApprovalFlow(ctx, id)
}

func (s *DBStore) getApprovalFlow(ctx context.Context, id string) (model.ApprovalFlow, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, scope, environment, status, description, created_at, updated_at
		FROM approval_flows
		WHERE id = ?
	`, id)
	var item model.ApprovalFlow
	if err := row.Scan(&item.ID, &item.Name, &item.Scope, &item.Environment, &item.Status, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ApprovalFlow{}, os.ErrNotExist
		}
		return model.ApprovalFlow{}, err
	}
	steps, err := s.listApprovalFlowSteps(ctx, id)
	if err != nil {
		return model.ApprovalFlow{}, err
	}
	item.Steps = steps
	return item, nil
}

func (s *DBStore) listApprovalFlowSteps(ctx context.Context, flowID string) ([]model.ApprovalFlowStep, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, flow_id, step_order, approver_role, approver_label, required_action, timeout_minutes, created_at, updated_at
		FROM approval_flow_steps
		WHERE flow_id = ?
		ORDER BY step_order ASC
	`, flowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ApprovalFlowStep{}
	for rows.Next() {
		var item model.ApprovalFlowStep
		if err := rows.Scan(&item.ID, &item.FlowID, &item.StepOrder, &item.ApproverRole, &item.ApproverLabel, &item.RequiredAction, &item.TimeoutMinutes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func insertApprovalFlowSteps(ctx context.Context, tx *dialectTx, flowID string, steps []model.ApprovalFlowStepRequest, now string) error {
	for index, step := range steps {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO approval_flow_steps (
				id, flow_id, step_order, approver_role, approver_label, required_action,
				timeout_minutes, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, newID("step"), flowID, index+1, step.ApproverRole, step.ApproverLabel, step.RequiredAction, step.TimeoutMinutes, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) ListApprovals(ctx context.Context) ([]model.ApprovalRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ar.id, ar.flow_id, ar.current_step_id, ar.requester, ar.request_type, ar.target_type, ar.target_id, ar.environment,
		       ar.reason, ar.permission_level, ar.duration_minutes, ar.status, ar.approver,
		       ar.decision_summary, ar.created_at, ar.updated_at, ar.decided_at,
		       COALESCE(a.name, '')
		FROM approval_requests ar
		LEFT JOIN assets a ON a.id = ar.target_id
		ORDER BY ar.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ApprovalRequest{}
	for rows.Next() {
		item, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		tasks, err := s.listApprovalTasks(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		item.Tasks = tasks
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) ListAccessGrants(ctx context.Context) ([]model.AccessGrant, error) {
	if err := s.expireAccessGrants(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, action, target_type, target_id, environment, source_approval_id,
		       temporary_credential, temporary_credential_hash, status, expires_at, revoked_at, created_at, updated_at
		FROM access_grants
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.AccessGrant{}
	for rows.Next() {
		item, err := scanAccessGrant(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) CreateApproval(ctx context.Context, req model.ApprovalCreateRequest) (model.ApprovalRequest, error) {
	req = normalizeApprovalCreateRequest(req)
	if err := validateApprovalCreateRequest(req); err != nil {
		return model.ApprovalRequest{}, err
	}
	now := time.Now().Format(time.RFC3339)
	id := newID("apr")
	flow, steps, err := s.findApprovalFlowForRequest(ctx, req.RequestType, req.Environment)
	if err != nil {
		return model.ApprovalRequest{}, err
	}
	flowID := flow.ID
	currentStepID := ""
	if len(steps) > 0 {
		currentStepID = steps[0].ID
	}
	if err := s.withTx(ctx, func(tx *dialectTx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO approval_requests (
				id, flow_id, current_step_id, requester, request_type, target_type, target_id, environment, reason,
				permission_level, duration_minutes, status, approver, decision_summary,
				created_at, updated_at, decided_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', '', '', ?, ?, '')
		`, id, flowID, currentStepID, req.Requester, req.RequestType, req.TargetType, req.TargetID, req.Environment, req.Reason,
			req.PermissionLevel, req.DurationMinutes, now, now)
		if err != nil {
			return err
		}
		for index, step := range steps {
			status := "waiting"
			if index == 0 {
				status = "pending"
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO approval_tasks (
					id, approval_id, flow_id, step_id, step_order, approver_role, approver_label,
					status, approver, decision_summary, created_at, updated_at, decided_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', '', ?, ?, '')
			`, newID("apt"), id, flowID, step.ID, step.StepOrder, step.ApproverRole, step.ApproverLabel, status, now, now); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return model.ApprovalRequest{}, err
	}
	return s.getApproval(ctx, id)
}

func (s *DBStore) DecideApproval(ctx context.Context, id string, req model.ApprovalDecisionRequest, approver model.AppUser) (model.ApprovalRequest, error) {
	req = normalizeApprovalDecisionRequest(req)
	req.Approver = approver.Username
	if err := validateApprovalDecisionRequest(req); err != nil {
		return model.ApprovalRequest{}, err
	}
	now := time.Now().Format(time.RFC3339)
	existing, err := s.getApproval(ctx, id)
	if err != nil {
		return model.ApprovalRequest{}, err
	}
	if existing.Status != "pending" {
		return model.ApprovalRequest{}, errors.New("approval is not pending")
	}
	if err := s.withTx(ctx, func(tx *dialectTx) error {
		if existing.FlowID == "" || existing.CurrentStepID == "" {
			if !canApproveLegacyRequest(approver.Role) {
				return ErrForbidden
			}
			result, err := tx.ExecContext(ctx, `
				UPDATE approval_requests
				SET status = ?, approver = ?, decision_summary = ?, decided_at = ?, updated_at = ?
				WHERE id = ? AND status = 'pending'
			`, req.Status, req.Approver, req.DecisionSummary, now, now, id)
			if err != nil {
				return err
			}
			if err := ensureRowsAffected(result); err != nil {
				return err
			}
			if req.Status == "approved" {
				return insertAccessGrantFromApprovalTx(ctx, tx, existing, now)
			}
			return nil
		}

		task, err := getPendingApprovalTaskTx(ctx, tx, id, existing.CurrentStepID)
		if err != nil {
			return err
		}
		if !canApproveTask(approver.Role, task.ApproverRole) {
			return ErrForbidden
		}
		if req.Status == "rejected" {
			if _, err := tx.ExecContext(ctx, `
				UPDATE approval_tasks
				SET status = 'rejected', approver = ?, decision_summary = ?, decided_at = ?, updated_at = ?
				WHERE id = ? AND status = 'pending'
			`, req.Approver, req.DecisionSummary, now, now, task.ID); err != nil {
				return err
			}
			result, err := tx.ExecContext(ctx, `
				UPDATE approval_requests
				SET status = 'rejected', approver = ?, decision_summary = ?, decided_at = ?, updated_at = ?
				WHERE id = ? AND status = 'pending'
			`, req.Approver, req.DecisionSummary, now, now, id)
			if err != nil {
				return err
			}
			return ensureRowsAffected(result)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE approval_tasks
			SET status = 'approved', approver = ?, decision_summary = ?, decided_at = ?, updated_at = ?
			WHERE id = ? AND status = 'pending'
		`, req.Approver, req.DecisionSummary, now, now, task.ID); err != nil {
			return err
		}
		next, hasNext, err := getNextWaitingApprovalTaskTx(ctx, tx, id, task.StepOrder)
		if err != nil {
			return err
		}
		if hasNext {
			if _, err := tx.ExecContext(ctx, `
				UPDATE approval_tasks
				SET status = 'pending', updated_at = ?
				WHERE id = ? AND status = 'waiting'
			`, now, next.ID); err != nil {
				return err
			}
			result, err := tx.ExecContext(ctx, `
				UPDATE approval_requests
				SET current_step_id = ?, approver = ?, decision_summary = ?, updated_at = ?
				WHERE id = ? AND status = 'pending'
			`, next.StepID, req.Approver, req.DecisionSummary, now, id)
			if err != nil {
				return err
			}
			return ensureRowsAffected(result)
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE approval_requests
			SET status = 'approved', approver = ?, decision_summary = ?, decided_at = ?, updated_at = ?
			WHERE id = ? AND status = 'pending'
		`, req.Approver, req.DecisionSummary, now, now, id)
		if err != nil {
			return err
		}
		if err := ensureRowsAffected(result); err != nil {
			return err
		}
		return insertAccessGrantFromApprovalTx(ctx, tx, existing, now)
	}); err != nil {
		return model.ApprovalRequest{}, err
	}
	return s.getApproval(ctx, id)
}

func (s *DBStore) OpenWebSSH(ctx context.Context, user model.AppUser, assetID string, ip string, userAgent string) (model.WebSSHSession, error) {
	if strings.TrimSpace(assetID) == "" {
		return model.WebSSHSession{}, errors.New("asset_id is required")
	}
	if err := s.expireAccessGrants(ctx); err != nil {
		return model.WebSSHSession{}, err
	}
	asset, err := s.GetAsset(ctx, assetID)
	if err != nil {
		return model.WebSSHSession{}, err
	}
	if strings.ToLower(asset.PlatformCode) != "aws" || strings.ToLower(asset.ResourceType) != "ec2" {
		return model.WebSSHSession{}, errors.New("webssh only supports aws ec2 assets")
	}

	grant, err := s.getActiveAccessGrant(ctx, user.Username, "webssh", "asset", assetID)
	if err != nil {
		return model.WebSSHSession{}, err
	}
	sessionToken, err := randomToken(16)
	if err != nil {
		return model.WebSSHSession{}, err
	}
	now := time.Now().Format(time.RFC3339)
	session := model.WebSSHSession{
		ID:            "wss-" + sessionToken,
		Username:      user.Username,
		AssetID:       assetID,
		AssetName:     asset.Name,
		AccessGrantID: grant.ID,
		Status:        "active",
		IP:            strings.TrimSpace(ip),
		UserAgent:     strings.TrimSpace(userAgent),
		StartedAt:     now,
		ExpiresAt:     grant.ExpiresAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	session.LoginURL = fmt.Sprintf("/webssh/session/%s?asset=%s", url.PathEscape(session.ID), url.QueryEscape(assetID))
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO webssh_sessions (
			id, username, asset_id, access_grant_id, status, login_url,
			ip, user_agent, close_reason, error_message, started_at, expires_at, ended_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', '', ?, ?, '', ?, ?)
	`, session.ID, session.Username, session.AssetID, session.AccessGrantID, session.Status, session.LoginURL,
		session.IP, session.UserAgent, session.StartedAt, session.ExpiresAt, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return model.WebSSHSession{}, err
	}
	return session, nil
}

func (s *DBStore) ValidateWebSSHSession(ctx context.Context, user model.AppUser, sessionID string, assetID string) (model.AccessGrant, error) {
	sessionID = strings.TrimSpace(sessionID)
	assetID = strings.TrimSpace(assetID)
	if sessionID == "" || assetID == "" {
		return model.AccessGrant{}, ErrForbidden
	}
	if err := s.expireAccessGrants(ctx); err != nil {
		return model.AccessGrant{}, err
	}
	now := time.Now().Format(time.RFC3339)
	row := s.db.QueryRowContext(ctx, `
		SELECT ag.id, ag.username, ag.action, ag.target_type, ag.target_id, ag.environment, ag.source_approval_id,
		       ag.temporary_credential, ag.temporary_credential_hash, ag.status, ag.expires_at, ag.revoked_at, ag.created_at, ag.updated_at
		FROM webssh_sessions ws
		JOIN access_grants ag ON ag.id = ws.access_grant_id
		WHERE ws.id = ? AND ws.username = ? AND ws.asset_id = ? AND ws.status = 'active'
		  AND ag.username = ? AND ag.action = 'webssh' AND ag.target_type = 'asset' AND ag.target_id = ?
		  AND ag.status = 'active' AND ag.revoked_at = '' AND ag.expires_at > ? AND ws.expires_at > ?
		LIMIT 1
	`, sessionID, user.Username, assetID, user.Username, assetID, now, now)
	item, err := scanAccessGrant(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AccessGrant{}, ErrForbidden
		}
		return model.AccessGrant{}, err
	}
	return item, nil
}

func (s *DBStore) CloseWebSSHSession(ctx context.Context, user model.AppUser, id string, status string, reason string, errorMessage string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("webssh session id is required")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		status = "closed"
	}
	if !slices.Contains([]string{"closed", "failed", "expired"}, status) {
		status = "closed"
	}
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		UPDATE webssh_sessions
		SET status = ?,
		    close_reason = ?,
		    error_message = ?,
		    ended_at = CASE WHEN ended_at = '' THEN ? ELSE ended_at END,
		    updated_at = ?
		WHERE id = ? AND username = ?
	`, status, limitText(reason, 240), limitText(errorMessage, 1000), now, now, id, user.Username)
	if err != nil {
		return err
	}
	return ensureRowsAffected(result)
}

func (s *DBStore) getApproval(ctx context.Context, id string) (model.ApprovalRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT ar.id, ar.flow_id, ar.current_step_id, ar.requester, ar.request_type, ar.target_type, ar.target_id, ar.environment,
		       ar.reason, ar.permission_level, ar.duration_minutes, ar.status, ar.approver,
		       ar.decision_summary, ar.created_at, ar.updated_at, ar.decided_at,
		       COALESCE(a.name, '')
		FROM approval_requests ar
		LEFT JOIN assets a ON a.id = ar.target_id
		WHERE ar.id = ?
	`, id)
	item, err := scanApproval(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ApprovalRequest{}, os.ErrNotExist
		}
		return model.ApprovalRequest{}, err
	}
	tasks, err := s.listApprovalTasks(ctx, id)
	if err != nil {
		return model.ApprovalRequest{}, err
	}
	item.Tasks = tasks
	return item, nil
}

func (s *DBStore) getActiveAccessGrant(ctx context.Context, username string, action string, targetType string, targetID string) (model.AccessGrant, error) {
	now := time.Now().Format(time.RFC3339)
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, action, target_type, target_id, environment, source_approval_id,
		       temporary_credential, temporary_credential_hash, status, expires_at, revoked_at, created_at, updated_at
		FROM access_grants
		WHERE username = ? AND action = ? AND target_type = ? AND target_id = ?
		  AND status = 'active' AND revoked_at = '' AND expires_at > ?
		ORDER BY expires_at DESC
		LIMIT 1
	`, username, action, targetType, targetID, now)
	item, err := scanAccessGrant(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.AccessGrant{}, ErrForbidden
		}
		return model.AccessGrant{}, err
	}
	return item, nil
}

func (s *DBStore) expireAccessGrants(ctx context.Context) error {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE access_grants
		SET status = 'expired', temporary_credential = '', temporary_credential_hash = '', updated_at = ?
		WHERE status = 'active' AND expires_at <= ?
	`, now, now)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE webssh_sessions
		SET status = 'expired',
		    close_reason = CASE WHEN close_reason = '' THEN 'grant expired' ELSE close_reason END,
		    ended_at = CASE WHEN ended_at = '' THEN ? ELSE ended_at END,
		    updated_at = ?
		WHERE status = 'active' AND expires_at <= ?
	`, now, now, now)
	return err
}

func (s *DBStore) listApprovalTasks(ctx context.Context, approvalID string) ([]model.ApprovalTask, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, approval_id, flow_id, step_id, step_order, approver_role, approver_label,
		       status, approver, decision_summary, created_at, updated_at, decided_at
		FROM approval_tasks
		WHERE approval_id = ?
		ORDER BY step_order ASC
	`, approvalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.ApprovalTask{}
	for rows.Next() {
		task, err := scanApprovalTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, task)
	}
	return items, rows.Err()
}

func (s *DBStore) findApprovalFlowForRequest(ctx context.Context, requestType string, environment string) (model.ApprovalFlow, []model.ApprovalFlowStep, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, scope, environment, status, description, created_at, updated_at
		FROM approval_flows
		WHERE status = 'active'
		  AND (scope = ? OR scope = '*')
		  AND (environment = ? OR environment = '*')
		ORDER BY
		  CASE WHEN environment = ? THEN 0 ELSE 1 END,
		  CASE WHEN scope = ? THEN 0 ELSE 1 END,
		  updated_at DESC
		LIMIT 1
	`, requestType, environment, environment, requestType)
	if err != nil {
		return model.ApprovalFlow{}, nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return model.ApprovalFlow{}, nil, nil
	}
	var flow model.ApprovalFlow
	if err := rows.Scan(&flow.ID, &flow.Name, &flow.Scope, &flow.Environment, &flow.Status, &flow.Description, &flow.CreatedAt, &flow.UpdatedAt); err != nil {
		return model.ApprovalFlow{}, nil, err
	}
	steps, err := s.listApprovalFlowSteps(ctx, flow.ID)
	if err != nil {
		return model.ApprovalFlow{}, nil, err
	}
	return flow, steps, rows.Err()
}

func getPendingApprovalTaskTx(ctx context.Context, tx *dialectTx, approvalID string, stepID string) (model.ApprovalTask, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, approval_id, flow_id, step_id, step_order, approver_role, approver_label,
		       status, approver, decision_summary, created_at, updated_at, decided_at
		FROM approval_tasks
		WHERE approval_id = ? AND step_id = ? AND status = 'pending'
	`, approvalID, stepID)
	task, err := scanApprovalTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ApprovalTask{}, os.ErrNotExist
		}
		return model.ApprovalTask{}, err
	}
	return task, nil
}

func getNextWaitingApprovalTaskTx(ctx context.Context, tx *dialectTx, approvalID string, currentOrder int) (model.ApprovalTask, bool, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, approval_id, flow_id, step_id, step_order, approver_role, approver_label,
		       status, approver, decision_summary, created_at, updated_at, decided_at
		FROM approval_tasks
		WHERE approval_id = ? AND status = 'waiting' AND step_order > ?
		ORDER BY step_order ASC
		LIMIT 1
	`, approvalID, currentOrder)
	task, err := scanApprovalTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ApprovalTask{}, false, nil
		}
		return model.ApprovalTask{}, false, err
	}
	return task, true, nil
}

func insertAccessGrantFromApprovalTx(ctx context.Context, tx *dialectTx, approval model.ApprovalRequest, now string) error {
	action := approval.RequestType
	if action == "temporary_credential" {
		action = "credential"
	}
	if action == "" || approval.TargetID == "" {
		return nil
	}
	expiresAt := time.Now().Add(time.Duration(approval.DurationMinutes) * time.Minute).Format(time.RFC3339)
	temporaryCredential, err := randomToken(24)
	if err != nil {
		return err
	}
	temporaryCredentialHash := hashSessionToken(temporaryCredential)
	if _, err := tx.ExecContext(ctx, `
		UPDATE access_grants
		SET status = 'revoked', temporary_credential = '', temporary_credential_hash = '', revoked_at = ?, updated_at = ?
		WHERE username = ? AND action = ? AND target_type = ? AND target_id = ? AND status = 'active'
	`, now, now, approval.Requester, action, approval.TargetType, approval.TargetID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO access_grants (
			id, username, action, target_type, target_id, environment, source_approval_id,
			temporary_credential, temporary_credential_hash, status, expires_at, revoked_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, '', ?, 'active', ?, '', ?, ?)
	`, newID("grant"), approval.Requester, action, approval.TargetType, approval.TargetID, approval.Environment,
		approval.ID, temporaryCredentialHash, expiresAt, now, now)
	return err
}

func canApproveTask(userRole string, approverRole string) bool {
	userRole = strings.TrimSpace(userRole)
	approverRole = strings.TrimSpace(approverRole)
	return userRole == "admin" || userRole == approverRole
}

func canApproveLegacyRequest(userRole string) bool {
	return userRole == "admin" || userRole == "ops" || userRole == "lead"
}
