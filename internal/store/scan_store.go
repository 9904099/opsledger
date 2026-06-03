package store

import (
	"encoding/json"
	"fmt"
	"github.com/9904099/opsledger/internal/model"
	"strings"
)

type assetScanner interface {
	Scan(dest ...any) error
}

type changeScanner interface {
	Scan(dest ...any) error
}

type syncScanner interface {
	Scan(dest ...any) error
}

type inspectionScanner interface {
	Scan(dest ...any) error
}

type inspectionAttachmentScanner interface {
	Scan(dest ...any) error
}

type probeScanner interface {
	Scan(dest ...any) error
}

type alertScanner interface {
	Scan(dest ...any) error
}

type toolScanner interface {
	Scan(dest ...any) error
}

type approvalScanner interface {
	Scan(dest ...any) error
}

type approvalTaskScanner interface {
	Scan(dest ...any) error
}

type accessGrantScanner interface {
	Scan(dest ...any) error
}

type auditEventScanner interface {
	Scan(dest ...any) error
}

func scanAsset(scanner assetScanner) (model.Asset, error) {
	var item model.Asset
	var tags string
	var specsJSON string
	err := scanner.Scan(
		&item.ID, &item.PlatformID, &item.PlatformCode, &item.PlatformName, &item.CloudAccountID, &item.CloudAccountName, &item.AccountID,
		&item.ProjectCode, &item.Category, &item.ResourceType, &item.Region, &item.Environment, &item.Name, &item.Endpoint, &item.Owner, &item.Status, &item.Criticality,
		&item.LastCheckedAt, &tags, &item.Notes, &specsJSON, &item.Source, &item.ExternalID, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return model.Asset{}, err
	}
	item.Tags = splitTags(tags)
	item.Specs = parseJSONMap(specsJSON)
	return item, nil
}

func scanTool(scanner toolScanner) (model.ToolAsset, error) {
	var item model.ToolAsset
	var approvalRequired int
	var webSSHEnabled int
	var tags string
	if err := scanner.Scan(
		&item.ID, &item.AssetID, &item.Environment, &item.ToolType, &item.LoginPolicy, &item.CredentialPolicy,
		&approvalRequired, &webSSHEnabled, &item.Description, &item.CreatedAt, &item.UpdatedAt,
		&item.AssetName, &item.Endpoint, &item.Owner, &item.Status, &item.Criticality, &tags,
	); err != nil {
		return model.ToolAsset{}, err
	}
	item.ApprovalRequired = approvalRequired == 1
	item.WebSSHEnabled = webSSHEnabled == 1
	item.Tags = splitTags(tags)
	return item, nil
}

func scanApproval(scanner approvalScanner) (model.ApprovalRequest, error) {
	var item model.ApprovalRequest
	if err := scanner.Scan(
		&item.ID, &item.FlowID, &item.CurrentStepID, &item.Requester, &item.RequestType, &item.TargetType, &item.TargetID, &item.Environment,
		&item.Reason, &item.PermissionLevel, &item.DurationMinutes, &item.Status, &item.Approver,
		&item.DecisionSummary, &item.CreatedAt, &item.UpdatedAt, &item.DecidedAt, &item.TargetName,
	); err != nil {
		return model.ApprovalRequest{}, err
	}
	return item, nil
}

func scanApprovalTask(scanner approvalTaskScanner) (model.ApprovalTask, error) {
	var item model.ApprovalTask
	if err := scanner.Scan(
		&item.ID, &item.ApprovalID, &item.FlowID, &item.StepID, &item.StepOrder,
		&item.ApproverRole, &item.ApproverLabel, &item.Status, &item.Approver,
		&item.DecisionSummary, &item.CreatedAt, &item.UpdatedAt, &item.DecidedAt,
	); err != nil {
		return model.ApprovalTask{}, err
	}
	return item, nil
}

func scanAccessGrant(scanner accessGrantScanner) (model.AccessGrant, error) {
	var item model.AccessGrant
	if err := scanner.Scan(
		&item.ID, &item.Username, &item.Action, &item.TargetType, &item.TargetID, &item.Environment,
		&item.SourceApprovalID, &item.TemporaryCredential, &item.TemporaryCredentialHash, &item.Status, &item.ExpiresAt,
		&item.RevokedAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return model.AccessGrant{}, err
	}
	return item, nil
}

func scanAuditEvent(scanner auditEventScanner) (model.AuditEvent, error) {
	var item model.AuditEvent
	var metadataJSON string
	if err := scanner.Scan(
		&item.ID, &item.Actor, &item.ActorRole, &item.Action, &item.TargetType, &item.TargetID,
		&item.TargetName, &item.Outcome, &item.IP, &item.UserAgent, &item.Summary,
		&metadataJSON, &item.CreatedAt,
	); err != nil {
		return model.AuditEvent{}, err
	}
	item.Metadata = map[string]string{}
	if strings.TrimSpace(metadataJSON) != "" {
		_ = json.Unmarshal([]byte(metadataJSON), &item.Metadata)
	}
	return item, nil
}

func scanChange(scanner changeScanner) (model.ChangeRecord, error) {
	var item model.ChangeRecord
	err := scanner.Scan(
		&item.ID, &item.AssetID, &item.Title, &item.Category, &item.Executor, &item.RiskLevel,
		&item.Window, &item.Status, &item.Summary, &item.RollbackPlan, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return model.ChangeRecord{}, err
	}
	return item, nil
}

func scanSyncRecord(scanner syncScanner) (model.CloudAccountSyncRecord, error) {
	var item model.CloudAccountSyncRecord
	var warningsJSON string
	var breakdownJSON string
	if err := scanner.Scan(
		&item.ID, &item.CloudAccountID, &item.StartedAt, &item.FinishedAt, &item.Status, &item.DiscoveredAssets,
		&item.CreatedAssets, &item.UpdatedAssets, &warningsJSON, &breakdownJSON, &item.Summary,
	); err != nil {
		return model.CloudAccountSyncRecord{}, err
	}
	item.StaleAssets = parseStaleCountFromSyncSummary(item.Summary)
	_ = json.Unmarshal([]byte(warningsJSON), &item.Warnings)
	_ = json.Unmarshal([]byte(breakdownJSON), &item.Breakdown)
	if item.Breakdown == nil {
		item.Breakdown = map[string]int{}
	}
	return item, nil
}

func parseStaleCountFromSyncSummary(summary string) int {
	marker := "标记 stale "
	index := strings.Index(summary, marker)
	if index < 0 {
		return 0
	}
	rest := strings.TrimSpace(summary[index+len(marker):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return 0
	}
	var value int
	if _, err := fmt.Sscanf(fields[0], "%d", &value); err != nil {
		return 0
	}
	return value
}

func scanInspection(scanner inspectionScanner) (model.InspectionRecord, error) {
	var item model.InspectionRecord
	if err := scanner.Scan(
		&item.ID, &item.AssetID, &item.Executor, &item.Result, &item.Summary, &item.CheckedAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return model.InspectionRecord{}, err
	}
	return item, nil
}

func scanInspectionAttachment(scanner inspectionAttachmentScanner) (model.InspectionAttachment, error) {
	var item model.InspectionAttachment
	if err := scanner.Scan(
		&item.ID, &item.InspectionID, &item.AssetID, &item.FileName, &item.ContentType, &item.SizeBytes,
		&item.Uploader, &item.Description, &item.CreatedAt,
	); err != nil {
		return model.InspectionAttachment{}, err
	}
	return item, nil
}

func scanProbe(scanner probeScanner) (model.ProbeRecord, error) {
	var item model.ProbeRecord
	if err := scanner.Scan(
		&item.ID, &item.AssetID, &item.URL, &item.Method, &item.Status, &item.StatusCode, &item.LatencyMS, &item.Error,
		&item.CheckedAt, &item.TLSExpiresAt, &item.CertDaysRemaining, &item.CreatedAt,
	); err != nil {
		return model.ProbeRecord{}, err
	}
	return item, nil
}

func scanAlert(scanner alertScanner) (model.AlertRecord, error) {
	var item model.AlertRecord
	if err := scanner.Scan(
		&item.ID, &item.AssetID, &item.AssetName, &item.Source, &item.Severity, &item.Status,
		&item.Title, &item.Summary, &item.FirstSeenAt, &item.LastSeenAt, &item.ResolvedAt,
		&item.ResolvedBy, &item.Resolution, &item.EventCount, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return model.AlertRecord{}, err
	}
	return item, nil
}
