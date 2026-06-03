package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/9904099/opsledger/internal/model"
)

const sessionCookieName = "opsledger_session"
const csrfCookieName = "opsledger_csrf"
const csrfHeaderName = "X-OpsLedger-CSRF"

type contextKey string

const currentUserContextKey contextKey = "current_user"

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	ensureCSRFCookie(w, r)
	count, err := s.store.CountUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	driver := ""
	if provider, ok := s.store.(interface{ DriverName() string }); ok {
		driver = provider.DriverName()
	}
	required := count == 0
	message := "setup completed"
	if required {
		message = "database initialized; create the first administrator"
	}
	writeJSON(w, http.StatusOK, model.SetupStatusResponse{
		Required:       required,
		DatabaseReady:  true,
		UserCount:      count,
		Driver:         driver,
		Message:        message,
		SetupCompleted: !required,
	})
}

func (s *Server) handleSetupInitialAdmin(w http.ResponseWriter, r *http.Request) {
	ensureCSRFCookie(w, r)
	var req model.SetupAdminRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, err := s.store.CreateInitialAdmin(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	authedUser, token, session, err := s.store.AuthenticateUser(r.Context(), model.LoginRequest{Username: req.Username, Password: req.Password}, clientIP(r), r.UserAgent())
	if err != nil {
		s.audit(r.Context(), r, user, "setup.initial_admin", "user", user.Username, user.DisplayName, "success", "initial administrator created; login failed", map[string]string{"error": err.Error()})
		writeJSON(w, http.StatusCreated, map[string]any{"user": user, "login_required": true})
		return
	}
	expiresAt, _ := time.Parse(time.RFC3339, session.ExpiresAt)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   cookieSecureEnabled(),
		SameSite: http.SameSiteLaxMode,
	})
	s.audit(r.Context(), r, authedUser, "setup.initial_admin", "user", authedUser.Username, authedUser.DisplayName, "success", "initial administrator created", nil)
	writeJSON(w, http.StatusCreated, model.LoginResponse{User: authedUser, ExpiresAt: session.ExpiresAt})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ensureCSRFCookie(w, r)
	var req model.LoginRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		s.audit(r.Context(), r, model.AppUser{}, "auth.login", "user", "", req.Username, "failed", err.Error(), nil)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, token, session, err := s.store.AuthenticateUser(r.Context(), req, clientIP(r), r.UserAgent())
	if err != nil {
		s.audit(r.Context(), r, model.AppUser{Username: req.Username}, "auth.login", "user", req.Username, req.Username, "failed", err.Error(), nil)
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	expiresAt, _ := time.Parse(time.RFC3339, session.ExpiresAt)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   cookieSecureEnabled(),
		SameSite: http.SameSiteLaxMode,
	})
	s.audit(r.Context(), r, user, "auth.login", "user", user.Username, user.DisplayName, "success", "用户登录成功", nil)
	writeJSON(w, http.StatusOK, model.LoginResponse{User: user, ExpiresAt: session.ExpiresAt})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_ = s.store.RevokeSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   cookieSecureEnabled(),
		SameSite: http.SameSiteLaxMode,
	})
	s.audit(r.Context(), r, user, "auth.logout", "user", user.Username, user.DisplayName, "success", "用户退出登录", nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	permissions, err := s.store.ListPermissionsForRole(r.Context(), user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, model.CurrentUserResponse{User: user, Permissions: permissions})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r) {
			if err := validateCSRF(r); err != nil {
				writeError(w, http.StatusForbidden, err)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if err := validateCSRF(r); err != nil {
			writeError(w, http.StatusForbidden, err)
			return
		}
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
		user, _, err := s.store.CurrentUserBySessionToken(r.Context(), cookie.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
		if !isAuthorizedRoute(user, r) {
			s.audit(r.Context(), r, user, "auth.forbidden", "route", r.URL.Path, "", "denied", "forbidden", map[string]string{"method": r.Method})
			writeError(w, http.StatusForbidden, errors.New("forbidden"))
			return
		}
		ctx := context.WithValue(r.Context(), currentUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicRoute(r *http.Request) bool {
	if r.Method == http.MethodGet && (r.URL.Path == "/" || r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/static/")) {
		return true
	}
	if r.Method == http.MethodGet && r.URL.Path == "/api/setup" {
		return true
	}
	if r.Method == http.MethodPost && r.URL.Path == "/api/setup" {
		return true
	}
	if r.Method == http.MethodPost && r.URL.Path == "/api/auth/login" {
		return true
	}
	return false
}

func currentUser(ctx context.Context) (model.AppUser, bool) {
	user, ok := ctx.Value(currentUserContextKey).(model.AppUser)
	return user, ok
}

func isAuthorizedRoute(user model.AppUser, r *http.Request) bool {
	if r.Method == http.MethodGet && r.URL.Path == "/api/auth/me" {
		return true
	}
	if r.Method == http.MethodPost && r.URL.Path == "/api/auth/logout" {
		return true
	}
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/webssh/session/") {
		return true
	}
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/webssh/ws/") {
		return true
	}

	if user.Role == "admin" || user.Role == "ops" {
		return true
	}

	if r.Method == http.MethodGet {
		switch r.URL.Path {
		case "/api/bootstrap", "/api/platforms":
			return true
		}
	}

	if user.Role == "developer" || user.Role == "lead" {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/inspection-attachments/") && strings.HasSuffix(r.URL.Path, "/download") {
			return true
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/approvals" {
			return true
		}
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/credentials/") &&
			(strings.HasSuffix(r.URL.Path, "/reveal") || strings.HasSuffix(r.URL.Path, "/copy")) {
			return true
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/webssh/open" {
			return true
		}
		if user.Role == "lead" && r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/approvals/") && strings.HasSuffix(r.URL.Path, "/decision") {
			return true
		}
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/assets/") && strings.HasSuffix(r.URL.Path, "/probe") {
			return true
		}
		return false
	}

	if user.Role == "auditor" {
		return r.Method == http.MethodGet && (r.URL.Path == "/api/bootstrap" || r.URL.Path == "/api/platforms" || r.URL.Path == "/api/audit-events")
	}

	if user.Role == "viewer" {
		return r.Method == http.MethodGet && (r.URL.Path == "/api/bootstrap" || r.URL.Path == "/api/platforms")
	}

	return false
}

func clientIP(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			value = strings.TrimSpace(strings.Split(value, ",")[0])
		}
		if value != "" {
			return value
		}
	}
	return r.RemoteAddr
}

func validateCSRF(r *http.Request) error {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return nil
	}
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return errors.New("csrf token is required")
	}
	header := strings.TrimSpace(r.Header.Get(csrfHeaderName))
	if header == "" || header != strings.TrimSpace(cookie.Value) {
		return errors.New("csrf token is invalid")
	}
	return nil
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(csrfCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return
	}
	token, err := randomHexToken(16)
	if err != nil {
		log.Printf("generate csrf token failed: %v", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		Secure:   cookieSecureEnabled(),
		SameSite: http.SameSiteLaxMode,
	})
}
