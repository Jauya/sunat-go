package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Jauya/sunat-go/soap"
	"github.com/Jauya/sunat-go/ubl"
	"github.com/Jauya/sunat-go/xmldsig"
)

func main() {
	key, cert := mustTestCert()
	signer := xmldsig.NewSigner(key, cert)
	client := soap.NewClient(soap.EndpointBeta, "20000000001", "MODDATOS", "moddatos")
	today := time.Now().Format("2006-01-02")

	company := ubl.Company{
		RUC:         "20000000001",
		RazonSocial: "EMPRESA DE PRUEBA SAC",
		Address:     ubl.Address{Ubigeo: "150101", Direccion: "AV. LARCO 123", Country: "PE"},
	}
	details := []ubl.SaleDetail{{
		UnidadMedida:   "NIU",
		Cantidad:       1,
		CodProducto:    "P001",
		Descripcion:    "Planta Ornamental Demo",
		ValorUnitario:  100.00,
		PrecioUnitario: 118.00,
		ValorVenta:     100.00,
		IGV:            18.00,
		PorcentajeIGV:  18.00,
		TipAfeIGV:      "10",
	}}
	legends := []ubl.Legend{{Code: "1000", Value: "SON CIENTO DIECIOCHO CON 00/100 SOLES"}}

	// 1. Factura (01)
	facXML, err := ubl.BuildInvoiceXML(ubl.Invoice{
		TipoDoc: "01", Serie: "F001", Correlativo: "1", FechaEmision: today, TipoMoneda: "PEN", TipoOperacion: "0101",
		Company: company,
		Client:  ubl.Client{TipoDoc: "6", NumDoc: "20100000001", RazonSocial: "CLIENTE PRUEBA SAC"},
		MtoOperGravada: 100, MtoIGV: 18, TotalImpuestos: 18, ValorVenta: 100, MtoImpVenta: 118,
		Details: details, Legends: legends,
	})
	mustSend(client, signer, "20000000001-01-F001-1.xml", facXML, err)

	// 2. Boleta de Venta (03)
	bolXML, err := ubl.BuildInvoiceXML(ubl.Invoice{
		TipoDoc: "03", Serie: "B001", Correlativo: "1", FechaEmision: today, TipoMoneda: "PEN", TipoOperacion: "0101",
		Company: company,
		Client:  ubl.Client{TipoDoc: "1", NumDoc: "12345678", RazonSocial: "CLIENTE FINAL"},
		MtoOperGravada: 100, MtoIGV: 18, TotalImpuestos: 18, ValorVenta: 100, MtoImpVenta: 118,
		Details: details, Legends: legends,
	})
	mustSend(client, signer, "20000000001-03-B001-1.xml", bolXML, err)

	// 3. Nota de Crédito (07)
	ncXML, err := ubl.BuildNoteXML(ubl.Note{
		TipoDoc: "07", Serie: "FC01", Correlativo: "1", FechaEmision: today, TipoMoneda: "PEN",
		TipDocAfectado: "01", NumDocAfectado: "F001-1", CodMotivo: "01", DesMotivo: "Anulacion de la operacion",
		Company: company,
		Client:  ubl.Client{TipoDoc: "6", NumDoc: "20100000001", RazonSocial: "CLIENTE PRUEBA SAC"},
		MtoOperGravada: 100, MtoIGV: 18, TotalImpuestos: 18, MtoImpVenta: 118,
		Details: details, Legends: legends,
	})
	mustSend(client, signer, "20000000001-07-FC01-1.xml", ncXML, err)

	// 4. Nota de Débito (08)
	ndXML, err := ubl.BuildNoteXML(ubl.Note{
		TipoDoc: "08", Serie: "FD01", Correlativo: "1", FechaEmision: today, TipoMoneda: "PEN",
		TipDocAfectado: "01", NumDocAfectado: "F001-1", CodMotivo: "01", DesMotivo: "Intereses por mora",
		Company: company,
		Client:  ubl.Client{TipoDoc: "6", NumDoc: "20100000001", RazonSocial: "CLIENTE PRUEBA SAC"},
		MtoOperGravada: 100, MtoIGV: 18, TotalImpuestos: 18, MtoImpVenta: 118,
		Details: details, Legends: legends,
	})
	mustSend(client, signer, "20000000001-08-FD01-1.xml", ndXML, err)
}

func mustSend(client *soap.Client, signer *xmldsig.Signer, filename string, xmlBytes []byte, buildErr error) {
	if buildErr != nil {
		log.Fatalf("%s build error: %v", filename, buildErr)
	}
	signed, err := signer.SignUBL(xmlBytes)
	if err != nil {
		log.Fatalf("%s sign error: %v", filename, err)
	}
	cdrZip, err := client.SendBill(context.Background(), filename, signed)
	if err != nil {
		log.Fatalf("%s sendBill error: %v", filename, err)
	}
	cdr, err := soap.ParseCDR(cdrZip)
	if err != nil {
		log.Fatalf("%s parseCDR error: %v", filename, err)
	}
	fmt.Printf("[%s] Accepted=%v Code=%s Desc=%s\n", filename, cdr.IsAccepted(), cdr.ResponseCode, cdr.Description)
}

func mustTestCert() (*rsa.PrivateKey, *x509.Certificate) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Beta Test Cert", Organization: []string{"EMPRESA BETA SAC"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		log.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		log.Fatal(err)
	}
	return key, cert
}
