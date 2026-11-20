package ubl

// Company represents the emitter (Contribuyente) in UBL 2.1.
type Company struct {
	RUC             string
	RazonSocial     string
	NombreComercial string
	Address         Address
}

// Address represents a SUNAT fiscal address (Ubigeo + Dirección).
type Address struct {
	Ubigeo    string
	Direccion string
	District  string
	Province  string
	Region    string
	Country   string // Default: "PE"
}

// Client represents the customer (Adquiriente / Usuario).
type Client struct {
	TipoDoc     string // "6" = RUC, "1" = DNI, "-" = Varios
	NumDoc      string
	RazonSocial string
}

// SaleDetail represents a single line item on an invoice or credit/debit note.
type SaleDetail struct {
	UnidadMedida     string  // e.g. "NIU", "ZZ"
	Cantidad         float64
	CodProducto      string
	Descripcion      string
	ValorUnitario    float64 // Without IGV
	PrecioUnitario   float64 // With IGV
	ValorVenta       float64 // Cantidad * ValorUnitario
	IGV              float64
	PorcentajeIGV    float64 // e.g. 18.0
	TipAfeIGV        string  // e.g. "10" (Gravado - Operación Onerosa)
}

// Legend represents UBL Note legends (e.g. code "1000" = monto en letras).
type Legend struct {
	Code  string
	Value string
}

// Invoice models a Factura (01) or Boleta (03) inspired by Greenter.
type Invoice struct {
	TipoDoc        string // "01" Factura, "03" Boleta
	Serie          string // e.g. "F001", "B001"
	Correlativo    string // e.g. "1"
	FechaEmision   string // YYYY-MM-DD
	TipoMoneda     string // "PEN", "USD"
	TipoOperacion  string // "0101" Venta interna
	Company        Company
	Client         Client
	MtoOperGravada float64
	MtoIGV         float64
	TotalImpuestos float64
	ValorVenta     float64
	MtoImpVenta    float64
	Details        []SaleDetail
	Legends        []Legend
}
