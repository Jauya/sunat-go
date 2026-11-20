package soap

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// CDRResult represents the parsed SUNAT Constancia de Recepción (CDR).
type CDRResult struct {
	ID           string
	ResponseCode string // "0" means accepted
	Description  string
	Notes        []string
	RawXML       []byte
}

// IsAccepted reports whether SUNAT accepted the document without rejection ("0").
func (c *CDRResult) IsAccepted() bool {
	return c != nil && c.ResponseCode == "0"
}

// ParseCDR extracts the CDR XML from the ZIP returned by SUNAT and parses its status.
func ParseCDR(cdrZipBytes []byte) (*CDRResult, error) {
	zr, err := zip.NewReader(bytes.NewReader(cdrZipBytes), int64(len(cdrZipBytes)))
	if err != nil {
		return nil, fmt.Errorf("soap: invalid CDR zip: %w", err)
	}
	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(io.LimitReader(rc, 2<<20))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}

		var doc struct {
			ID       string   `xml:"ID"`
			Code     string   `xml:"DocumentResponse>Response>ResponseCode"`
			Desc     string   `xml:"DocumentResponse>Response>Description"`
			Notes    []string `xml:"Note"`
		}
		if err := xml.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("soap: parse CDR xml: %w", err)
		}
		return &CDRResult{
			ID:           doc.ID,
			ResponseCode: strings.TrimSpace(doc.Code),
			Description:  strings.TrimSpace(doc.Desc),
			Notes:        doc.Notes,
			RawXML:       raw,
		}, nil
	}
	return nil, fmt.Errorf("soap: no XML file found inside CDR zip")
}
