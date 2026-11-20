package xmldsig

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// Signer signs SUNAT UBL 2.1 XML documents using RSA-SHA256 enveloped XMLDSig.
type Signer struct {
	key  *rsa.PrivateKey
	cert *x509.Certificate
}

// NewSignerFromPEM loads an RSA private key and X.509 certificate from PEM-encoded bytes.
func NewSignerFromPEM(certPEM, keyPEM []byte) (*Signer, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, errors.New("xmldsig: invalid certificate PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("xmldsig: parse cert: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, errors.New("xmldsig: invalid private key PEM")
	}
	var rsaKey *rsa.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err == nil {
		rsaKey = k
	} else if kAny, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes); err == nil {
		var ok bool
		rsaKey, ok = kAny.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("xmldsig: PKCS8 key is not RSA")
		}
	} else {
		return nil, fmt.Errorf("xmldsig: unsupported private key format")
	}

	return &Signer{key: rsaKey, cert: cert}, nil
}

// NewSigner creates a Signer from an already parsed RSA key and X.509 certificate.
func NewSigner(key *rsa.PrivateKey, cert *x509.Certificate) *Signer {
	return &Signer{key: key, cert: cert}
}

// SignUBL inserts an enveloped XMLDSig signature inside <ext:ExtensionContent> of a UBL 2.1 document.
func (s *Signer) SignUBL(xmlBytes []byte) ([]byte, error) {
	xmlStr := string(xmlBytes)
	const emptyExt = "<ext:ExtensionContent/>"
	const pairExt = "<ext:ExtensionContent></ext:ExtensionContent>"

	var target string
	if strings.Contains(xmlStr, emptyExt) {
		target = emptyExt
	} else if strings.Contains(xmlStr, pairExt) {
		target = pairExt
	} else {
		return nil, errors.New("xmldsig: <ext:ExtensionContent> placeholder not found in UBL XML")
	}

	// Canonical form of the document excluding the XML declaration for digest calculation.
	docForDigest := strings.Replace(xmlStr, emptyExt, pairExt, 1)
	if idx := strings.Index(docForDigest, "?>"); idx != -1 {
		docForDigest = strings.TrimLeft(docForDigest[idx+2:], "\r\n ")
	}

	digestSum := sha256.Sum256([]byte(docForDigest))
	digestB64 := base64.StdEncoding.EncodeToString(digestSum[:])

	signedInfoForHash := fmt.Sprintf(`<ds:SignedInfo xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2"><ds:CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"></ds:CanonicalizationMethod><ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"></ds:SignatureMethod><ds:Reference URI=""><ds:Transforms><ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"></ds:Transform></ds:Transforms><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"></ds:DigestMethod><ds:DigestValue>%s</ds:DigestValue></ds:Reference></ds:SignedInfo>`, digestB64)

	signedInfoSum := sha256.Sum256([]byte(signedInfoForHash))
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, signedInfoSum[:])
	if err != nil {
		return nil, fmt.Errorf("xmldsig: sign failed: %w", err)
	}
	sigB64 := base64.StdEncoding.EncodeToString(sigBytes)
	certB64 := base64.StdEncoding.EncodeToString(s.cert.Raw)

	signedInfoInner := fmt.Sprintf(`<ds:SignedInfo><ds:CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"></ds:CanonicalizationMethod><ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"></ds:SignatureMethod><ds:Reference URI=""><ds:Transforms><ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"></ds:Transform></ds:Transforms><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"></ds:DigestMethod><ds:DigestValue>%s</ds:DigestValue></ds:Reference></ds:SignedInfo>`, digestB64)

	signatureBlock := fmt.Sprintf(`<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Id="SignatureSP">%s<ds:SignatureValue>%s</ds:SignatureValue><ds:KeyInfo><ds:X509Data><ds:X509Certificate>%s</ds:X509Certificate></ds:X509Data></ds:KeyInfo></ds:Signature>`,
		signedInfoInner, sigB64, certB64)

	signed := bytes.Replace(xmlBytes, []byte(target), []byte("<ext:ExtensionContent>"+signatureBlock+"</ext:ExtensionContent>"), 1)
	return signed, nil
}
