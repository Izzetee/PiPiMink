package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"PiPiMink/internal/database"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
)

const (
	sessionCookieName = "pipimink_session"
	stateCookieName   = "pipimink_oauth_state"
)

// initOAuth sets up the OIDC provider and OAuth2 config if configured.
// Called once at server start. Retries OIDC discovery up to 6 times with
// 5-second intervals to handle slow-starting identity providers (e.g. Authentik).
func (s *Server) initOAuth() {
	if !s.config.OAuthEnabled() {
		log.Println("OAuth not configured — running in API-key-only mode")
		return
	}

	// Set up session cookie encryption early so it's ready when OAuth completes.
	s.initSessionCookie()

	// Try OIDC discovery in the background with retries so the HTTP server
	// can start accepting requests immediately (non-OAuth routes work fine).
	go s.discoverOIDCWithRetry()
}

// initSessionCookie configures the securecookie instance for session encoding.
func (s *Server) initSessionCookie() {
	var hashKey, blockKey []byte
	if s.config.SessionSecret != "" {
		decoded, err := hex.DecodeString(s.config.SessionSecret)
		if err == nil && len(decoded) >= 32 {
			hashKey = decoded[:32]
			if len(decoded) >= 64 {
				blockKey = decoded[32:64]
			}
		}
	}
	if hashKey == nil {
		hashKey = securecookie.GenerateRandomKey(32)
		blockKey = securecookie.GenerateRandomKey(32)
		log.Println("Warning: SESSION_SECRET not set — using random key (sessions won't survive restart)")
	}
	s.secureCookie = securecookie.New(hashKey, blockKey)
}

// discoverOIDCWithRetry attempts OIDC provider discovery with retries.
func (s *Server) discoverOIDCWithRetry() {
	const maxRetries = 6
	const retryInterval = 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		provider, err := oidc.NewProvider(ctx, s.config.OAuthIssuerURL)
		cancel()

		if err == nil {
			s.finalizeOAuth(provider)
			log.Printf("OAuth configured: issuer=%s clientID=%s (attempt %d/%d)",
				s.config.OAuthIssuerURL, s.config.OAuthClientID, attempt, maxRetries)
			return
		}

		if attempt < maxRetries {
			log.Printf("OIDC discovery attempt %d/%d failed: %v — retrying in %s",
				attempt, maxRetries, err, retryInterval)
			time.Sleep(retryInterval)
		} else {
			log.Printf("Warning: OIDC discovery failed after %d attempts: %v — OAuth disabled",
				maxRetries, err)
		}
	}
}

// finalizeOAuth sets the oauth config and OIDC verifier once discovery succeeds.
func (s *Server) finalizeOAuth(provider *oidc.Provider) {
	scopes := strings.Fields(s.config.OAuthScopes)
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email", "groups"}
	}

	s.oauthConfig = &oauth2.Config{
		ClientID:     s.config.OAuthClientID,
		ClientSecret: s.config.OAuthClientSecret,
		RedirectURL:  s.config.OAuthRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	s.oidcVerifier = provider.Verifier(&oidc.Config{ClientID: s.config.OAuthClientID})
}

// handleAuthLogin redirects to the OAuth provider's authorization page.
func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if s.oauthConfig == nil {
		if s.config.OAuthEnabled() {
			http.Error(w, "OAuth is configured but the identity provider is not reachable yet — please try again in a few seconds", http.StatusServiceUnavailable)
		} else {
			http.Error(w, "OAuth not configured", http.StatusNotFound)
		}
		return
	}

	isHTTPS := strings.HasPrefix(r.Header.Get("X-Forwarded-Proto"), "https") ||
		strings.HasPrefix(s.config.OAuthRedirectURL, "https://")

	state := generateRandomState()
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS,
	})

	http.Redirect(w, r, s.oauthConfig.AuthCodeURL(state), http.StatusFound)
}

// handleAuthCallback processes the OAuth callback after user authenticates.
func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if s.oauthConfig == nil {
		http.Error(w, "OAuth not configured", http.StatusNotFound)
		return
	}

	// Validate state
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value == "" {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Check for OAuth error
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		desc := r.URL.Query().Get("error_description")
		log.Printf("OAuth error: %s — %s", errParam, desc)
		http.Error(w, "Authentication failed: "+errParam, http.StatusUnauthorized)
		return
	}

	// Exchange code for tokens
	ctx := r.Context()
	token, err := s.oauthConfig.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("OAuth token exchange error: %v", err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	// Extract and verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token in response", http.StatusInternalServerError)
		return
	}

	idToken, err := s.oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("ID token verification error: %v", err)
		http.Error(w, "Token verification failed", http.StatusUnauthorized)
		return
	}

	// Extract claims
	var claims struct {
		Sub    string   `json:"sub"`
		Email  string   `json:"email"`
		Name   string   `json:"name"`
		Groups []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("Error parsing ID token claims: %v", err)
		http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
		return
	}

	if claims.Email == "" {
		http.Error(w, "No email in token claims", http.StatusBadRequest)
		return
	}
	if claims.Name == "" {
		claims.Name = claims.Email
	}

	// Upsert user
	providerName := "Authentik"
	existing, _ := s.db.GetUserByEmail(claims.Email)
	now := time.Now().Format(time.RFC3339)

	user := database.UserRow{
		Name:             claims.Name,
		Email:            claims.Email,
		AuthSource:       "oauth",
		AuthProviderName: &providerName,
		Groups:           claims.Groups,
		LastLogin:        now,
	}

	if existing != nil {
		user.ID = existing.ID
		user.Role = existing.Role
		user.CreatedAt = existing.CreatedAt
		user.RequestCount = existing.RequestCount
		user.TokenUsage = existing.TokenUsage
	} else {
		if !s.config.OAuthAutoProvision {
			http.Error(w, "User not provisioned — contact an admin", http.StatusForbidden)
			return
		}
		user.ID = "user-" + uuid.New().String()[:8]
		user.CreatedAt = now

		// First user gets admin role; subsequent users get user role.
		users, err := s.db.GetUsers()
		if err != nil || len(users) == 0 {
			user.Role = "admin"
			log.Printf("First OAuth user %s granted admin role", claims.Email)
		} else {
			user.Role = "user"
		}
	}

	if err := s.db.UpsertUser(user); err != nil {
		log.Printf("Error upserting user %s: %v", claims.Email, err)
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	// Sync groups if present
	if len(claims.Groups) > 0 {
		s.syncGroupsFromClaims(claims.Groups)
	}

	// Write audit entry for new users
	if existing == nil {
		go func() {
			_ = s.db.SaveAuditEntry(database.AuditEntryRow{
				ID:      "audit-" + uuid.New().String()[:8],
				Actor:   "System",
				Action:  "user_created",
				Target:  claims.Name,
				Details: "User auto-provisioned via Authentik OAuth login",
			})
		}()
	}

	// Create session cookie
	sessionData := map[string]string{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
	}
	encoded, err := s.secureCookie.Encode(sessionCookieName, sessionData)
	if err != nil {
		log.Printf("Error encoding session cookie: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	isHTTPS := strings.HasPrefix(r.Header.Get("X-Forwarded-Proto"), "https") ||
		strings.HasPrefix(s.config.OAuthRedirectURL, "https://")

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS,
	})

	http.Redirect(w, r, "/console/", http.StatusFound)
}

// handleAuthLogout clears the session cookie.
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// handleAuthMe returns the current user from the session.
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	oauthEnabled := s.config.OAuthEnabled()

	user := getUserFromContext(r)
	if user != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"oauthEnabled":  oauthEnabled,
			"user":          user,
		})
		return
	}

	// Check if this is an API-key authenticated request
	if isAPIKeyAuth(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"oauthEnabled":  oauthEnabled,
			"user": map[string]interface{}{
				"id":    "api-key-admin",
				"name":  "Admin",
				"email": "admin@localhost",
				"role":  "admin",
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": false,
		"oauthEnabled":  oauthEnabled,
	})
}

// syncGroupsFromClaims ensures groups from OAuth claims exist in the DB.
func (s *Server) syncGroupsFromClaims(groupNames []string) {
	for _, name := range groupNames {
		groupID := "group-" + strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		_ = s.db.SaveGroup(database.GroupRow{
			ID:       groupID,
			Name:     name,
			Source:   "Authentik",
			Role:     "user",
			SyncedAt: time.Now().Format(time.RFC3339),
		})
	}
}

func generateRandomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
