package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

func (s *DBStore) ListCredentials(ctx context.Context) ([]model.CredentialItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.owner_type, c.owner_id,
		       COALESCE(ca.name, a.name, '') AS owner_name,
		       c.kind, c.key_name, c.masked_value, c.environment, c.project_code, c.access_policy,
		       c.status, c.last_viewed_at, c.last_viewed_by, c.last_rotated_at, c.created_at, c.updated_at
		FROM credentials c
		LEFT JOIN cloud_accounts ca ON c.owner_type = 'cloud_account' AND ca.id = c.owner_id
		LEFT JOIN assets a ON c.owner_type = 'asset' AND a.id = c.owner_id
		ORDER BY c.owner_type ASC, owner_name ASC, c.kind ASC, c.key_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CredentialItem{}
	for rows.Next() {
		var item model.CredentialItem
		if err := rows.Scan(
			&item.ID, &item.OwnerType, &item.OwnerID, &item.OwnerName, &item.Kind, &item.KeyName,
			&item.MaskedValue, &item.Environment, &item.ProjectCode, &item.AccessPolicy,
			&item.Status, &item.LastViewedAt, &item.LastViewedBy, &item.LastRotatedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DBStore) RevealCredential(ctx context.Context, id string, actor model.AppUser) (model.CredentialValueResponse, error) {
	item, encryptedValue, err := s.getCredentialWithSecret(ctx, id)
	if err != nil {
		return model.CredentialValueResponse{}, err
	}
	if item.Status != "active" {
		return model.CredentialValueResponse{}, ErrForbidden
	}
	if err := s.authorizeCredentialAccess(ctx, item, actor); err != nil {
		return model.CredentialValueResponse{}, err
	}
	value, err := decryptCredentialValue(encryptedValue)
	if err != nil {
		return model.CredentialValueResponse{}, err
	}
	now := time.Now().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE credentials
		SET last_viewed_at = ?, last_viewed_by = ?, updated_at = ?
		WHERE id = ?
	`, now, actor.Username, now, id); err != nil {
		return model.CredentialValueResponse{}, err
	}
	item.LastViewedAt = now
	item.LastViewedBy = actor.Username
	return model.CredentialValueResponse{Credential: item, Value: value}, nil
}

func (s *DBStore) UpsertCredential(ctx context.Context, req model.CredentialUpsertRequest) (model.CredentialItem, error) {
	req = normalizeCredentialRequest(req)
	if err := validateCredentialRequest(req); err != nil {
		return model.CredentialItem{}, err
	}
	item, err := s.credentialItemFromRequest(ctx, req)
	if err != nil {
		return model.CredentialItem{}, err
	}
	if err := s.upsertCredential(ctx, item, req.Value); err != nil {
		return model.CredentialItem{}, err
	}
	created, _, err := s.getCredentialByOwner(ctx, req.OwnerType, req.OwnerID, req.Kind, req.KeyName)
	return created, err
}

func (s *DBStore) RecordCredentialCopy(ctx context.Context, id string, actor model.AppUser) (model.CredentialItem, error) {
	item, _, err := s.getCredentialWithSecret(ctx, id)
	if err != nil {
		return model.CredentialItem{}, err
	}
	if item.Status != "active" {
		return model.CredentialItem{}, ErrForbidden
	}
	if err := s.authorizeCredentialAccess(ctx, item, actor); err != nil {
		return model.CredentialItem{}, err
	}
	now := time.Now().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE credentials
		SET last_viewed_at = ?, last_viewed_by = ?, updated_at = ?
		WHERE id = ?
	`, now, actor.Username, now, id); err != nil {
		return model.CredentialItem{}, err
	}
	item.LastViewedAt = now
	item.LastViewedBy = actor.Username
	return item, nil
}

func (s *DBStore) authorizeCredentialAccess(ctx context.Context, item model.CredentialItem, actor model.AppUser) error {
	if canRevealCredential(actor) {
		return nil
	}
	if actor.Username == "" {
		return ErrForbidden
	}
	if err := s.expireAccessGrants(ctx); err != nil {
		return err
	}
	if _, err := s.getActiveAccessGrant(ctx, actor.Username, "credential", item.OwnerType, item.OwnerID); err == nil {
		return nil
	}
	if item.OwnerType == "asset" {
		if _, err := s.getActiveAccessGrant(ctx, actor.Username, "credential", "tool", item.OwnerID); err == nil {
			return nil
		}
	}
	if _, err := s.getActiveAccessGrant(ctx, actor.Username, "credential", "credential", item.ID); err == nil {
		return nil
	}
	return ErrForbidden
}

func (s *DBStore) getCredentialWithSecret(ctx context.Context, id string) (model.CredentialItem, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.owner_type, c.owner_id,
		       COALESCE(ca.name, a.name, '') AS owner_name,
		       c.kind, c.key_name, c.masked_value, c.environment, c.project_code, c.access_policy,
		       c.status, c.last_viewed_at, c.last_viewed_by, c.last_rotated_at, c.created_at, c.updated_at,
		       c.encrypted_value
		FROM credentials c
		LEFT JOIN cloud_accounts ca ON c.owner_type = 'cloud_account' AND ca.id = c.owner_id
		LEFT JOIN assets a ON c.owner_type = 'asset' AND a.id = c.owner_id
		WHERE c.id = ?
	`, id)

	var item model.CredentialItem
	var encryptedValue string
	if err := row.Scan(
		&item.ID, &item.OwnerType, &item.OwnerID, &item.OwnerName, &item.Kind, &item.KeyName,
		&item.MaskedValue, &item.Environment, &item.ProjectCode, &item.AccessPolicy,
		&item.Status, &item.LastViewedAt, &item.LastViewedBy, &item.LastRotatedAt, &item.CreatedAt, &item.UpdatedAt,
		&encryptedValue,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CredentialItem{}, "", os.ErrNotExist
		}
		return model.CredentialItem{}, "", err
	}
	return item, encryptedValue, nil
}

func (s *DBStore) getCredentialByOwner(ctx context.Context, ownerType string, ownerID string, kind string, keyName string) (model.CredentialItem, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.owner_type, c.owner_id,
		       COALESCE(ca.name, a.name, '') AS owner_name,
		       c.kind, c.key_name, c.masked_value, c.environment, c.project_code, c.access_policy,
		       c.status, c.last_viewed_at, c.last_viewed_by, c.last_rotated_at, c.created_at, c.updated_at,
		       c.encrypted_value
		FROM credentials c
		LEFT JOIN cloud_accounts ca ON c.owner_type = 'cloud_account' AND ca.id = c.owner_id
		LEFT JOIN assets a ON c.owner_type = 'asset' AND a.id = c.owner_id
		WHERE c.owner_type = ? AND c.owner_id = ? AND c.kind = ? AND c.key_name = ?
	`, ownerType, ownerID, kind, keyName)

	var item model.CredentialItem
	var encryptedValue string
	if err := row.Scan(
		&item.ID, &item.OwnerType, &item.OwnerID, &item.OwnerName, &item.Kind, &item.KeyName,
		&item.MaskedValue, &item.Environment, &item.ProjectCode, &item.AccessPolicy,
		&item.Status, &item.LastViewedAt, &item.LastViewedBy, &item.LastRotatedAt, &item.CreatedAt, &item.UpdatedAt,
		&encryptedValue,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CredentialItem{}, "", os.ErrNotExist
		}
		return model.CredentialItem{}, "", err
	}
	return item, encryptedValue, nil
}

func (s *DBStore) getCredentialPlainValue(ctx context.Context, ownerType string, ownerID string, kind string, keyName string) (string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT encrypted_value
		FROM credentials
		WHERE owner_type = ? AND owner_id = ? AND kind = ? AND key_name = ? AND status = 'active'
	`, ownerType, ownerID, kind, keyName)

	var encryptedValue string
	if err := row.Scan(&encryptedValue); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", os.ErrNotExist
		}
		return "", err
	}
	return decryptCredentialValue(encryptedValue)
}

func (s *DBStore) credentialItemFromRequest(ctx context.Context, req model.CredentialUpsertRequest) (model.CredentialItem, error) {
	item := model.CredentialItem{
		OwnerType:    req.OwnerType,
		OwnerID:      req.OwnerID,
		Kind:         req.Kind,
		KeyName:      req.KeyName,
		MaskedValue:  maskCredentialValue(req.Kind, req.Value),
		Environment:  req.Environment,
		ProjectCode:  req.ProjectCode,
		AccessPolicy: req.AccessPolicy,
		Status:       req.Status,
	}
	switch req.OwnerType {
	case "cloud_account":
		account, err := s.GetCloudAccount(ctx, req.OwnerID)
		if err != nil {
			return model.CredentialItem{}, err
		}
		if item.Environment == "" {
			item.Environment = account.Environment
		}
		if item.ProjectCode == "" {
			item.ProjectCode = "cloud"
		}
	case "asset", "tool":
		asset, err := s.GetAsset(ctx, req.OwnerID)
		if err != nil {
			return model.CredentialItem{}, err
		}
		item.OwnerType = "asset"
		if item.Environment == "" {
			item.Environment = asset.Environment
		}
		if item.ProjectCode == "" {
			item.ProjectCode = asset.ProjectCode
		}
	default:
		return model.CredentialItem{}, errors.New("unsupported credential owner_type")
	}
	return item, nil
}

func (s *DBStore) upsertCloudAccountCredentials(ctx context.Context, accountID string, accessKeyID string, secretAccessKey string) error {
	account, err := s.GetCloudAccount(ctx, accountID)
	if err != nil {
		return err
	}
	if accessKeyID != "" {
		if err := s.upsertCredential(ctx, model.CredentialItem{
			OwnerType:    "cloud_account",
			OwnerID:      accountID,
			Kind:         "access_key_id",
			KeyName:      "default",
			MaskedValue:  maskAccessKey(accessKeyID),
			Environment:  account.Environment,
			ProjectCode:  "cloud",
			AccessPolicy: "ops_only",
			Status:       "active",
		}, accessKeyID); err != nil {
			return err
		}
	}
	if secretAccessKey != "" {
		if err := s.upsertCredential(ctx, model.CredentialItem{
			OwnerType:    "cloud_account",
			OwnerID:      accountID,
			Kind:         "secret_access_key",
			KeyName:      "default",
			MaskedValue:  maskSecretKey(secretAccessKey),
			Environment:  account.Environment,
			ProjectCode:  "cloud",
			AccessPolicy: "ops_only",
			Status:       "active",
		}, secretAccessKey); err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) upsertCredential(ctx context.Context, item model.CredentialItem, plainValue string) error {
	encryptedValue, err := encryptCredentialValue(plainValue)
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	if item.ID == "" {
		item.ID = newID("cred")
	}
	if item.AccessPolicy == "" {
		item.AccessPolicy = "ops_only"
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if item.ProjectCode == "" {
		item.ProjectCode = "public"
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO credentials (
			id, owner_type, owner_id, kind, key_name, encrypted_value, masked_value,
			environment, project_code, access_policy, status, last_rotated_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(owner_type, owner_id, kind, key_name) DO UPDATE SET
			encrypted_value = excluded.encrypted_value,
			masked_value = excluded.masked_value,
			environment = excluded.environment,
			project_code = excluded.project_code,
			access_policy = excluded.access_policy,
			status = excluded.status,
			last_rotated_at = excluded.last_rotated_at,
			updated_at = excluded.updated_at
	`, item.ID, item.OwnerType, item.OwnerID, item.Kind, item.KeyName, encryptedValue, item.MaskedValue,
		item.Environment, item.ProjectCode, item.AccessPolicy, item.Status, now, now, now)
	return err
}

func validateCredentialRequest(req model.CredentialUpsertRequest) error {
	switch {
	case req.OwnerType == "":
		return errors.New("owner_type is required")
	case req.OwnerID == "":
		return errors.New("owner_id is required")
	case req.Kind == "":
		return errors.New("kind is required")
	case req.Value == "":
		return errors.New("value is required")
	case req.Status != "active" && req.Status != "disabled":
		return errors.New("status must be active or disabled")
	}
	switch req.Kind {
	case "username", "password", "api_token", "access_key_id", "secret_access_key", "ssh_key", "secret":
		return nil
	default:
		return errors.New("unsupported credential kind")
	}
}

func canRevealCredential(actor model.AppUser) bool {
	switch actor.Role {
	case "admin", "ops":
		return true
	default:
		return false
	}
}

func encryptCredentialValue(value string) (string, error) {
	block, err := aes.NewCipher(credentialEncryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(value), nil)
	payload := append(nonce, ciphertext...)
	return "v1:" + base64.RawURLEncoding.EncodeToString(payload), nil
}

func decryptCredentialValue(value string) (string, error) {
	encoded := strings.TrimPrefix(value, "v1:")
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(credentialEncryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("invalid credential payload")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func credentialEncryptionKey() []byte {
	material := strings.TrimSpace(os.Getenv("OPSLEDGER_CREDENTIAL_KEY"))
	if material == "" {
		material = "opsledger-development-credential-key"
	}
	sum := sha256.Sum256([]byte(material))
	return sum[:]
}

func normalizeCredentialRequest(req model.CredentialUpsertRequest) model.CredentialUpsertRequest {
	req.OwnerType = strings.ToLower(strings.TrimSpace(req.OwnerType))
	if req.OwnerType == "tool" {
		req.OwnerType = "asset"
	}
	req.OwnerID = strings.TrimSpace(req.OwnerID)
	req.Kind = strings.ToLower(strings.TrimSpace(req.Kind))
	req.KeyName = strings.TrimSpace(req.KeyName)
	if req.KeyName == "" {
		req.KeyName = "default"
	}
	req.Value = strings.TrimSpace(req.Value)
	req.Environment = strings.TrimSpace(req.Environment)
	req.ProjectCode = normalizeProjectCode(req.ProjectCode)
	req.AccessPolicy = strings.TrimSpace(req.AccessPolicy)
	if req.AccessPolicy == "" {
		req.AccessPolicy = "ops_only"
	}
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if req.Status == "" {
		req.Status = "active"
	}
	return req
}

func maskCredentialValue(kind string, value string) string {
	value = strings.TrimSpace(value)
	switch kind {
	case "access_key_id":
		return maskAccessKey(value)
	case "username":
		if value == "" {
			return ""
		}
		if len(value) <= 2 {
			return value[:1] + "*"
		}
		return value[:1] + "***" + value[len(value)-1:]
	default:
		return maskSecretKey(value)
	}
}
