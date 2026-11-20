package gre

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Jauya/sunat-go/auth"
	"github.com/Jauya/sunat-go/soap"
)

const defaultBaseURL = "https://api-cpe.sunat.gob.pe/v1/contribuyente/gem/comprobantes"

// Client consumes SUNAT's GRE (Guías de Remisión Electrónica) REST API via OAuth2.
type Client struct {
	tokens     *auth.TokenProvider
	httpClient *http.Client
	baseURL    string
}

// NewClient initializes a GRE REST API client.
func NewClient(tokens *auth.TokenProvider) *Client {
	return &Client{
		tokens:     tokens,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		baseURL:    defaultBaseURL,
	}
}

// SubmitResponse is returned by SUNAT when submitting a GRE document.
type SubmitResponse struct {
	NumTicket    string `json:"numTicket"`
	FecRecepcion string `json:"fecRecepcion"`
}

// Send submits a signed GRE XML (e.g. "20000000001-09-T001-1.xml") and returns the ticket.
func (c *Client) Send(ctx context.Context, xmlFilename string, xmlContent []byte) (*SubmitResponse, error) {
	token, err := c.tokens.Token(ctx, auth.ScopeCPE)
	if err != nil {
		return nil, err
	}
	zipBytes, err := soap.PackageZIP(xmlFilename, xmlContent)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(zipBytes)
	zipName := xmlFilename[:len(xmlFilename)-4] + ".zip"
	docID := xmlFilename[:len(xmlFilename)-4]

	payload, _ := json.Marshal(map[string]any{
		"archivo": map[string]string{
			"nomArchivo": zipName,
			"arcGreZip":  base64.StdEncoding.EncodeToString(zipBytes),
			"hashZip":    hex.EncodeToString(hash[:]),
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+docID, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gre error (HTTP %d): %s", resp.StatusCode, string(raw))
	}
	var res SubmitResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetTicket queries the processing status/CDR for a GRE submission ticket.
func (c *Client) GetTicket(ctx context.Context, numTicket string) ([]byte, error) {
	token, err := c.tokens.Token(ctx, auth.ScopeCPE)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/envios/"+numTicket, nil)
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

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gre ticket error (HTTP %d): %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}
