package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-gonic/gin"
)

type antigravityOAuthSession struct {
	Status      string
	AccessToken string
	TokenType   string
	Scope       string
	Error       string
	CreatedAt   time.Time
}

var (
	antigravityOAuthSessions = map[string]*antigravityOAuthSession{}
	antigravityOAuthLock     sync.Mutex
)

func init() {
	router.NewGroupRouter("/api/v1/channel/antigravity").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/oauth/start", http.MethodPost).
				Handle(startAntigravityOAuth),
		).
		AddRoute(
			router.NewRoute("/oauth/poll", http.MethodPost).
				Use(middleware.RequireJSON()).
				Handle(pollAntigravityOAuth),
		)

	router.NewGroupRouter("/api/v1/channel/antigravity").
		AddRoute(
			router.NewRoute("/oauth/callback", http.MethodGet).
				Handle(antigravityOAuthCallback),
		)
}

func cleanupAntigravitySessions() {
	deadline := time.Now().Add(-15 * time.Minute)
	for state, session := range antigravityOAuthSessions {
		if session.CreatedAt.Before(deadline) {
			delete(antigravityOAuthSessions, state)
		}
	}
}

func antigravityConfig() (clientID, clientSecret, authorizeURL, tokenURL, scope string) {
	clientID = strings.TrimSpace(os.Getenv("OCTOPUS_ANTIGRAVITY_CLIENT_ID"))
	if clientID == "" {
		clientID = strings.TrimSpace(os.Getenv("ANTIGRAVITY_CLIENT_ID"))
	}
	if clientID == "" {
		clientID = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	}
	clientSecret = strings.TrimSpace(os.Getenv("OCTOPUS_ANTIGRAVITY_CLIENT_SECRET"))
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(os.Getenv("ANTIGRAVITY_CLIENT_SECRET"))
	}
	if clientSecret == "" {
		clientSecret = "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"
	}
	authorizeURL = strings.TrimSpace(os.Getenv("OCTOPUS_ANTIGRAVITY_AUTHORIZE_URL"))
	if authorizeURL == "" {
		authorizeURL = strings.TrimSpace(os.Getenv("ANTIGRAVITY_AUTHORIZE_URL"))
	}
	if authorizeURL == "" {
		authorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	tokenURL = strings.TrimSpace(os.Getenv("OCTOPUS_ANTIGRAVITY_TOKEN_URL"))
	if tokenURL == "" {
		tokenURL = strings.TrimSpace(os.Getenv("ANTIGRAVITY_TOKEN_URL"))
	}
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}
	scope = strings.TrimSpace(os.Getenv("OCTOPUS_ANTIGRAVITY_SCOPE"))
	if scope == "" {
		scope = strings.TrimSpace(os.Getenv("ANTIGRAVITY_SCOPE"))
	}
	if scope == "" {
		scope = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	}
	return
}

func buildAPIPublicBase(c *gin.Context) string {
	if apiBaseURL, err := op.SettingGetString(model.SettingKeyAPIBaseURL); err == nil {
		trimmed := strings.TrimSpace(apiBaseURL)
		if trimmed != "" {
			return strings.TrimRight(trimmed, "/")
		}
	}
	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
	}
	host := c.Request.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func randomOAuthState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func startAntigravityOAuth(c *gin.Context) {
	clientID, _, authorizeURL, _, scope := antigravityConfig()
	if clientID == "" {
		resp.Error(c, http.StatusBadRequest, "antigravity oauth is not configured: missing client_id")
		return
	}
	state, err := randomOAuthState()
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	redirectURI := buildAPIPublicBase(c) + "/api/v1/channel/antigravity/oauth/callback"

	u, err := url.Parse(authorizeURL)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "invalid antigravity authorize url")
		return
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", scope)
	q.Set("state", state)
	u.RawQuery = q.Encode()

	antigravityOAuthLock.Lock()
	defer antigravityOAuthLock.Unlock()
	cleanupAntigravitySessions()
	antigravityOAuthSessions[state] = &antigravityOAuthSession{
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	resp.Success(c, gin.H{
		"state":    state,
		"auth_url": u.String(),
	})
}

type antigravityPollRequest struct {
	State string `json:"state"`
}

func pollAntigravityOAuth(c *gin.Context) {
	var req antigravityPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if strings.TrimSpace(req.State) == "" {
		resp.Error(c, http.StatusBadRequest, "state is required")
		return
	}

	antigravityOAuthLock.Lock()
	session, ok := antigravityOAuthSessions[req.State]
	if !ok {
		antigravityOAuthLock.Unlock()
		resp.Error(c, http.StatusNotFound, "oauth session not found or expired")
		return
	}
	copySession := *session
	if session.Status == "authorized" || session.Status == "failed" {
		delete(antigravityOAuthSessions, req.State)
	}
	antigravityOAuthLock.Unlock()

	resp.Success(c, gin.H{
		"status":       copySession.Status,
		"access_token": copySession.AccessToken,
		"token_type":   copySession.TokenType,
		"scope":        copySession.Scope,
		"error":        copySession.Error,
	})
}

func antigravityOAuthCallback(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		c.String(http.StatusBadRequest, "missing state")
		return
	}

	antigravityOAuthLock.Lock()
	session, ok := antigravityOAuthSessions[state]
	antigravityOAuthLock.Unlock()
	if !ok {
		c.String(http.StatusBadRequest, "session not found or expired")
		return
	}

	if errText := c.Query("error"); errText != "" {
		msg := c.Query("error_description")
		if msg == "" {
			msg = errText
		}
		antigravityOAuthLock.Lock()
		session.Status = "failed"
		session.Error = msg
		antigravityOAuthLock.Unlock()
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body><h3>Authorization failed</h3><p>You can close this window.</p></body></html>"))
		return
	}

	code := c.Query("code")
	if code == "" {
		c.String(http.StatusBadRequest, "missing code")
		return
	}

	clientID, clientSecret, _, tokenURL, _ := antigravityConfig()
	if clientID == "" || clientSecret == "" {
		antigravityOAuthLock.Lock()
		session.Status = "failed"
		session.Error = "antigravity oauth is not configured: missing client_id/client_secret"
		antigravityOAuthLock.Unlock()
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body><h3>Authorization failed</h3><p>Server oauth config is missing.</p></body></html>"))
		return
	}

	redirectURI := buildAPIPublicBase(c) + "/api/v1/channel/antigravity/oauth/callback"
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		antigravityOAuthLock.Lock()
		session.Status = "failed"
		session.Error = err.Error()
		antigravityOAuthLock.Unlock()
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body><h3>Authorization failed</h3><p>You can close this window.</p></body></html>"))
		return
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		antigravityOAuthLock.Lock()
		session.Status = "failed"
		session.Error = err.Error()
		antigravityOAuthLock.Unlock()
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body><h3>Authorization failed</h3><p>You can close this window.</p></body></html>"))
		return
	}
	defer httpResp.Body.Close()

	payload, _ := io.ReadAll(httpResp.Body)
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	_ = json.Unmarshal(payload, &tokenResp)

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 || tokenResp.AccessToken == "" {
		errMsg := tokenResp.ErrorDesc
		if errMsg == "" {
			errMsg = tokenResp.Error
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("token exchange failed: %s", string(payload))
		}
		antigravityOAuthLock.Lock()
		session.Status = "failed"
		session.Error = errMsg
		antigravityOAuthLock.Unlock()
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body><h3>Authorization failed</h3><p>You can close this window.</p></body></html>"))
		return
	}

	antigravityOAuthLock.Lock()
	session.Status = "authorized"
	session.AccessToken = tokenResp.AccessToken
	session.TokenType = tokenResp.TokenType
	session.Scope = tokenResp.Scope
	antigravityOAuthLock.Unlock()

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body><h3>Authorization successful</h3><p>You can close this window and return to Octopus.</p><script>setTimeout(function(){window.close();}, 800);</script></body></html>"))
}
