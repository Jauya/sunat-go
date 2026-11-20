package ubl

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// BuildInvoiceXML serializes an Invoice into SUNAT-compliant UBL 2.1 XML.
func BuildInvoiceXML(inv Invoice) ([]byte, error) {
	if inv.Serie == "" || inv.Correlativo == "" {
		return nil, fmt.Errorf("serie and correlativo are required")
	}
	currency := inv.TipoMoneda
	if currency == "" {
		currency = "PEN"
	}
	country := inv.Company.Address.Country
	if country == "" {
		country = "PE"
	}
	opType := inv.TipoOperacion
	if opType == "" {
		opType = "0101"
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, `<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2" xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2">`)
	buf.WriteString(`<ext:UBLExtensions><ext:UBLExtension><ext:ExtensionContent/></ext:UBLExtension></ext:UBLExtensions>`)
	buf.WriteString(`<cbc:UBLVersionID>2.1</cbc:UBLVersionID>`)
	buf.WriteString(`<cbc:CustomizationID>2.0</cbc:CustomizationID>`)
	fmt.Fprintf(&buf, `<cbc:ID>%s-%s</cbc:ID>`, esc(inv.Serie), esc(inv.Correlativo))
	fmt.Fprintf(&buf, `<cbc:IssueDate>%s</cbc:IssueDate>`, esc(inv.FechaEmision))
	fmt.Fprintf(&buf, `<cbc:InvoiceTypeCode listID="%s">%s</cbc:InvoiceTypeCode>`, esc(opType), esc(inv.TipoDoc))
	for _, leg := range inv.Legends {
		fmt.Fprintf(&buf, `<cbc:Note languageLocaleID="%s">%s</cbc:Note>`, esc(leg.Code), esc(leg.Value))
	}
	fmt.Fprintf(&buf, `<cbc:DocumentCurrencyCode>%s</cbc:DocumentCurrencyCode>`, esc(currency))

	// Supplier
	fmt.Fprintf(&buf, `<cac:AccountingSupplierParty><cac:Party><cac:PartyIdentification><cbc:ID schemeID="6">%s</cbc:ID></cac:PartyIdentification><cac:PartyLegalEntity><cbc:RegistrationName>%s</cbc:RegistrationName><cac:RegistrationAddress><cbc:ID>%s</cbc:ID><cbc:AddressTypeCode>0000</cbc:AddressTypeCode><cac:AddressLine><cbc:Line>%s</cbc:Line></cac:AddressLine><cac:Country><cbc:IdentificationCode>%s</cbc:IdentificationCode></cac:Country></cac:RegistrationAddress></cac:PartyLegalEntity></cac:Party></cac:AccountingSupplierParty>`,
		esc(inv.Company.RUC), esc(inv.Company.RazonSocial), esc(inv.Company.Address.Ubigeo), esc(inv.Company.Address.Direccion), esc(country))

	// Customer
	fmt.Fprintf(&buf, `<cac:AccountingCustomerParty><cac:Party><cac:PartyIdentification><cbc:ID schemeID="%s">%s</cbc:ID></cac:PartyIdentification><cac:PartyLegalEntity><cbc:RegistrationName>%s</cbc:RegistrationName></cac:PartyLegalEntity></cac:Party></cac:AccountingCustomerParty>`,
		esc(inv.Client.TipoDoc), esc(inv.Client.NumDoc), esc(inv.Client.RazonSocial))

	// PaymentTerms (required for Factura 01 in UBL 2.1)
	if inv.TipoDoc == "01" {
		buf.WriteString(`<cac:PaymentTerms><cbc:ID>FormaPago</cbc:ID><cbc:PaymentMeansID>Contado</cbc:PaymentMeansID></cac:PaymentTerms>`)
	}

	// TaxTotal
	fmt.Fprintf(&buf, `<cac:TaxTotal><cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount><cac:TaxSubtotal><cbc:TaxableAmount currencyID="%s">%.2f</cbc:TaxableAmount><cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount><cac:TaxCategory><cac:TaxScheme><cbc:ID>1000</cbc:ID><cbc:Name>IGV</cbc:Name><cbc:TaxTypeCode>VAT</cbc:TaxTypeCode></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal>`,
		esc(currency), inv.TotalImpuestos, esc(currency), inv.MtoOperGravada, esc(currency), inv.MtoIGV)

	// LegalMonetaryTotal
	fmt.Fprintf(&buf, `<cac:LegalMonetaryTotal><cbc:LineExtensionAmount currencyID="%s">%.2f</cbc:LineExtensionAmount><cbc:TaxInclusiveAmount currencyID="%s">%.2f</cbc:TaxInclusiveAmount><cbc:PayableAmount currencyID="%s">%.2f</cbc:PayableAmount></cac:LegalMonetaryTotal>`,
		esc(currency), inv.ValorVenta, esc(currency), inv.MtoImpVenta, esc(currency), inv.MtoImpVenta)

	// Lines
	for i, d := range inv.Details {
		fmt.Fprintf(&buf, `<cac:InvoiceLine><cbc:ID>%d</cbc:ID><cbc:InvoicedQuantity unitCode="%s">%.2f</cbc:InvoicedQuantity><cbc:LineExtensionAmount currencyID="%s">%.2f</cbc:LineExtensionAmount><cac:PricingReference><cac:AlternativeConditionPrice><cbc:PriceAmount currencyID="%s">%.2f</cbc:PriceAmount><cbc:PriceTypeCode>01</cbc:PriceTypeCode></cac:AlternativeConditionPrice></cac:PricingReference><cac:TaxTotal><cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount><cac:TaxSubtotal><cbc:TaxableAmount currencyID="%s">%.2f</cbc:TaxableAmount><cbc:TaxAmount currencyID="%s">%.2f</cbc:TaxAmount><cac:TaxCategory><cbc:Percent>%.2f</cbc:Percent><cbc:TaxExemptionReasonCode>%s</cbc:TaxExemptionReasonCode><cac:TaxScheme><cbc:ID>1000</cbc:ID><cbc:Name>IGV</cbc:Name><cbc:TaxTypeCode>VAT</cbc:TaxTypeCode></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal></cac:TaxTotal><cac:Item><cbc:Description>%s</cbc:Description><cac:SellersItemIdentification><cbc:ID>%s</cbc:ID></cac:SellersItemIdentification></cac:Item><cac:Price><cbc:PriceAmount currencyID="%s">%.2f</cbc:PriceAmount></cac:Price></cac:InvoiceLine>`,
			i+1, esc(d.UnidadMedida), d.Cantidad, esc(currency), d.ValorVenta, esc(currency), d.PrecioUnitario, esc(currency), d.IGV, esc(currency), d.ValorVenta, esc(currency), d.IGV, d.PorcentajeIGV, esc(d.TipAfeIGV), esc(d.Descripcion), esc(d.CodProducto), esc(currency), d.ValorUnitario)
	}

	buf.WriteString(`</Invoice>`)
	return buf.Bytes(), nil
}

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
