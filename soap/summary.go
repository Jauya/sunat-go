package soap

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

// SendSummary submits a SummaryDocuments (RC) or VoidedDocuments (RA) ZIP and returns the SUNAT ticket.
func (c *Client) SendSummary(ctx context.Context, xmlFilename string, xmlContent []byte) (string, error) {
	zipBytes, err := PackageZIP(xmlFilename, xmlContent)
	if err != nil {
		return "", err
	}
	zipName := xmlFilename[:len(xmlFilename)-4] + ".zip"
	b64Zip := base64.StdEncoding.EncodeToString(zipBytes)

	envelope := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ser="http://service.sunat.gob.pe" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
  <soapenv:Header>
    <wsse:Security>
      <wsse:UsernameToken>
        <wsse:Username>%s%s</wsse:Username>
        <wsse:Password>%s</wsse:Password>
      </wsse:UsernameToken>
    </wsse:Security>
  </soapenv:Header>
  <soapenv:Body>
    <ser:sendSummary>
      <fileName>%s</fileName>
      <contentFile>%s</contentFile>
    </ser:sendSummary>
  </soapenv:Body>
</soapenv:Envelope>`, c.ruc, c.solUser, c.solPass, zipName, b64Zip)

	body, err := c.doSOAP(ctx, "urn:sendSummary", envelope)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Ticket string `xml:"Body>sendSummaryResponse>ticket"`
	}
	if err := xml.Unmarshal(body, &parsed); err != nil || parsed.Ticket == "" {
		return "", fmt.Errorf("unexpected sendSummary response: %s", string(body))
	}
	return parsed.Ticket, nil
}

// GetStatus polls SUNAT for the CDR ZIP of a previously submitted summary/voided ticket.
func (c *Client) GetStatus(ctx context.Context, ticket string) ([]byte, string, error) {
	envelope := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ser="http://service.sunat.gob.pe" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
  <soapenv:Header>
    <wsse:Security>
      <wsse:UsernameToken>
        <wsse:Username>%s%s</wsse:Username>
        <wsse:Password>%s</wsse:Password>
      </wsse:UsernameToken>
    </wsse:Security>
  </soapenv:Header>
  <soapenv:Body>
    <ser:getStatus>
      <ticket>%s</ticket>
    </ser:getStatus>
  </soapenv:Body>
</soapenv:Envelope>`, c.ruc, c.solUser, c.solPass, ticket)

	body, err := c.doSOAP(ctx, "urn:getStatus", envelope)
	if err != nil {
		return nil, "", err
	}
	var parsed struct {
		StatusCode string `xml:"Body>getStatusResponse>status>statusCode"`
		Content    string `xml:"Body>getStatusResponse>status>content"`
	}
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("unexpected getStatus response: %s", string(body))
	}
	if parsed.Content == "" {
		return nil, parsed.StatusCode, nil
	}
	cdrZip, err := base64.StdEncoding.DecodeString(parsed.Content)
	return cdrZip, parsed.StatusCode, err
}

func (c *Client) doSOAP(ctx context.Context, action, envelope string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBufferString(envelope))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", action)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sunat soap error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}
