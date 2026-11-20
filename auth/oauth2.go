package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// ScopeSIRE is the OAuth2 scope for SIRE (RVIE / RCE).
	ScopeSIRE = "https://api-sire.sunat.gob.pe"
	// ScopeCPE is the OAuth2 scope for GRE / CPE REST APIs.
	ScopeCPE = "https://api-cpe.sunat.gob.pe"

	defaultAuthBaseURL = "https://api-seguridad.sunat.gob.pe/v1/clientessol"
)

// Credentials holds the SUNAT SOL and OAuth2 client credentials.
type Credentials struct {
	RUC          string
	SolUser      string
	SolPass      string
	ClientID     string
	ClientSecret string
}

// TokenResponse represents the JSON payload returned by SUNAT's OAuth2 endpoint.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// TokenProvider caches and automatically refreshes OAuth2 Bearer tokens per scope.
type TokenProvider struct {
	creds      Credentials
	httpClient *http.Client
	baseURL    string

	mu     sync.Mutex
	tokens map[string]cachedToken
}

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

// NewTokenProvider creates a thread-safe SUNAT OAuth2 token provider.
func NewTokenProvider(creds Credentials, opts ...Option) *TokenProvider {
	tp := &TokenProvider{
		creds:      creds,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    defaultAuthBaseURL,
		tokens:     make(map[string]cachedToken),
	}
	for _, opt := range opts {
		opt(tp)
	}
	return tp
}

// Option customizes a TokenProvider.
type Option func(*TokenProvider)

// WithHTTPClient overrides the HTTP client used for token requests.
func WithHTTPClient(c *http.Client) Option {
	return func(tp *TokenProvider) {
		if c != nil {
			tp.httpClient = c
		}
	}
}

// WithBaseURL overrides the OAuth2 base URL (useful for unit tests).
func WithBaseURL(u string) Option {
	return func(tp *TokenProvider) {
		if u != "" {
			tp.baseURL = strings.TrimRight(u, "/")
		}
	}
}

// Token returns a valid Bearer token for the requested scope, refreshing if expired.
func (tp *TokenProvider) Token(ctx context.Context, scope string) (string, error) {
	if scope == "" {
		scope = ScopeSIRE
	}
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if c, ok := tp.tokens[scope]; ok && time.Now().Add(60*time.Second).Before(c.expiresAt) {
		return c.accessToken, nil
	}

	endpoint := fmt.Sprintf("%s/%s/oauth2/token/", tp.baseURL, url.PathEscape(tp.creds.ClientID))
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("scope", scope)
	form.Set("client_id", tp.creds.ClientID)
	form.Set("client_secret", tp.creds.ClientSecret)
	form.Set("username", tp.creds.RUC+tp.creds.SolUser)
	form.Set("password", tp.creds.SolPass)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tp.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("sunat oauth2 error (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed TokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("sunat oauth2 decode error: %w", err)
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("sunat oauth2 returned empty access_token")
	}

	ttl := time.Duration(parsed.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 3600 * time.Second
	}
	tp.tokens[scope] = cachedToken{
		accessToken: parsed.AccessToken,
		expiresAt:   time.Now().Add(ttl),
	}
	return parsed.AccessToken, nil
}
