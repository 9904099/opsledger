package store

import (
	"errors"
	"fmt"
	"github.com/9904099/opsledger/internal/model"
	"strings"
)

const maxInspectionAttachmentSize = 10 << 20

func validateCloudAccountRequest(req model.CloudAccountUpsertRequest) error {
	switch {
	case strings.TrimSpace(req.PlatformCode) == "":
		return errors.New("platform_code is required")
	case strings.TrimSpace(req.Name) == "":
		return errors.New("name is required")
	case strings.TrimSpace(req.Environment) == "":
		return errors.New("environment is required")
	case strings.TrimSpace(req.Owner) == "":
		return errors.New("owner is required")
	case strings.TrimSpace(req.Criticality) == "":
		return errors.New("criticality is required")
	default:
		return nil
	}
}

func validateAsset(asset model.Asset) error {
	switch {
	case strings.TrimSpace(asset.PlatformCode) == "":
		return errors.New("platform_code is required")
	case strings.TrimSpace(asset.Environment) == "":
		return errors.New("environment is required")
	case strings.TrimSpace(asset.Category) == "":
		return errors.New("category is required")
	case strings.TrimSpace(asset.ResourceType) == "":
		return errors.New("resource_type is required")
	case strings.TrimSpace(asset.Name) == "":
		return errors.New("name is required")
	case strings.TrimSpace(asset.Owner) == "":
		return errors.New("owner is required")
	case strings.TrimSpace(asset.Status) == "":
		return errors.New("status is required")
	case strings.TrimSpace(asset.Criticality) == "":
		return errors.New("criticality is required")
	case strings.TrimSpace(asset.LastCheckedAt) == "":
		return errors.New("last_checked_at is required")
	default:
		return nil
	}
}

func validateChange(change model.ChangeRecord) error {
	switch {
	case change.AssetID == "":
		return errors.New("asset_id is required")
	case change.Title == "":
		return errors.New("title is required")
	case change.Category == "":
		return errors.New("category is required")
	case change.Executor == "":
		return errors.New("executor is required")
	case change.RiskLevel == "":
		return errors.New("risk_level is required")
	case change.Window == "":
		return errors.New("window is required")
	case change.Status == "":
		return errors.New("status is required")
	case change.RollbackPlan == "":
		return errors.New("rollback_plan is required")
	default:
		return nil
	}
}

func validateInspection(record model.InspectionRecord) error {
	switch {
	case record.AssetID == "":
		return errors.New("asset_id is required")
	case record.Executor == "":
		return errors.New("executor is required")
	case record.Result == "":
		return errors.New("result is required")
	case record.CheckedAt == "":
		return errors.New("checked_at is required")
	default:
		return nil
	}
}

func validateInspectionAttachmentRequest(req model.InspectionAttachmentCreateRequest) error {
	switch {
	case req.InspectionID == "":
		return errors.New("inspection_id is required")
	case req.FileName == "":
		return errors.New("file_name is required")
	case len(req.Data) == 0:
		return errors.New("attachment file is required")
	case len(req.Data) > maxInspectionAttachmentSize:
		return fmt.Errorf("attachment exceeds %d bytes", maxInspectionAttachmentSize)
	default:
		return nil
	}
}

func validateToolRequest(req model.ToolAssetUpsertRequest) error {
	switch {
	case req.Environment == "":
		return errors.New("environment is required")
	case req.ToolType == "":
		return errors.New("tool_type is required")
	case req.Name == "":
		return errors.New("name is required")
	case req.Endpoint == "":
		return errors.New("endpoint is required")
	case req.Owner == "":
		return errors.New("owner is required")
	default:
		return nil
	}
}

func validateUserRequest(req model.AppUserUpsertRequest) error {
	switch {
	case req.Username == "":
		return errors.New("username is required")
	case req.DisplayName == "":
		return errors.New("display_name is required")
	case req.Role == "":
		return errors.New("role is required")
	case req.Status == "":
		return errors.New("status is required")
	default:
		return nil
	}
}

func validateRoleRequest(req model.RoleDefinitionUpsertRequest) error {
	switch {
	case req.Code == "":
		return errors.New("code is required")
	case req.Name == "":
		return errors.New("name is required")
	case req.Status == "":
		return errors.New("status is required")
	default:
		return nil
	}
}

func validatePermissionRequest(req model.RolePermissionUpsertRequest) error {
	switch {
	case req.Role == "":
		return errors.New("role is required")
	case req.Scope == "":
		return errors.New("scope is required")
	case req.Action == "":
		return errors.New("action is required")
	case req.Environment == "":
		return errors.New("environment is required")
	default:
		return nil
	}
}

func validateApprovalFlowRequest(req model.ApprovalFlowUpsertRequest) error {
	switch {
	case req.Name == "":
		return errors.New("name is required")
	case req.Scope == "":
		return errors.New("scope is required")
	case req.Environment == "":
		return errors.New("environment is required")
	case req.Status == "":
		return errors.New("status is required")
	case len(req.Steps) == 0:
		return errors.New("steps is required")
	}
	for _, step := range req.Steps {
		if strings.TrimSpace(step.ApproverRole) == "" {
			return errors.New("step approver_role is required")
		}
		if strings.TrimSpace(step.RequiredAction) == "" {
			return errors.New("step required_action is required")
		}
	}
	return nil
}

func validateApprovalCreateRequest(req model.ApprovalCreateRequest) error {
	switch {
	case req.Requester == "":
		return errors.New("requester is required")
	case req.RequestType == "":
		return errors.New("request_type is required")
	case req.TargetType == "":
		return errors.New("target_type is required")
	case req.Environment == "":
		return errors.New("environment is required")
	case req.Reason == "":
		return errors.New("reason is required")
	default:
		return nil
	}
}

func validateApprovalDecisionRequest(req model.ApprovalDecisionRequest) error {
	switch {
	case req.Approver == "":
		return errors.New("approver is required")
	case req.Status != "approved" && req.Status != "rejected":
		return errors.New("status must be approved or rejected")
	default:
		return nil
	}
}
