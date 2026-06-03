package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"github.com/9904099/opsledger/internal/model"
)

type Store interface {
	DashboardData(ctx context.Context) (model.DashboardData, error)
	ListPlatforms(ctx context.Context) ([]model.Platform, error)
	ListCloudAccounts(ctx context.Context) ([]model.CloudAccount, error)
	ListEnvironments(ctx context.Context) ([]model.Environment, error)
	ListTools(ctx context.Context) ([]model.ToolAsset, error)
	CreateTool(ctx context.Context, req model.ToolAssetUpsertRequest) (model.ToolAsset, error)
	UpdateTool(ctx context.Context, id string, req model.ToolAssetUpsertRequest) (model.ToolAsset, error)
	ListUsers(ctx context.Context) ([]model.AppUser, error)
	CountUsers(ctx context.Context) (int, error)
	CreateInitialAdmin(ctx context.Context, req model.SetupAdminRequest) (model.AppUser, error)
	CreateUser(ctx context.Context, req model.AppUserUpsertRequest) (model.AppUser, error)
	UpdateUser(ctx context.Context, id string, req model.AppUserUpsertRequest) (model.AppUser, error)
	ListRoles(ctx context.Context) ([]model.RoleDefinition, error)
	CreateRole(ctx context.Context, req model.RoleDefinitionUpsertRequest) (model.RoleDefinition, error)
	UpdateRole(ctx context.Context, id string, req model.RoleDefinitionUpsertRequest) (model.RoleDefinition, error)
	AuthenticateUser(ctx context.Context, req model.LoginRequest, ip string, userAgent string) (model.AppUser, string, model.UserSession, error)
	CurrentUserBySessionToken(ctx context.Context, token string) (model.AppUser, model.UserSession, error)
	RevokeSession(ctx context.Context, token string) error
	ListPermissionsForRole(ctx context.Context, role string) ([]model.RolePermission, error)
	ListPermissions(ctx context.Context) ([]model.RolePermission, error)
	CreatePermission(ctx context.Context, req model.RolePermissionUpsertRequest) (model.RolePermission, error)
	UpdatePermission(ctx context.Context, id string, req model.RolePermissionUpsertRequest) (model.RolePermission, error)
	DeletePermission(ctx context.Context, id string) error
	ListApprovalFlows(ctx context.Context) ([]model.ApprovalFlow, error)
	CreateApprovalFlow(ctx context.Context, req model.ApprovalFlowUpsertRequest) (model.ApprovalFlow, error)
	UpdateApprovalFlow(ctx context.Context, id string, req model.ApprovalFlowUpsertRequest) (model.ApprovalFlow, error)
	ListApprovals(ctx context.Context) ([]model.ApprovalRequest, error)
	ListAccessGrants(ctx context.Context) ([]model.AccessGrant, error)
	ListCredentials(ctx context.Context) ([]model.CredentialItem, error)
	UpsertCredential(ctx context.Context, req model.CredentialUpsertRequest) (model.CredentialItem, error)
	RevealCredential(ctx context.Context, id string, actor model.AppUser) (model.CredentialValueResponse, error)
	RecordCredentialCopy(ctx context.Context, id string, actor model.AppUser) (model.CredentialItem, error)
	ListAuditEvents(ctx context.Context, limit int) ([]model.AuditEvent, error)
	RecordAuditEvent(ctx context.Context, event model.AuditEvent) error
	CreateApproval(ctx context.Context, req model.ApprovalCreateRequest) (model.ApprovalRequest, error)
	DecideApproval(ctx context.Context, id string, req model.ApprovalDecisionRequest, approver model.AppUser) (model.ApprovalRequest, error)
	OpenWebSSH(ctx context.Context, user model.AppUser, assetID string, ip string, userAgent string) (model.WebSSHSession, error)
	ValidateWebSSHSession(ctx context.Context, user model.AppUser, sessionID string, assetID string) (model.AccessGrant, error)
	CloseWebSSHSession(ctx context.Context, user model.AppUser, id string, status string, reason string, errorMessage string) error
	GetCloudAccount(ctx context.Context, id string) (model.CloudAccount, error)
	CreateCloudAccount(ctx context.Context, req model.CloudAccountUpsertRequest) (model.CloudAccount, error)
	UpdateCloudAccount(ctx context.Context, id string, req model.CloudAccountUpsertRequest) (model.CloudAccount, error)
	SetCloudAccountSyncResult(ctx context.Context, accountID string, result model.CloudAccountSyncResult) error
	SetCloudAccountCostResult(ctx context.Context, accountID string, result model.CloudAccountCostResult) error
	UpsertCloudAccountCostRecords(ctx context.Context, records []model.CloudAccountCostRecord) error
	ListCloudAccountCostRecords(ctx context.Context, limit int) ([]model.CloudAccountCostRecord, error)
	RecordCloudAccountSync(ctx context.Context, record model.CloudAccountSyncRecord) error
	CreateAsset(ctx context.Context, asset model.Asset) (model.Asset, error)
	UpdateAsset(ctx context.Context, id string, asset model.Asset) (model.Asset, error)
	UpsertAssetBySource(ctx context.Context, asset model.Asset) (model.Asset, bool, error)
	MarkAssetsStaleBySource(ctx context.Context, cloudAccountID string, source string, activeExternalIDs []string, syncedRegions []string, checkedAt string) (int, error)
	DeleteAsset(ctx context.Context, id string) error
	CreateChange(ctx context.Context, change model.ChangeRecord) (model.ChangeRecord, error)
	UpdateChange(ctx context.Context, id string, change model.ChangeRecord) (model.ChangeRecord, error)
	DeleteChange(ctx context.Context, id string) error
	CreateInspection(ctx context.Context, record model.InspectionRecord) (model.InspectionRecord, error)
	ListInspectionAttachments(ctx context.Context) ([]model.InspectionAttachment, error)
	CreateInspectionAttachment(ctx context.Context, req model.InspectionAttachmentCreateRequest) (model.InspectionAttachment, error)
	GetInspectionAttachment(ctx context.Context, id string) (model.InspectionAttachment, []byte, error)
	GetAsset(ctx context.Context, id string) (model.Asset, error)
	ListProbeAssets(ctx context.Context) ([]model.Asset, error)
	CreateProbe(ctx context.Context, record model.ProbeRecord) (model.ProbeRecord, error)
	LatestProbe(ctx context.Context, assetID string) (model.ProbeRecord, error)
	ListAlerts(ctx context.Context) ([]model.AlertRecord, error)
	UpsertAlert(ctx context.Context, req model.AlertUpsertRequest) (model.AlertRecord, error)
	ResolveAlert(ctx context.Context, id string, req model.AlertResolveRequest) (model.AlertRecord, error)
	ResolveOpenAlertsForAssetSource(ctx context.Context, assetID string, source string, resolver string, resolution string) error
	Close() error
}

var ErrForbidden = errors.New("forbidden")

type DBStore struct {
	db      *dialectDB
	dialect databaseDialect
}

type DatabaseConfig struct {
	Driver string
	DSN    string
	Path   string
}

func NewStore(config DatabaseConfig) (*DBStore, error) {
	dialect, sqlDriver, err := normalizeDatabaseDialect(config.Driver)
	if err != nil {
		return nil, err
	}
	switch dialect {
	case dialectSQLite:
		path := strings.TrimSpace(config.Path)
		if path == "" {
			path = strings.TrimSpace(config.DSN)
		}
		if path == "" {
			path = filepath.Join("data", "opsledger.db")
		}
		return NewSQLiteStore(path)
	case dialectPostgres:
		dsn := strings.TrimSpace(config.DSN)
		if dsn == "" {
			return nil, errors.New("postgres database dsn is required")
		}
		return newSQLStore(sqlDriver, dsn, dialect)
	case dialectMySQL:
		dsn := strings.TrimSpace(config.DSN)
		if dsn == "" {
			return nil, errors.New("mysql database dsn is required")
		}
		return newSQLStore(sqlDriver, normalizeMySQLDSN(dsn), dialect)
	}
	return nil, fmt.Errorf("unsupported database driver %q", config.Driver)
}

func NewSQLiteStore(path string) (*DBStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	store := &DBStore{db: newDialectDB(db, dialectSQLite), dialect: dialectSQLite}
	if err := store.initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func newSQLStore(driverName string, dsn string, dialect databaseDialect) (*DBStore, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &DBStore{db: newDialectDB(db, dialect), dialect: dialect}
	if err := store.initialize(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *DBStore) Close() error {
	return s.db.Close()
}

func (s *DBStore) DriverName() string {
	return string(s.dialect)
}
