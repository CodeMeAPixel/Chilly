package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Chilly/musicbot"
	"github.com/disgoorg/snowflake/v2"
)

const (
	sessionCookie  = "chilly_session"
	stateCookie    = "chilly_oauth_state"
	discordAPI     = "https://discord.com/api/v10"
	discordAuthURL = "https://discord.com/oauth2/authorize"
	oauthScopes    = "identify guilds"
)

type discordOAuth struct {
	clientID     string
	clientSecret string
	redirectURI  string
	http         *http.Client
}

func newDiscordOAuth(cfg musicbot.APIConfig) *discordOAuth {
	redirectURI := ""
	if cfg.PublicURL != "" {
		redirectURI = cfg.PublicURL + apiPrefix + "/auth/callback"
	}
	return &discordOAuth{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURI:  redirectURI,
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Server) oauthConfigured() bool {
	return s.oauth != nil && s.oauth.clientSecret != "" && s.oauth.redirectURI != ""
}

func (s *Server) setCookie(w http.ResponseWriter, name, value, path string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   s.cfg.CookieDomain,
		MaxAge:   int(maxAge.Seconds()),
		Secure:   s.cfg.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearCookie(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Domain:   s.cfg.CookieDomain,
		MaxAge:   -1,
		Secure:   s.cfg.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func safeRedirect(path string) string {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.Contains(path, "\\") {
		return "/"
	}
	return path
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.oauthConfigured() {
		writeError(w, http.StatusServiceUnavailable, "oauth_disabled", "discord login is not configured")
		return
	}
	state, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "failed to create state")
		return
	}
	redirect := safeRedirect(r.URL.Query().Get("redirect"))
	s.setCookie(w, stateCookie, state+"|"+url.QueryEscape(redirect), apiPrefix+"/auth", 10*time.Minute)

	q := url.Values{}
	q.Set("client_id", s.oauth.clientID)
	q.Set("redirect_uri", s.oauth.redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", oauthScopes)
	q.Set("state", state)
	q.Set("prompt", "none")
	http.Redirect(w, r, discordAuthURL+"?"+q.Encode(), http.StatusFound)
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	if !s.oauthConfigured() {
		writeError(w, http.StatusServiceUnavailable, "oauth_disabled", "discord login is not configured")
		return
	}

	cookie, err := r.Cookie(stateCookie)
	s.clearCookie(w, stateCookie, apiPrefix+"/auth")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_state", "login session expired, please try again")
		return
	}
	expected, redirectEnc, _ := strings.Cut(cookie.Value, "|")
	state := r.URL.Query().Get("state")
	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(state)) != 1 {
		writeError(w, http.StatusBadRequest, "invalid_state", "invalid login state, please try again")
		return
	}
	redirect, _ := url.QueryUnescape(redirectEnc)
	redirect = safeRedirect(redirect)

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		s.redirectToDashboard(w, r, "/login?error="+url.QueryEscape(errParam))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing_code", "missing authorization code")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	accessToken, err := s.oauth.exchange(ctx, code)
	if err != nil {
		slog.Warn("oauth code exchange failed", slog.Any("error", err))
		writeError(w, http.StatusBadGateway, "oauth_failed", "failed to log in with discord")
		return
	}

	var user struct {
		ID         snowflake.ID `json:"id"`
		Username   string       `json:"username"`
		GlobalName *string      `json:"global_name"`
		Avatar     *string      `json:"avatar"`
	}
	if err := s.oauth.get(ctx, accessToken, "/users/@me", &user); err != nil {
		slog.Warn("failed to fetch discord user", slog.Any("error", err))
		writeError(w, http.StatusBadGateway, "oauth_failed", "failed to fetch your discord profile")
		return
	}
	var guilds []SessionGuild
	if err := s.oauth.get(ctx, accessToken, "/users/@me/guilds", &guilds); err != nil {
		slog.Warn("failed to fetch discord guilds", slog.Any("error", err))
		writeError(w, http.StatusBadGateway, "oauth_failed", "failed to fetch your servers")
		return
	}
	go s.oauth.revoke(accessToken)

	token, err := s.sessions.Create(ctx, Session{
		UserID:     user.ID,
		Username:   user.Username,
		GlobalName: user.GlobalName,
		Avatar:     user.Avatar,
		Guilds:     guilds,
	}, s.cfg.SessionTTL)
	if err != nil {
		slog.Error("failed to create session", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "internal", "failed to create session")
		return
	}

	s.setCookie(w, sessionCookie, token, "/", s.cfg.SessionTTL)
	slog.Info("user logged in", slog.String("user_id", user.ID.String()))
	s.redirectToDashboard(w, r, redirect)
}

func (s *Server) redirectToDashboard(w http.ResponseWriter, r *http.Request, path string) {
	if s.cfg.DashboardURL == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	http.Redirect(w, r, s.cfg.DashboardURL+path, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if token := requestToken(r); token != "" {
		if err := s.sessions.Delete(r.Context(), token); err != nil {
			slog.Warn("failed to delete session", slog.Any("error", err))
		}
	}
	s.clearCookie(w, sessionCookie, "/")
	w.WriteHeader(http.StatusNoContent)
}

type meResponse struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
	Avatar     *string `json:"avatar"`
	Admin      bool    `json:"admin"`
	ExpiresAt  string  `json:"session_expires_at"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, sess *Session) {
	writeJSON(w, http.StatusOK, meResponse{
		ID:         sess.UserID.String(),
		Username:   sess.Username,
		GlobalName: sess.GlobalName,
		Avatar:     sess.Avatar,
		Admin:      s.isAdmin(sess.UserID),
		ExpiresAt:  sess.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (s *Server) isAdmin(userID snowflake.ID) bool {
	return slices.Contains(s.cfg.AdminUserIDs, userID)
}

func requestToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if c, err := r.Cookie(sessionCookie); err == nil {
		return c.Value
	}
	return ""
}

type authedHandler func(w http.ResponseWriter, r *http.Request, sess *Session)

func (s *Server) authed(h authedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := requestToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "login required")
			return
		}

		usesCookie := !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")
		if usesCookie && r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origin := r.Header.Get("Origin"); origin == "" || !s.originAllowed(origin) {
				writeError(w, http.StatusForbidden, "forbidden_origin", "request origin not allowed")
				return
			}
		}

		sess, err := s.sessions.Get(r.Context(), token)
		if err != nil {
			if !errors.Is(err, errSessionNotFound) {
				slog.Error("failed to load session", slog.Any("error", err))
				writeError(w, http.StatusInternalServerError, "internal", "failed to load session")
				return
			}
			if usesCookie {
				s.clearCookie(w, sessionCookie, "/")
			}
			writeError(w, http.StatusUnauthorized, "unauthorized", "session expired, please log in again")
			return
		}
		h(w, r, sess)
	})
}

func (o *discordOAuth) exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", o.redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discordAPI+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(o.clientID, o.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := o.do(req, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", errors.New("discord returned no access token")
	}
	return out.AccessToken, nil
}

func (o *discordOAuth) get(ctx context.Context, accessToken, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPI+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return o.do(req, out)
}

func (o *discordOAuth) revoke(accessToken string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	form := url.Values{}
	form.Set("token", accessToken)
	form.Set("token_type_hint", "access_token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discordAPI+"/oauth2/token/revoke", strings.NewReader(form.Encode()))
	if err != nil {
		return
	}
	req.SetBasicAuth(o.clientID, o.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = o.do(req, nil)
}

func (o *discordOAuth) do(req *http.Request, out any) error {
	req.Header.Set("User-Agent", "Chilly (https://github.com/CodeMeAPixel/Chilly, 1.0)")
	resp, err := o.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("discord %s %s: %s: %s", req.Method, req.URL.Path, resp.Status, body)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}
