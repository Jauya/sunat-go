package ubl

import (
	"strings"
	"testing"
)

func TestBuildNoteXML(t *testing.T) {
	xmlBytes, err := BuildNoteXML(Note{
		TipoDoc:        "07",
		Serie:          "FC01",
		Correlativo:    "1",
		FechaEmision:   "2026-11-20",
		TipoMoneda:     "PEN",
		TipDocAfectado: "01",
		NumDocAfectado: "F001-1",
		CodMotivo:      "01",
		DesMotivo:      "Anulacion de la operacion",
		Company: Company{
			RUC:         "20000000001",
			RazonSocial: "EMPRESA DE PRUEBA SAC",
			Address:     Address{Ubigeo: "150101", Direccion: "AV. LARCO 123"},
		},
		Client: Client{
			TipoDoc:     "6",
			NumDoc:      "20100000001",
			RazonSocial: "CLIENTE TEST SAC",
		},
		MtoOperGravada: 100.00,
		MtoIGV:         18.00,
		TotalImpuestos: 18.00,
		MtoImpVenta:    118.00,
		Details: []SaleDetail{
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
	})
	if err != nil {
		t.Fatalf("BuildNoteXML failed: %v", err)
	}
	s := string(xmlBytes)
	for _, want := range []string{
		`<CreditNote `,
		`<cbc:ID>FC01-1</cbc:ID>`,
		`<cbc:ResponseCode>01</cbc:ResponseCode>`,
		`<cbc:ID>F001-1</cbc:ID>`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("expected CreditNote XML to contain %q", want)
		}
	}
}
