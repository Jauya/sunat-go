package soap

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// EndpointBeta is SUNAT's public sandbox for SEE electronic billing tests.
	EndpointBeta = "https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService"
	// EndpointProd is SUNAT's production SEE endpoint.
	EndpointProd = "https://e-factura.sunat.gob.pe/ol-ti-itcpfegem/billService"
)

// Client sends signed UBL 2.1 ZIP payloads to SUNAT's SOAP billService.
type Client struct {
	endpoint   string
	ruc        string
	solUser    string
	solPass    string
	httpClient *http.Client
}

// NewClient initializes a SUNAT SOAP client.
func NewClient(endpoint, ruc, solUser, solPass string) *Client {
	if endpoint == "" {
		endpoint = EndpointBeta
	}
	return &Client{
		endpoint:   endpoint,
		ruc:        ruc,
		solUser:    solUser,
		solPass:    solPass,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// PackageZIP compresses a single XML file into an in-memory ZIP archive as required by SUNAT.
func PackageZIP(xmlFilename string, xmlContent []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(xmlFilename)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(xmlContent); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SendBill submits a zipped XML document (e.g. "20000000001-01-F001-1.xml") and returns the base64 CDR ZIP.
func (c *Client) SendBill(ctx context.Context, xmlFilename string, xmlContent []byte) ([]byte, error) {
	zipBytes, err := PackageZIP(xmlFilename, xmlContent)
	if err != nil {
		return nil, err
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
    <ser:sendBill>
      <fileName>%s</fileName>
      <contentFile>%s</contentFile>
    </ser:sendBill>
  </soapenv:Body>
</soapenv:Envelope>`, c.ruc, c.solUser, c.solPass, zipName, b64Zip)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBufferString(envelope))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "urn:sendBill")

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

	var parsed struct {
		AppResponse string `xml:"Body>sendBillResponse>applicationResponse"`
	}
	if err := xml.Unmarshal(body, &parsed); err != nil || parsed.AppResponse == "" {
		return nil, fmt.Errorf("unexpected soap response: %s", string(body))
	}
	return base64.StdEncoding.DecodeString(parsed.AppResponse)
}
