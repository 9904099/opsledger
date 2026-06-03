package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
	"slices"
	"strings"
	"time"
)

func ensureRowsAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return os.ErrNotExist
	}
	return nil
}

func joinTags(tags []string) string {
	return strings.Join(tags, ",")
}

func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	items := strings.Split(raw, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func devSeedUsersEnabled() bool {
	return envBool("OPSLEDGER_DEV_SEED_USERS", false)
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return slices.Contains([]string{"1", "true", "yes", "on"}, value)
}

func limitText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}

func maskAccessKey(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func maskSecretKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return "******"
	}
	return value[:2] + "******" + value[len(value)-2:]
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func mustJSON(value map[string]string) string {
	if value == nil {
		return "{}"
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func parseJSONMap(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}
	}
	result := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return map[string]string{}
	}
	return result
}

func newID(prefix string) string {
	buffer := make([]byte, 4)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buffer)
}
