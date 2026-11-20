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

	inv := ubl.Invoice{
		TipoDoc:        "01",
		Serie:          "F001",
		Correlativo:    "1",
		FechaEmision:   time.Now().Format("2006-01-02"),
		TipoMoneda:     "PEN",
		TipoOperacion:  "0101",
		Company: ubl.Company{
			RUC:         "20000000001",
			RazonSocial: "EMPRESA DE PRUEBA SAC",
			Address:     ubl.Address{Ubigeo: "150101", Direccion: "AV. LARCO 123", Country: "PE"},
		},
		Client: ubl.Client{
			TipoDoc:     "6",
			NumDoc:      "20100000001",
			RazonSocial: "CLIENTE PRUEBA SAC",
		},
		MtoOperGravada: 100.00,
		MtoIGV:         18.00,
		TotalImpuestos: 18.00,
		ValorVenta:     100.00,
		MtoImpVenta:    118.00,
		Details: []ubl.SaleDetail{
			{
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
			},
		},
		Legends: []ubl.Legend{{Code: "1000", Value: "SON CIENTO DIECIOCHO CON 00/100 SOLES"}},
	}

	xmlBytes, err := ubl.BuildInvoiceXML(inv)
	if err != nil {
		log.Fatal(err)
	}

	signedXML, err := xmldsig.NewSigner(key, cert).SignUBL(xmlBytes)
	if err != nil {
		log.Fatal(err)
	}

	client := soap.NewClient(soap.EndpointBeta, "20000000001", "MODDATOS", "moddatos")
	cdrZip, err := client.SendBill(context.Background(), "20000000001-01-F001-1.xml", signedXML)
	if err != nil {
		log.Fatalf("SendBill Beta error: %v", err)
	}

	cdr, err := soap.ParseCDR(cdrZip)
	if err != nil {
		log.Fatalf("ParseCDR error: %v", err)
	}
	fmt.Printf("Accepted=%v Code=%s Desc=%s Notes=%v\n", cdr.IsAccepted(), cdr.ResponseCode, cdr.Description, cdr.Notes)
}
