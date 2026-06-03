package model

type Platform struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CloudAccount struct {
	ID                    string `json:"id"`
	PlatformID            string `json:"platform_id"`
	PlatformCode          string `json:"platform_code"`
	PlatformName          string `json:"platform_name"`
	Name                  string `json:"name"`
	AccountID             string `json:"account_id"`
	DefaultRegion         string `json:"default_region"`
	Environment           string `json:"environment"`
	Owner                 string `json:"owner"`
	Criticality           string `json:"criticality"`
	AccessKeyIDMasked     string `json:"access_key_id_masked"`
	SecretAccessKeyMasked string `json:"secret_access_key_masked"`
	SyncEnabled           bool   `json:"sync_enabled"`
	SyncMode              string `json:"sync_mode"`
	SyncCron              string `json:"sync_cron"`
	LastSyncAt            string `json:"last_sync_at"`
	LastSyncStatus        string `json:"last_sync_status"`
	LastSyncSummary       string `json:"last_sync_summary"`
	CostCurrency          string `json:"cost_currency"`
	LastMonthCost         string `json:"last_month_cost"`
	LastMonthToDateCost   string `json:"last_month_to_date_cost"`
	CurrentMonthCost      string `json:"current_month_cost"`
	ForecastMonthCost     string `json:"forecast_month_cost"`
	MonthOverMonthDelta   string `json:"month_over_month_delta"`
	LastCostSyncAt        string `json:"last_cost_sync_at"`
	LastCostSyncStatus    string `json:"last_cost_sync_status"`
	LastCostSyncSummary   string `json:"last_cost_sync_summary"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

type Asset struct {
	ID               string            `json:"id"`
	PlatformID       string            `json:"platform_id"`
	PlatformCode     string            `json:"platform_code"`
	PlatformName     string            `json:"platform_name"`
	CloudAccountID   string            `json:"cloud_account_id"`
	CloudAccountName string            `json:"cloud_account_name"`
	AccountID        string            `json:"account_id"`
	ProjectCode      string            `json:"project_code"`
	Category         string            `json:"category"`
	ResourceType     string            `json:"resource_type"`
	Region           string            `json:"region"`
	Environment      string            `json:"environment"`
	Name             string            `json:"name"`
	Endpoint         string            `json:"endpoint"`
	Owner            string            `json:"owner"`
	Status           string            `json:"status"`
	Criticality      string            `json:"criticality"`
	LastCheckedAt    string            `json:"last_checked_at"`
	Tags             []string          `json:"tags"`
	Notes            string            `json:"notes"`
	Specs            map[string]string `json:"specs"`
	Source           string            `json:"source"`
	ExternalID       string            `json:"external_id"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
}

type ChangeRecord struct {
	ID           string `json:"id"`
	AssetID      string `json:"asset_id"`
	Title        string `json:"title"`
	Category     string `json:"category"`
	Executor     string `json:"executor"`
	RiskLevel    string `json:"risk_level"`
	Window       string `json:"window"`
	Status       string `json:"status"`
	Summary      string `json:"summary"`
	RollbackPlan string `json:"rollback_plan"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type CloudAccountSyncRecord struct {
	ID               string         `json:"id"`
	CloudAccountID   string         `json:"cloud_account_id"`
	StartedAt        string         `json:"started_at"`
	FinishedAt       string         `json:"finished_at"`
	Status           string         `json:"status"`
	DiscoveredAssets int            `json:"discovered_assets"`
	CreatedAssets    int            `json:"created_assets"`
	UpdatedAssets    int            `json:"updated_assets"`
	StaleAssets      int            `json:"stale_assets"`
	Warnings         []string       `json:"warnings"`
	Breakdown        map[string]int `json:"breakdown"`
	Summary          string         `json:"summary"`
}

type InspectionRecord struct {
	ID        string `json:"id"`
	AssetID   string `json:"asset_id"`
	Executor  string `json:"executor"`
	Result    string `json:"result"`
	Summary   string `json:"summary"`
	CheckedAt string `json:"checked_at"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type InspectionAttachment struct {
	ID           string `json:"id"`
	InspectionID string `json:"inspection_id"`
	AssetID      string `json:"asset_id"`
	FileName     string `json:"file_name"`
	ContentType  string `json:"content_type"`
	SizeBytes    int64  `json:"size_bytes"`
	Uploader     string `json:"uploader"`
	Description  string `json:"description"`
	CreatedAt    string `json:"created_at"`
}

type ProbeRecord struct {
	ID                string `json:"id"`
	AssetID           string `json:"asset_id"`
	URL               string `json:"url"`
	Method            string `json:"method"`
	Status            string `json:"status"`
	StatusCode        int    `json:"status_code"`
	LatencyMS         int    `json:"latency_ms"`
	Error             string `json:"error"`
	CheckedAt         string `json:"checked_at"`
	TLSExpiresAt      string `json:"tls_expires_at"`
	CertDaysRemaining int    `json:"cert_days_remaining"`
	CreatedAt         string `json:"created_at"`
}

type AlertRecord struct {
	ID          string `json:"id"`
	AssetID     string `json:"asset_id"`
	AssetName   string `json:"asset_name"`
	Source      string `json:"source"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	FirstSeenAt string `json:"first_seen_at"`
	LastSeenAt  string `json:"last_seen_at"`
	ResolvedAt  string `json:"resolved_at"`
	ResolvedBy  string `json:"resolved_by"`
	Resolution  string `json:"resolution"`
	EventCount  int    `json:"event_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Environment struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Owner       string `json:"owner"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ToolAsset struct {
	ID               string   `json:"id"`
	AssetID          string   `json:"asset_id"`
	Environment      string   `json:"environment"`
	ToolType         string   `json:"tool_type"`
	LoginPolicy      string   `json:"login_policy"`
	CredentialPolicy string   `json:"credential_policy"`
	ApprovalRequired bool     `json:"approval_required"`
	WebSSHEnabled    bool     `json:"webssh_enabled"`
	Description      string   `json:"description"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	AssetName        string   `json:"asset_name"`
	Endpoint         string   `json:"endpoint"`
	Owner            string   `json:"owner"`
	Status           string   `json:"status"`
	Criticality      string   `json:"criticality"`
	Tags             []string `json:"tags"`
}

type AppUser struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	DisplayName       string `json:"display_name"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	Role              string `json:"role"`
	Team              string `json:"team"`
	Status            string `json:"status"`
	AuthSource        string `json:"auth_source"`
	ExternalSubject   string `json:"external_subject"`
	PasswordChangedAt string `json:"password_changed_at"`
	FailedLoginCount  int    `json:"failed_login_count"`
	LockedUntil       string `json:"locked_until"`
	LastLoginAt       string `json:"last_login_at"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type RolePermission struct {
	ID               string `json:"id"`
	Role             string `json:"role"`
	Scope            string `json:"scope"`
	Action           string `json:"action"`
	Environment      string `json:"environment"`
	ProjectCode      string `json:"project_code"`
	RequiresApproval bool   `json:"requires_approval"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type RoleDefinition struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Level       int    `json:"level"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type RoleDefinitionUpsertRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Level       int    `json:"level"`
	Status      string `json:"status"`
}

type RolePermissionUpsertRequest struct {
	Role             string `json:"role"`
	Scope            string `json:"scope"`
	Action           string `json:"action"`
	Environment      string `json:"environment"`
	ProjectCode      string `json:"project_code"`
	RequiresApproval bool   `json:"requires_approval"`
}

type ApprovalFlow struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Scope       string             `json:"scope"`
	Environment string             `json:"environment"`
	Status      string             `json:"status"`
	Description string             `json:"description"`
	Steps       []ApprovalFlowStep `json:"steps"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

type ApprovalFlowStep struct {
	ID             string `json:"id"`
	FlowID         string `json:"flow_id"`
	StepOrder      int    `json:"step_order"`
	ApproverRole   string `json:"approver_role"`
	ApproverLabel  string `json:"approver_label"`
	RequiredAction string `json:"required_action"`
	TimeoutMinutes int    `json:"timeout_minutes"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type ApprovalFlowUpsertRequest struct {
	Name        string                    `json:"name"`
	Scope       string                    `json:"scope"`
	Environment string                    `json:"environment"`
	Status      string                    `json:"status"`
	Description string                    `json:"description"`
	Steps       []ApprovalFlowStepRequest `json:"steps"`
}

type ApprovalFlowStepRequest struct {
	ApproverRole   string `json:"approver_role"`
	ApproverLabel  string `json:"approver_label"`
	RequiredAction string `json:"required_action"`
	TimeoutMinutes int    `json:"timeout_minutes"`
}

type ApprovalRequest struct {
	ID              string         `json:"id"`
	FlowID          string         `json:"flow_id"`
	CurrentStepID   string         `json:"current_step_id"`
	Requester       string         `json:"requester"`
	RequestType     string         `json:"request_type"`
	TargetType      string         `json:"target_type"`
	TargetID        string         `json:"target_id"`
	Environment     string         `json:"environment"`
	Reason          string         `json:"reason"`
	PermissionLevel string         `json:"permission_level"`
	DurationMinutes int            `json:"duration_minutes"`
	Status          string         `json:"status"`
	Approver        string         `json:"approver"`
	DecisionSummary string         `json:"decision_summary"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
	DecidedAt       string         `json:"decided_at"`
	TargetName      string         `json:"target_name"`
	Tasks           []ApprovalTask `json:"tasks"`
}

type ApprovalTask struct {
	ID              string `json:"id"`
	ApprovalID      string `json:"approval_id"`
	FlowID          string `json:"flow_id"`
	StepID          string `json:"step_id"`
	StepOrder       int    `json:"step_order"`
	ApproverRole    string `json:"approver_role"`
	ApproverLabel   string `json:"approver_label"`
	Status          string `json:"status"`
	Approver        string `json:"approver"`
	DecisionSummary string `json:"decision_summary"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	DecidedAt       string `json:"decided_at"`
}

type AccessGrant struct {
	ID                      string `json:"id"`
	Username                string `json:"username"`
	Action                  string `json:"action"`
	TargetType              string `json:"target_type"`
	TargetID                string `json:"target_id"`
	Environment             string `json:"environment"`
	SourceApprovalID        string `json:"source_approval_id"`
	TemporaryCredential     string `json:"-"`
	TemporaryCredentialHash string `json:"-"`
	Status                  string `json:"status"`
	ExpiresAt               string `json:"expires_at"`
	RevokedAt               string `json:"revoked_at"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
}

type CredentialItem struct {
	ID            string `json:"id"`
	OwnerType     string `json:"owner_type"`
	OwnerID       string `json:"owner_id"`
	OwnerName     string `json:"owner_name"`
	Kind          string `json:"kind"`
	KeyName       string `json:"key_name"`
	MaskedValue   string `json:"masked_value"`
	Environment   string `json:"environment"`
	ProjectCode   string `json:"project_code"`
	AccessPolicy  string `json:"access_policy"`
	Status        string `json:"status"`
	LastViewedAt  string `json:"last_viewed_at"`
	LastViewedBy  string `json:"last_viewed_by"`
	LastRotatedAt string `json:"last_rotated_at"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type CredentialValueResponse struct {
	Credential CredentialItem `json:"credential"`
	Value      string         `json:"value"`
}

type CredentialUpsertRequest struct {
	OwnerType    string `json:"owner_type"`
	OwnerID      string `json:"owner_id"`
	Kind         string `json:"kind"`
	KeyName      string `json:"key_name"`
	Value        string `json:"value"`
	Environment  string `json:"environment"`
	ProjectCode  string `json:"project_code"`
	AccessPolicy string `json:"access_policy"`
	Status       string `json:"status"`
}

type WebSSHSession struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	AssetID       string `json:"asset_id"`
	AssetName     string `json:"asset_name"`
	AccessGrantID string `json:"access_grant_id"`
	Status        string `json:"status"`
	LoginURL      string `json:"login_url"`
	IP            string `json:"ip"`
	UserAgent     string `json:"user_agent"`
	CloseReason   string `json:"close_reason"`
	ErrorMessage  string `json:"error_message"`
	StartedAt     string `json:"started_at"`
	ExpiresAt     string `json:"expires_at"`
	EndedAt       string `json:"ended_at"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type WebSSHOpenRequest struct {
	AssetID string `json:"asset_id"`
}

type AuditEvent struct {
	ID         string            `json:"id"`
	Actor      string            `json:"actor"`
	ActorRole  string            `json:"actor_role"`
	Action     string            `json:"action"`
	TargetType string            `json:"target_type"`
	TargetID   string            `json:"target_id"`
	TargetName string            `json:"target_name"`
	Outcome    string            `json:"outcome"`
	IP         string            `json:"ip"`
	UserAgent  string            `json:"user_agent"`
	Summary    string            `json:"summary"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  string            `json:"created_at"`
}

type Summary struct {
	TotalAssets        int            `json:"total_assets"`
	ActiveAssets       int            `json:"active_assets"`
	MaintenanceAssets  int            `json:"maintenance_assets"`
	CriticalAssets     int            `json:"critical_assets"`
	ProbeAlerts        int            `json:"probe_alerts"`
	OpenAlerts         int            `json:"open_alerts"`
	ToolAssets         int            `json:"tool_assets"`
	PendingApprovals   int            `json:"pending_approvals"`
	PlannedChanges     int            `json:"planned_changes"`
	InProgressChanges  int            `json:"in_progress_changes"`
	CompletedChanges   int            `json:"completed_changes"`
	AssetsByPlatform   map[string]int `json:"assets_by_platform"`
	AssetsByCategory   map[string]int `json:"assets_by_category"`
	AssetsByEnv        map[string]int `json:"assets_by_environment"`
	ChangesByRiskLevel map[string]int `json:"changes_by_risk_level"`
}

type DashboardData struct {
	Platforms     []Platform               `json:"platforms"`
	CloudAccounts []CloudAccount           `json:"cloud_accounts"`
	CostRecords   []CloudAccountCostRecord `json:"cloud_account_cost_records"`
	Assets        []Asset                  `json:"assets"`
	Environments  []Environment            `json:"environments"`
	Tools         []ToolAsset              `json:"tools"`
	Users         []AppUser                `json:"users"`
	Roles         []RoleDefinition         `json:"roles"`
	Permissions   []RolePermission         `json:"permissions"`
	ApprovalFlows []ApprovalFlow           `json:"approval_flows"`
	Approvals     []ApprovalRequest        `json:"approvals"`
	AccessGrants  []AccessGrant            `json:"access_grants"`
	Credentials   []CredentialItem         `json:"credentials"`
	Changes       []ChangeRecord           `json:"changes"`
	Inspections   []InspectionRecord       `json:"inspections"`
	Attachments   []InspectionAttachment   `json:"attachments"`
	Probes        []ProbeRecord            `json:"probes"`
	Alerts        []AlertRecord            `json:"alerts"`
	RecentSyncs   []CloudAccountSyncRecord `json:"recent_syncs"`
	AuditEvents   []AuditEvent             `json:"audit_events"`
	Summary       Summary                  `json:"summary"`
	GeneratedAt   string                   `json:"generated_at"`
}

type AssetImportRequest struct {
	Assets []Asset `json:"assets"`
}

type AssetImportResult struct {
	ImportedAssets int      `json:"imported_assets"`
	SkippedAssets  int      `json:"skipped_assets"`
	CreatedIDs     []string `json:"created_ids"`
	Warnings       []string `json:"warnings"`
}

type AssetBulkUpdateRequest struct {
	AssetIDs     []string  `json:"asset_ids"`
	ProjectCode  *string   `json:"project_code,omitempty"`
	Category     *string   `json:"category,omitempty"`
	ResourceType *string   `json:"resource_type,omitempty"`
	Region       *string   `json:"region,omitempty"`
	Environment  *string   `json:"environment,omitempty"`
	Owner        *string   `json:"owner,omitempty"`
	Status       *string   `json:"status,omitempty"`
	Criticality  *string   `json:"criticality,omitempty"`
	Tags         *[]string `json:"tags,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
}

type AssetBulkUpdateResult struct {
	UpdatedAssets int      `json:"updated_assets"`
	SkippedAssets int      `json:"skipped_assets"`
	UpdatedIDs    []string `json:"updated_ids"`
	Warnings      []string `json:"warnings"`
}

type UserSession struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	TokenHash  string `json:"-"`
	ExpiresAt  string `json:"expires_at"`
	LastSeenAt string `json:"last_seen_at"`
	RevokedAt  string `json:"revoked_at"`
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type CloudAccountUpsertRequest struct {
	PlatformCode    string `json:"platform_code"`
	Name            string `json:"name"`
	AccountID       string `json:"account_id"`
	DefaultRegion   string `json:"default_region"`
	Environment     string `json:"environment"`
	Owner           string `json:"owner"`
	Criticality     string `json:"criticality"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SyncEnabled     bool   `json:"sync_enabled"`
	SyncMode        string `json:"sync_mode"`
	SyncCron        string `json:"sync_cron"`
}

type CloudAccountSyncRequest struct {
	CloudAccountID string `json:"cloud_account_id"`
	Region         string `json:"region"`
}

type CloudAccountSyncResult struct {
	CloudAccountID    string         `json:"cloud_account_id"`
	CloudAccountName  string         `json:"cloud_account_name"`
	PlatformCode      string         `json:"platform_code"`
	AccountID         string         `json:"account_id"`
	Regions           []string       `json:"regions"`
	DiscoveredAssets  int            `json:"discovered_assets"`
	CreatedAssets     int            `json:"created_assets"`
	UpdatedAssets     int            `json:"updated_assets"`
	StaleAssets       int            `json:"stale_assets"`
	ResourceBreakdown map[string]int `json:"resource_breakdown"`
	Warnings          []string       `json:"warnings"`
	StartedAt         string         `json:"started_at"`
	FinishedAt        string         `json:"finished_at"`
}

type CloudAccountCostResult struct {
	CloudAccountID      string `json:"cloud_account_id"`
	CloudAccountName    string `json:"cloud_account_name"`
	PlatformCode        string `json:"platform_code"`
	Currency            string `json:"currency"`
	LastMonthCost       string `json:"last_month_cost"`
	LastMonthToDateCost string `json:"last_month_to_date_cost"`
	CurrentMonthCost    string `json:"current_month_cost"`
	ForecastMonthCost   string `json:"forecast_month_cost"`
	MonthOverMonthDelta string `json:"month_over_month_delta"`
	LastMonthStart      string `json:"last_month_start"`
	LastMonthEnd        string `json:"last_month_end"`
	CurrentMonthStart   string `json:"current_month_start"`
	CurrentMonthEnd     string `json:"current_month_end"`
	StartedAt           string `json:"started_at"`
	FinishedAt          string `json:"finished_at"`
	Summary             string `json:"summary"`
}

type CloudAccountCostRecord struct {
	ID             string `json:"id"`
	CloudAccountID string `json:"cloud_account_id"`
	PeriodStart    string `json:"period_start"`
	PeriodEnd      string `json:"period_end"`
	Granularity    string `json:"granularity"`
	DimensionType  string `json:"dimension_type"`
	DimensionName  string `json:"dimension_name"`
	Currency       string `json:"currency"`
	Amount         string `json:"amount"`
	Source         string `json:"source"`
	SyncedAt       string `json:"synced_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type InspectionRecordCreateRequest struct {
	AssetID   string `json:"asset_id"`
	Executor  string `json:"executor"`
	Result    string `json:"result"`
	Summary   string `json:"summary"`
	CheckedAt string `json:"checked_at"`
}

type AlertUpsertRequest struct {
	AssetID  string `json:"asset_id"`
	Source   string `json:"source"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	SeenAt   string `json:"seen_at"`
}

type AlertResolveRequest struct {
	Resolver   string `json:"resolver"`
	Resolution string `json:"resolution"`
}

type InspectionAttachmentCreateRequest struct {
	InspectionID string
	FileName     string
	ContentType  string
	Data         []byte
	Uploader     string
	Description  string
}

type ToolAssetUpsertRequest struct {
	AssetID          string   `json:"asset_id"`
	Environment      string   `json:"environment"`
	ToolType         string   `json:"tool_type"`
	Name             string   `json:"name"`
	Endpoint         string   `json:"endpoint"`
	Owner            string   `json:"owner"`
	Status           string   `json:"status"`
	Criticality      string   `json:"criticality"`
	Tags             []string `json:"tags"`
	Description      string   `json:"description"`
	LoginPolicy      string   `json:"login_policy"`
	CredentialPolicy string   `json:"credential_policy"`
	ApprovalRequired bool     `json:"approval_required"`
	WebSSHEnabled    bool     `json:"webssh_enabled"`
}

type AppUserUpsertRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Role        string `json:"role"`
	Team        string `json:"team"`
	Status      string `json:"status"`
	Password    string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User      AppUser `json:"user"`
	ExpiresAt string  `json:"expires_at"`
}

type SetupStatusResponse struct {
	Required       bool   `json:"required"`
	DatabaseReady  bool   `json:"database_ready"`
	UserCount      int    `json:"user_count"`
	Driver         string `json:"driver"`
	Message        string `json:"message"`
	SetupCompleted bool   `json:"setup_completed"`
}

type SetupAdminRequest struct {
	Username        string `json:"username"`
	DisplayName     string `json:"display_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type CurrentUserResponse struct {
	User        AppUser          `json:"user"`
	Permissions []RolePermission `json:"permissions"`
}

type ApprovalCreateRequest struct {
	Requester       string `json:"requester"`
	RequestType     string `json:"request_type"`
	TargetType      string `json:"target_type"`
	TargetID        string `json:"target_id"`
	Environment     string `json:"environment"`
	Reason          string `json:"reason"`
	PermissionLevel string `json:"permission_level"`
	DurationMinutes int    `json:"duration_minutes"`
}

type ApprovalDecisionRequest struct {
	Approver        string `json:"approver"`
	Status          string `json:"status"`
	DecisionSummary string `json:"decision_summary"`
}
