package store

import (
	"github.com/9904099/opsledger/internal/model"
	"mime"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func normalizeCloudAccountRequest(req model.CloudAccountUpsertRequest) model.CloudAccountUpsertRequest {
	req.PlatformCode = strings.ToLower(strings.TrimSpace(req.PlatformCode))
	req.Name = strings.TrimSpace(req.Name)
	req.AccountID = strings.TrimSpace(req.AccountID)
	req.DefaultRegion = strings.TrimSpace(req.DefaultRegion)
	req.Environment = strings.TrimSpace(req.Environment)
	req.Owner = strings.TrimSpace(req.Owner)
	req.Criticality = strings.TrimSpace(req.Criticality)
	req.AccessKeyID = strings.TrimSpace(req.AccessKeyID)
	req.SecretAccessKey = strings.TrimSpace(req.SecretAccessKey)
	req.SyncMode = strings.TrimSpace(req.SyncMode)
	req.SyncCron = strings.TrimSpace(req.SyncCron)
	if req.SyncMode == "" {
		req.SyncMode = "manual"
	}
	return req
}

func normalizeAsset(asset model.Asset) model.Asset {
	asset.PlatformID = strings.TrimSpace(asset.PlatformID)
	asset.PlatformCode = strings.ToLower(strings.TrimSpace(asset.PlatformCode))
	asset.PlatformName = strings.TrimSpace(asset.PlatformName)
	asset.CloudAccountID = strings.TrimSpace(asset.CloudAccountID)
	asset.CloudAccountName = strings.TrimSpace(asset.CloudAccountName)
	asset.AccountID = strings.TrimSpace(asset.AccountID)
	asset.ProjectCode = normalizeProjectCode(asset.ProjectCode)
	asset.Category = strings.TrimSpace(asset.Category)
	asset.ResourceType = strings.TrimSpace(asset.ResourceType)
	asset.Region = strings.TrimSpace(asset.Region)
	asset.Environment = strings.TrimSpace(asset.Environment)
	asset.Name = strings.TrimSpace(asset.Name)
	asset.Endpoint = strings.TrimSpace(asset.Endpoint)
	asset.Owner = strings.TrimSpace(asset.Owner)
	asset.Status = strings.TrimSpace(asset.Status)
	asset.Criticality = strings.TrimSpace(asset.Criticality)
	asset.LastCheckedAt = strings.TrimSpace(asset.LastCheckedAt)
	asset.Notes = strings.TrimSpace(asset.Notes)
	asset.Source = strings.TrimSpace(asset.Source)
	asset.ExternalID = strings.TrimSpace(asset.ExternalID)

	tags := make([]string, 0, len(asset.Tags))
	seen := map[string]struct{}{}
	for _, tag := range asset.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		lowerTag := strings.ToLower(tag)
		if _, ok := seen[lowerTag]; ok {
			continue
		}
		seen[lowerTag] = struct{}{}
		tags = append(tags, tag)
	}
	slices.Sort(tags)
	asset.Tags = tags
	if asset.ProjectCode == "" {
		asset.ProjectCode = inferProjectCode(asset)
	}
	return asset
}

func normalizeProjectCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "", "none":
		return ""
	case "common", "global", "shared", "public-resource", "公共", "公共资源":
		return "public"
	case "pve", "proxmox":
		return "pve"
	case "business":
		return "business"
	case "enterprise", "ent":
		return "enterprise"
	case "cloud":
		return "cloud"
	case "edge":
		return "edge"
	default:
		return value
	}
}

func inferProjectCode(asset model.Asset) string {
	return "public"
}

func normalizeChange(change model.ChangeRecord) model.ChangeRecord {
	change.AssetID = strings.TrimSpace(change.AssetID)
	change.Title = strings.TrimSpace(change.Title)
	change.Category = strings.TrimSpace(change.Category)
	change.Executor = strings.TrimSpace(change.Executor)
	change.RiskLevel = strings.TrimSpace(change.RiskLevel)
	change.Window = strings.TrimSpace(change.Window)
	change.Status = strings.TrimSpace(change.Status)
	change.Summary = strings.TrimSpace(change.Summary)
	change.RollbackPlan = strings.TrimSpace(change.RollbackPlan)
	return change
}

func normalizeInspection(record model.InspectionRecord) model.InspectionRecord {
	record.AssetID = strings.TrimSpace(record.AssetID)
	record.Executor = strings.TrimSpace(record.Executor)
	record.Result = strings.TrimSpace(record.Result)
	record.Summary = strings.TrimSpace(record.Summary)
	record.CheckedAt = strings.TrimSpace(record.CheckedAt)
	return record
}

func normalizeInspectionAttachmentRequest(req model.InspectionAttachmentCreateRequest) model.InspectionAttachmentCreateRequest {
	req.InspectionID = strings.TrimSpace(req.InspectionID)
	req.FileName = filepath.Base(strings.TrimSpace(req.FileName))
	req.ContentType = strings.TrimSpace(req.ContentType)
	req.Uploader = strings.TrimSpace(req.Uploader)
	req.Description = strings.TrimSpace(req.Description)
	if req.ContentType == "" {
		req.ContentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(req.FileName)))
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}
	return req
}

func normalizeToolRequest(req model.ToolAssetUpsertRequest) model.ToolAssetUpsertRequest {
	req.AssetID = strings.TrimSpace(req.AssetID)
	req.Environment = strings.TrimSpace(req.Environment)
	req.ToolType = strings.TrimSpace(req.ToolType)
	req.Name = strings.TrimSpace(req.Name)
	req.Endpoint = strings.TrimSpace(req.Endpoint)
	req.Owner = strings.TrimSpace(req.Owner)
	req.Status = strings.TrimSpace(req.Status)
	req.Criticality = strings.TrimSpace(req.Criticality)
	req.Description = strings.TrimSpace(req.Description)
	req.LoginPolicy = strings.TrimSpace(req.LoginPolicy)
	req.CredentialPolicy = strings.TrimSpace(req.CredentialPolicy)
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Criticality == "" {
		req.Criticality = "medium"
	}
	if req.LoginPolicy == "" {
		req.LoginPolicy = "sso"
	}
	if req.CredentialPolicy == "" {
		req.CredentialPolicy = "none"
	}
	tags := append([]string{}, req.Tags...)
	tags = append(tags, "tool", req.ToolType, req.Environment)
	req.Tags = splitTags(joinTags(tags))
	return req
}

func normalizeUserRequest(req model.AppUserUpsertRequest) model.AppUserUpsertRequest {
	req.Username = strings.TrimSpace(req.Username)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Role = strings.TrimSpace(req.Role)
	req.Team = strings.TrimSpace(req.Team)
	req.Status = strings.TrimSpace(req.Status)
	req.Password = strings.TrimSpace(req.Password)
	if req.Status == "" {
		req.Status = "active"
	}
	return req
}

func normalizeRoleRequest(req model.RoleDefinitionUpsertRequest) model.RoleDefinitionUpsertRequest {
	req.Code = strings.ToLower(strings.TrimSpace(req.Code))
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		req.Status = "active"
	}
	return req
}

func normalizePermissionRequest(req model.RolePermissionUpsertRequest) model.RolePermissionUpsertRequest {
	req.Role = strings.TrimSpace(req.Role)
	req.Scope = strings.TrimSpace(req.Scope)
	req.Action = strings.TrimSpace(req.Action)
	req.Environment = strings.TrimSpace(req.Environment)
	if req.Environment == "" {
		req.Environment = "*"
	}
	req.ProjectCode = normalizeProjectCode(req.ProjectCode)
	if req.ProjectCode == "" {
		req.ProjectCode = "*"
	}
	return req
}

func normalizeApprovalFlowRequest(req model.ApprovalFlowUpsertRequest) model.ApprovalFlowUpsertRequest {
	req.Name = strings.TrimSpace(req.Name)
	req.Scope = strings.TrimSpace(req.Scope)
	req.Environment = strings.TrimSpace(req.Environment)
	req.Status = strings.TrimSpace(req.Status)
	req.Description = strings.TrimSpace(req.Description)
	if req.Environment == "" {
		req.Environment = "*"
	}
	if req.Status == "" {
		req.Status = "active"
	}
	for i := range req.Steps {
		req.Steps[i].ApproverRole = strings.TrimSpace(req.Steps[i].ApproverRole)
		req.Steps[i].ApproverLabel = strings.TrimSpace(req.Steps[i].ApproverLabel)
		req.Steps[i].RequiredAction = strings.TrimSpace(req.Steps[i].RequiredAction)
		if req.Steps[i].ApproverLabel == "" {
			req.Steps[i].ApproverLabel = req.Steps[i].ApproverRole
		}
		if req.Steps[i].TimeoutMinutes <= 0 {
			req.Steps[i].TimeoutMinutes = 60
		}
	}
	return req
}

func normalizeApprovalCreateRequest(req model.ApprovalCreateRequest) model.ApprovalCreateRequest {
	req.Requester = strings.TrimSpace(req.Requester)
	req.RequestType = strings.TrimSpace(req.RequestType)
	req.TargetType = strings.TrimSpace(req.TargetType)
	req.TargetID = strings.TrimSpace(req.TargetID)
	req.Environment = strings.TrimSpace(req.Environment)
	req.Reason = strings.TrimSpace(req.Reason)
	req.PermissionLevel = strings.TrimSpace(req.PermissionLevel)
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 30
	}
	if req.PermissionLevel == "" {
		req.PermissionLevel = "read"
	}
	return req
}

func normalizeApprovalDecisionRequest(req model.ApprovalDecisionRequest) model.ApprovalDecisionRequest {
	req.Approver = strings.TrimSpace(req.Approver)
	req.Status = strings.TrimSpace(req.Status)
	req.DecisionSummary = strings.TrimSpace(req.DecisionSummary)
	return req
}

func normalizeAlertUpsertRequest(req model.AlertUpsertRequest) model.AlertUpsertRequest {
	req.AssetID = strings.TrimSpace(req.AssetID)
	req.Source = strings.TrimSpace(req.Source)
	req.Severity = strings.TrimSpace(req.Severity)
	req.Title = strings.TrimSpace(req.Title)
	req.Summary = strings.TrimSpace(req.Summary)
	req.SeenAt = strings.TrimSpace(req.SeenAt)
	if req.Source == "" {
		req.Source = "probe"
	}
	if req.Severity == "" {
		req.Severity = "warning"
	}
	if req.SeenAt == "" {
		req.SeenAt = time.Now().Format(time.RFC3339)
	}
	return req
}
