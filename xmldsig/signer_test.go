package xmldsig

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestSignUBL(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test Cert"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}

	signer := NewSigner(key, cert)
	sample := []byte(`<?xml version="1.0" encoding="UTF-8"?><Invoice><ext:UBLExtensions><ext:UBLExtension><ext:ExtensionContent/></ext:ExtensionContent></ext:UBLExtension></ext:UBLExtensions></Invoice>`)
	// Fix sample tag closing for test
	sample = []byte(`<?xml version="1.0" encoding="UTF-8"?><Invoice><ext:UBLExtensions><ext:UBLExtension><ext:ExtensionContent/></ext:UBLExtension></ext:UBLExtensions></Invoice>`)

	signed, err := signer.SignUBL(sample)
	if err != nil {
		t.Fatalf("SignUBL failed: %v", err)
	}
	out := string(signed)
	for _, want := range []string{
		`<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Id="SignatureSP">`,
		`<ds:DigestValue>`,
		`<ds:SignatureValue>`,
		`<ds:X509Certificate>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected signed XML to contain %q", want)
		}
	}
}
