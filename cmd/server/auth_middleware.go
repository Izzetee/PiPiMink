package server

import (
	"context"
	"net/http"
	"strings"

	"PiPiMink/internal/database"
)

type contextKey string

const contextKeyUser contextKey = "user"
const contextKeyAPIKeyAuth contextKey = "apiKeyAuth"
const contextKeyAuthLevel contextKey = "authLevel"
const contextKeyUserID contextKey = "userID"

// authLevel represents the authentication tier achieved by a request.
type authLevel int

const (
	// AuthPublic — no credentials provided or only public endpoints.
	AuthPublic authLevel = iota
	// AuthUser — authenticated as a regular user (session, Bearer token).
	AuthUser
	// AuthAdmin — authenticated as admin (X-API-Key or admin-role user).
	AuthAdmin
)

// getUserFromContext returns the authenticated user from the request context.
// Returns nil if the request was authenticated via API key (backward compat).
func getUserFromContext(r *http.Request) *database.UserRow {
	if u, ok := r.Context().Value(contextKeyUser).(*database.UserRow); ok {
		return u
	}
	return nil
}

// isAPIKeyAuth returns true if the request was authenticated via X-API-Key.
func isAPIKeyAuth(r *http.Request) bool {
	v, _ := r.Context().Value(contextKeyAPIKeyAuth).(bool)
	return v
}

// getAuthLevel returns the auth level achieved by the request.
func getAuthLevel(r *http.Request) authLevel {
	if v, ok := r.Context().Value(contextKeyAuthLevel).(authLevel); ok {
		return v
	}
	return AuthPublic
}

// getUserID returns the user identifier for the request.
// Returns "anonymous" for unauthenticated requests, "admin:api-key" for global admin key.
func getUserID(r *http.Request) string {
	if v, ok := r.Context().Value(contextKeyUserID).(string); ok && v != "" {
		return v
	}
	return "anonymous"
}

// authMiddleware handles authentication for all requests.
// It identifies the caller, determines the required auth level for the path,
// and enforces access control.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		// ── Phase 1: Always-public paths (no auth needed) ──
		if isPublicPath(path) {
			ctx := context.WithValue(r.Context(), contextKeyAuthLevel, AuthPublic)
			ctx = context.WithValue(ctx, contextKeyUserID, "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// ── Phase 2: Identify the caller ──
		level, userID, user := s.identifyCaller(r)

		// ── Phase 3: Determine required level for this path ──
		required := s.requiredLevel(path, method)

		// ── Phase 4: Enforce ──
		if level < required {
			if s.config.OAuthEnabled() && (strings.HasPrefix(path, "/console") || path == "/") {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// ── Phase 5: Set context values and proceed ──
		ctx := r.Context()
		ctx = context.WithValue(ctx, contextKeyAuthLevel, level)
		ctx = context.WithValue(ctx, contextKeyUserID, userID)
		if level >= AuthAdmin && user == nil {
			// X-API-Key auth — set the flag for backward compat
			ctx = context.WithValue(ctx, contextKeyAPIKeyAuth, true)
		}
		if user != nil {
			ctx = context.WithValue(ctx, contextKeyUser, user)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isPublicPath returns true for paths that never require authentication.
func isPublicPath(path string) bool {
	// Specific auth flow endpoints (but NOT /auth/tokens — those require auth)
	if path == "/auth/login" || path == "/auth/callback" || path == "/auth/logout" || path == "/auth/me" {
		return true
	}
	if strings.HasPrefix(path, "/swagger/") ||
		strings.HasPrefix(path, "/assets/") ||
		path == "/admin/status" ||
		path == "/metrics" ||
		path == "/favicon.ico" {
		return true
	}
	return false
}

// identifyCaller tries all authentication methods and returns the achieved level,
// user ID, and user object (if available). Does NOT return errors — an invalid
// credential results in (AuthPublic, "anonymous", nil) plus a 401 written to w
// via a sentinel return.
func (s *Server) identifyCaller(r *http.Request) (authLevel, string, *database.UserRow) {
	// 1. Check X-API-Key header (global admin key)
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		if s.config.AdminAPIKey != "" && apiKey == s.config.AdminAPIKey {
			return AuthAdmin, "admin:api-key", nil
		}
		// Invalid key — still return Public so the middleware can 401
		return AuthPublic, "anonymous", nil
	}

	// 2. Check Authorization: Bearer <token>
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != "" {
			tokenHash := database.HashToken(token)
			user, err := s.db.GetUserByAPIToken(tokenHash)
			if err == nil && user != nil {
				lvl := AuthUser
				if user.Role == "admin" {
					lvl = AuthAdmin
				}
				return lvl, user.ID, user
			}
		}
		// Invalid Bearer token — still return Public
		return AuthPublic, "anonymous", nil
	}

	// 3. Check session cookie (OAuth flow)
	if s.secureCookie != nil {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			var sessionData map[string]string
			if err := s.secureCookie.Decode(sessionCookieName, cookie.Value, &sessionData); err == nil {
				email := sessionData["email"]
				if email != "" {
					user, err := s.db.GetUserByEmail(email)
					if err == nil && user != nil {
						lvl := AuthUser
						if user.Role == "admin" {
							lvl = AuthAdmin
						}
						return lvl, user.ID, user
					}
				}
			}
		}
	}

	// 4. No credentials — anonymous
	return AuthPublic, "anonymous", nil
}

// requiredLevel determines the minimum auth level needed for a given path and method.
func (s *Server) requiredLevel(path, method string) authLevel {
	// Chat / API endpoints — configurable
	if path == "/chat" ||
		strings.HasPrefix(path, "/v1/") ||
		strings.HasPrefix(path, "/api/") {
		if s.config.RequireAuthForChat {
			return AuthUser
		}
		return AuthPublic
	}

	// Read-only model endpoints — follow chat auth policy
	if strings.HasPrefix(path, "/models") {
		// Mutating model operations require admin
		if method != "GET" {
			return AuthAdmin
		}
		// GET /models, GET /models/{name}/benchmarks, etc.
		if s.config.RequireAuthForChat {
			return AuthUser
		}
		return AuthPublic
	}

	// Benchmark leaderboard — follow chat auth policy
	if strings.HasPrefix(path, "/benchmarks/") {
		if s.config.RequireAuthForChat {
			return AuthUser
		}
		return AuthPublic
	}

	// Console / root — require user auth only when OAuth is configured
	if strings.HasPrefix(path, "/console") || path == "/" {
		if s.config.OAuthEnabled() {
			return AuthUser
		}
		return AuthPublic
	}

	// Token management — user auth
	if strings.HasPrefix(path, "/auth/tokens") {
		return AuthUser
	}

	// All /admin/* endpoints (except /admin/status which is public)
	if strings.HasPrefix(path, "/admin/") {
		return AuthAdmin
	}

	// Provider management — admin
	if strings.HasPrefix(path, "/providers") {
		return AuthAdmin
	}

	// Default: admin for safety
	return AuthAdmin
}
