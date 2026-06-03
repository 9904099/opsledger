package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
	"github.com/9904099/opsledger/internal/store"
)

func decodeJSON(body io.ReadCloser, target any) error {
	defer body.Close()

	decoder := json.NewDecoder(io.LimitReader(body, 1<<20))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, os.ErrNotExist):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, store.ErrForbidden):
		writeError(w, http.StatusForbidden, err)
	default:
		writeError(w, http.StatusBadRequest, err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func randomHexToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func cookieSecureEnabled() bool {
	return envBool("OPSLEDGER_COOKIE_SECURE", false)
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func sanitizedLogPath(path string) string {
	if strings.HasPrefix(path, "/webssh/session/") {
		return "/webssh/session/[session]"
	}
	if strings.HasPrefix(path, "/webssh/ws/") {
		return "/webssh/ws/[session]"
	}
	return path
}

func (s *Server) audit(ctx context.Context, r *http.Request, user model.AppUser, action string, targetType string, targetID string, targetName string, outcome string, summary string, metadata map[string]string) {
	actor := strings.TrimSpace(user.Username)
	actorRole := strings.TrimSpace(user.Role)
	if actor == "" {
		actor = "anonymous"
	}
	event := model.AuditEvent{
		Actor:      actor,
		ActorRole:  actorRole,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		TargetName: targetName,
		Outcome:    outcome,
		IP:         clientIP(r),
		UserAgent:  r.UserAgent(),
		Summary:    summary,
		Metadata:   metadata,
	}
	auditCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.store.RecordAuditEvent(auditCtx, event); err != nil {
		log.Printf("record audit event %s failed: %v", action, err)
	}
}

func (s *Server) recordSystemAudit(action string, targetType string, targetID string, targetName string, outcome string, summary string, metadata map[string]string) {
	auditCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.store.RecordAuditEvent(auditCtx, model.AuditEvent{
		Actor:      "system",
		ActorRole:  "scheduler",
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		TargetName: targetName,
		Outcome:    outcome,
		Summary:    summary,
		Metadata:   metadata,
	}); err != nil {
		log.Printf("record audit event %s failed: %v", action, err)
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, sanitizedLogPath(r.URL.Path), time.Since(startedAt).Round(time.Millisecond))
	})
}
