package ubl

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// Note models a Nota de Crédito (07) or Nota de Débito (08) in UBL 2.1.
type Note struct {
	TipoDoc          string // "07" Crédito, "08" Débito
	Serie            string // e.g. "FC01", "FD01", "BC01"
	Correlativo      string
	FechaEmision     string // YYYY-MM-DD
	TipoMoneda       string // "PEN", "USD"
	TipDocAfectado   string // "01" Factura, "03" Boleta
	NumDocAfectado   string // e.g. "F001-1"
	CodMotivo        string // Catálogo 09 (NC) or 10 (ND), e.g. "01"
	DesMotivo        string // e.g. "Anulación de la operación"
	Company          Company
	Client           Client
	MtoOperGravada   float64
	MtoIGV           float64
	TotalImpuestos   float64
	MtoImpVenta      float64
	Details          []SaleDetail
	Legends          []Legend
}

// BuildNoteXML serializes a CreditNote (07) or DebitNote (08) into UBL 2.1 XML.
func BuildNoteXML(n Note) ([]byte, error) {
	if n.TipoDoc != "07" && n.TipoDoc != "08" {
		return nil, fmt.Errorf("ubl: TipoDoc must be 07 (CreditNote) or 08 (DebitNote)")
	}
	rootTag := "CreditNote"
	schemaNS := "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2"
	lineTag := "CreditNoteLine"
	qtyTag := "CreditedQuantity"
	totalTag := "LegalMonetaryTotal"
	if n.TipoDoc == "08" {
		rootTag = "DebitNote"
		schemaNS = "urn:oasis:names:specification:ubl:schema:xsd:DebitNote-2"
		lineTag = "DebitNoteLine"
		qtyTag = "DebitedQuantity"
		totalTag = "RequestedMonetaryTotal"
	}

	currency := n.TipoMoneda
	if currency == "" {
		currency = "PEN"
	}
	country := n.Company.Address.Country
	if country == "" {
		country = "PE"
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, `<%s xmlns="%s" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2" xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2">`, rootTag, schemaNS)
	buf.WriteString(`<ext:UBLExtensions><ext:UBLExtension><ext:ExtensionContent/></ext:UBLExtension></ext:UBLExtensions>`)
	buf.WriteString(`<cbc:UBLVersionID>2.1</cbc:UBLVersionID><cbc:CustomizationID>2.0</cbc:CustomizationID>`)
	fmt.Fprintf(&buf, `<cbc:ID>%s-%s</cbc:ID><cbc:IssueDate>%s</cbc:IssueDate>`, esc(n.Serie), esc(n.Correlativo), esc(n.FechaEmision))
	for _, leg := range n.Legends {
		fmt.Fprintf(&buf, `<cbc:Note languageLocaleID="%s">%s</cbc:Note>`, esc(leg.Code), esc(leg.Value))
	}
	fmt.Fprintf(&buf, `<cbc:DocumentCurrencyCode>%s</cbc:DocumentCurrencyCode>`, esc(currency))
	fmt.Fprintf(&buf, `<cac:DiscrepancyResponse><cbc:ReferenceID>%s</cbc:ReferenceID><cbc:ResponseCode>%s</cbc:ResponseCode><cbc:Description>%s</cbc:Description></cac:DiscrepancyResponse>`,
		esc(n.NumDocAfectado), esc(n.CodMotivo), esc(n.DesMotivo))
	fmt.Fprintf(&buf, `<cac:BillingReference><cac:InvoiceDocumentReference><cbc:ID>%s</cbc:ID><cbc:DocumentTypeCode>%s</cbc:DocumentTypeCode></cac:InvoiceDocumentReference></cac:BillingReference>`,
		esc(n.NumDocAfectado), esc(n.TipDocAfectado))

	fmt.Fprintf(&buf, `<cac:AccountingSupplierParty><cac:Party><cac:PartyIdentification><cbc:ID schemeID="6">%s</cbc:ID></cac:PartyIdentification><cac:PartyLegalEntity><cbc:RegistrationName>%s</cbc:RegistrationName><cac:RegistrationAddress><cbc:ID>%s</cbc:ID><cbc:AddressTypeCode>0000</cbc:AddressTypeCode><cac:AddressLine><cbc:Line>%s</cbc:Line></cac:AddressLine><cac:Country><cbc:IdentificationCode>%s</cbc:IdentificationCode></cac:Country></cac:RegistrationAddress></cac:PartyLegalEntity></cac:Party></cac:AccountingSupplierParty>`,
		esc(n.Company.RUC), esc(n.Company.RazonSocial), esc(n.Company.Address.Ubigeo), esc(n.Company.Address.Direccion), esc(country))
	fmt.Fprintf(&buf, `<cac:AccountingCustomerParty><cac:Party><cac:PartyIdentification><cbc:ID schemeID="%s">%s</cbc:ID></cac:PartyIdentification><cac:PartyLegalEntity><cbc:RegistrationName>%s</cbc:RegistrationName></cac:PartyLegalEntity></cac:Party></cac:AccountingCustomerParty>`,
		esc(n.Client.TipoDoc), esc(n.Client.NumDoc), esc(n.Client.RazonSocial))

	fmt.Fprintf(&buf, `<cac:TaxTotal><cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount><cac:TaxSubtotal><cbc:TaxableAmount currencyID="%s">%.2f</cbc:TaxableAmount><cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount><cac:TaxCategory><cac:TaxScheme><cbc:ID>1000</cbc:ID><cbc:Name>IGV</cbc:Name><cbc:TaxTypeCode>VAT</cbc:TaxTypeCode></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal>`,
		esc(currency), n.TotalImpuestos, esc(currency), n.MtoOperGravada, esc(currency), n.MtoIGV)

	fmt.Fprintf(&buf, `<cac:%s><cbc:PayableAmount currencyID="%s">%.2f</cbc:PayableAmount></cac:%s>`,
		totalTag, esc(currency), n.MtoImpVenta, totalTag)

	for i, d := range n.Details {
		fmt.Fprintf(&buf, `<cac:%s><cbc:ID>%d</cbc:ID><cbc:%s unitCode="%s">%.2f</cbc:%s><cbc:LineExtensionAmount currencyID="%s">%.2f</cbc:LineExtensionAmount><cac:PricingReference><cac:AlternativeConditionPrice><cbc:PriceAmount currencyID="%s">%.2f</cbc:PriceAmount><cbc:PriceTypeCode>01</cbc:PriceTypeCode></cac:AlternativeConditionPrice></cac:PricingReference><cac:TaxTotal><cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount><cac:TaxSubtotal><cbc:TaxableAmount currencyID="%s">%.2f</cbc:TaxableAmount><cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount><cac:TaxCategory><cbc:Percent>%.2f</cbc:Percent><cbc:TaxExemptionReasonCode>%s</cbc:TaxExemptionReasonCode><cac:TaxScheme><cbc:ID>1000</cbc:ID><cbc:Name>IGV</cbc:Name><cbc:TaxTypeCode>VAT</cbc:TaxTypeCode></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal><cac:Item><cbc:Description>%s</cbc:Description><cac:SellersItemIdentification><cbc:ID>%s</cbc:ID></cac:SellersItemIdentification></cac:Item><cac:Price><cbc:PriceAmount currencyID="%s">%.2f</cbc:PriceAmount></cac:Price></cac:%s>`,
			lineTag, i+1, qtyTag, esc(d.UnidadMedida), d.Cantidad, qtyTag, esc(currency), d.ValorVenta, esc(currency), d.PrecioUnitario, esc(currency), d.IGV, esc(currency), d.ValorVenta, esc(currency), d.IGV, d.PorcentajeIGV, esc(d.TipAfeIGV), esc(d.Descripcion), esc(d.CodProducto), esc(currency), d.ValorUnitario, lineTag)
	}
	fmt.Fprintf(&buf, `</%s>`, rootTag)
	return buf.Bytes(), nil
}
