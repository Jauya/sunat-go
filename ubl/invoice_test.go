package ubl

import (
	"strings"
	"testing"
)

func TestBuildInvoiceXML(t *testing.T) {
	xmlBytes, err := BuildInvoiceXML(Invoice{
		TipoDoc:      "01",
		Serie:        "F001",
		Correlativo:  "1",
		FechaEmision: "2026-11-20",
		TipoMoneda:   "PEN",
		Company: Company{
			RUC:         "20000000001",
			RazonSocial: "EMPRESA DE PRUEBA SAC",
			Address: Address{
				Ubigeo:    "150101",
				Direccion: "AV. LARCO 123",
			},
		},
		Client: Client{
			TipoDoc:     "6",
			NumDoc:      "20100000001",
			RazonSocial: "CLIENTE TEST SAC",
		},
		MtoOperGravada: 100.00,
		MtoIGV:         18.00,
		TotalImpuestos: 18.00,
		ValorVenta:     100.00,
		MtoImpVenta:    118.00,
		Details: []SaleDetail{
			{
				UnidadMedida:   "NIU",
				Cantidad:       1,
				CodProducto:    "P001",
				Descripcion:    "Servicio de Consultoria Go",
				ValorUnitario:  100.00,
				PrecioUnitario: 118.00,
				ValorVenta:     100.00,
				IGV:            18.00,
				PorcentajeIGV:  18.00,
				TipAfeIGV:      "10",
			},
		},
		Legends: []Legend{
			{Code: "1000", Value: "SON CIENTO DIECIOCHO CON 00/100 SOLES"},
		},
	})
	if err != nil {
		t.Fatalf("BuildInvoiceXML failed: %v", err)
	}
	xmlStr := string(xmlBytes)
	for _, want := range []string{
		"<cbc:ID>F001-1</cbc:ID>",
		"<cbc:UBLVersionID>2.1</cbc:UBLVersionID>",
		"<cbc:PayableAmount currencyID=\"PEN\">118.00</cbc:PayableAmount>",
		"Servicio de Consultoria Go",
	} {
		if !strings.Contains(xmlStr, want) {
			t.Fatalf("expected XML to contain %q", want)
		}
	}
}
