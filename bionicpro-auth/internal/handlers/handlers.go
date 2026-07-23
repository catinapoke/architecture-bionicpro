package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"bionicpro-auth/internal/config"
	"bionicpro-auth/internal/oidcauth"
	"bionicpro-auth/internal/profile"
	"bionicpro-auth/internal/session"
)

type Handler struct {
	cfg      config.Config
	oidc     *oidcauth.Client
	store    *session.Store
	profiles *profile.Store
}

func New(cfg config.Config, oidcClient *oidcauth.Client, store *session.Store, profiles *profile.Store) *Handler {
	return &Handler{
		cfg:      cfg,
		oidc:     oidcClient,
		store:    store,
		profiles: profiles,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /auth/login", h.login)
	mux.HandleFunc("GET /auth/callback", h.callback)
	mux.HandleFunc("POST /auth/logout", h.logout)
	mux.HandleFunc("GET /auth/me", h.withSession(h.me))
	mux.HandleFunc("GET /reports", h.withSession(h.reports))
	return h.cors(mux)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	state, err := randomString(32)
	if err != nil {
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}
	verifier, err := randomString(64)
	if err != nil {
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}
	challenge := pkceChallenge(verifier)

	if err := h.store.SavePending(state, verifier); err != nil {
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, h.oidc.AuthCodeURL(state, challenge), http.StatusFound)
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		http.Error(w, errMsg+": "+r.URL.Query().Get("error_description"), http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		http.Error(w, "missing state or code", http.StatusBadRequest)
		return
	}

	pending, ok := h.store.TakePending(state)
	if !ok {
		http.Error(w, "invalid or expired state", http.StatusBadRequest)
		return
	}

	tok, err := h.oidc.Exchange(r.Context(), code, pending.CodeVerifier)
	if err != nil {
		log.Printf("token exchange failed: %v", err)
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}

	idToken := oidcauth.IDToken(tok)
	h.persistProfile(r, idToken) // пропускаем ошибку

	sess, err := h.store.Create(tok.AccessToken, tok.RefreshToken, idToken, oidcauth.AccessExpiry(tok))
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, sess.ID, sess.ExpiresAt)
	http.Redirect(w, r, h.cfg.FrontendURL, http.StatusFound)
}

func (h *Handler) persistProfile(r *http.Request, idToken string) {
	if h.profiles == nil {
		return
	}
	p, ok := profile.FromIDToken(idToken)
	if !ok {
		return
	}
	if err := h.profiles.Upsert(r.Context(), p); err != nil {
		log.Printf("profile upsert failed: %v", err)
	}
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	idToken := ""
	if c, err := r.Cookie(h.cfg.CookieName); err == nil && c.Value != "" {
		if sess, ok := h.store.Get(c.Value); ok {
			idToken = sess.IDToken
		}
		h.store.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "logged_out",
		"logout_url": h.oidc.LogoutURL(idToken, h.cfg.FrontendURL),
	})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request, sess *session.Session) {
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"session_id":    sess.ID,
		"access_expiry": sess.Expiry.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) reports(w http.ResponseWriter, r *http.Request, sess *session.Session) {
	reportURL := strings.TrimRight(h.cfg.ReportURL, "/") + "/reports"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, reportURL, nil)
	if err != nil {
		http.Error(w, "failed to create report request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("report service request failed: %v", err)
		http.Error(w, "report service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("report response copy failed: %v", err)
	}
}

type reportAuthInfo struct {
	User       string
	Roles      []string
	RoleSource string
}

func accessTokenReportInfo(accessToken string) reportAuthInfo {
	info := reportAuthInfo{
		User:       "unknown",
		Roles:      []string{},
		RoleSource: "access_token",
	}

	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		info.RoleSource = "access_token_unavailable"
		return info
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		info.RoleSource = "access_token_unreadable"
		return info
	}

	var claims struct {
		Subject           string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		RealmAccess       struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		info.RoleSource = "access_token_unreadable"
		return info
	}

	if claims.PreferredUsername != "" {
		info.User = claims.PreferredUsername
	} else if claims.Subject != "" {
		info.User = claims.Subject
	}
	if claims.RealmAccess.Roles != nil {
		info.Roles = claims.RealmAccess.Roles
	}

	return info
}

func hasRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

type sessionHandler func(http.ResponseWriter, *http.Request, *session.Session)

func (h *Handler) withSession(next sessionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.cfg.CookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		sess, ok := h.store.Get(cookie.Value)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Refresh access token when expired (or about to expire).
		if time.Now().After(sess.Expiry.Add(-10 * time.Second)) {
			tok, err := h.oidc.Refresh(r.Context(), sess.RefreshToken)
			if err != nil {
				log.Printf("refresh failed: %v", err)
				h.store.Delete(sess.ID)
				h.clearSessionCookie(w)
				http.Error(w, "session expired", http.StatusUnauthorized)
				return
			}
			if err := h.store.UpdateTokens(sess.ID, tok.AccessToken, tok.RefreshToken, oidcauth.AccessExpiry(tok)); err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// Reload after update.
			sess, ok = h.store.Get(sess.ID)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Session fixation protection: rebind tokens to a new session id.
		rotated, err := h.store.Rotate(sess.ID)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.setSessionCookie(w, rotated.ID, rotated.ExpiresAt)

		next(w, r, rotated)
	}
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, id string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    id,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == h.cfg.CORSOrigin || origin == h.cfg.FrontendURL {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Set-Cookie")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
