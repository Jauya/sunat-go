package sire

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Jauya/sunat-go/auth"
)

const defaultBaseURL = "https://api-sire.sunat.gob.pe/v1/contribuyente/migeigv/libros"

// Client consumes SUNAT's SIRE (RVIE / RCE) REST APIs using OAuth2.
type Client struct {
	tokens     *auth.TokenProvider
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a SIRE REST API client backed by an OAuth2 TokenProvider.
func NewClient(tokens *auth.TokenProvider) *Client {
	return &Client{
		tokens:     tokens,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		baseURL:    defaultBaseURL,
	}
}

// GetPeriods fetches available RVIE ("rvie") or RCE ("rce") periods for a given codLibro.
// Common codLibro values: "140000" (RVIE - Ventas), "080000" (RCE - Compras).
func (c *Client) GetPeriods(ctx context.Context, libro string, codLibro string) ([]byte, error) {
	token, err := c.tokens.Token(ctx, auth.ScopeSIRE)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/%s/rvierce/padron/v1/periodos/%s", c.baseURL, libro, codLibro)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sire error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}
