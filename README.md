# sunat-go

Librería moderna en **Go** (cero dependencias externas) para integrarse con las APIs oficiales de **SUNAT (Perú)**, inspirada en la arquitectura de [Greenter](https://greenter.dev/).

## Módulos incluidos

- `auth`: Cliente OAuth2 (`api-seguridad.sunat.gob.pe`) con caché en memoria thread-safe y renovación automática por `scope` (`ScopeSIRE`, `ScopeCPE`).
- `sire`: Cliente REST para **SIRE** (RVIE / RCE — consulta de periodos y registros).
- `ubl`: Modelos y serializador **UBL 2.1** para Facturas (`01`), Boletas (`03`) y detalle de ítems.
- `soap`: Empaquetador ZIP en memoria y cliente **SOAP SEE** (`EndpointBeta` y `EndpointProd`) con WS-Security `UsernameToken`.

## Uso rápido (OAuth2 + SIRE)

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Jauya/sunat-go/auth"
	"github.com/Jauya/sunat-go/sire"
)

func main() {
	tp := auth.NewTokenProvider(auth.Credentials{
		RUC:          "20000000001",
		SolUser:      "MODDATOS",
		SolPass:      "moddatos",
		ClientID:     "tu-client-id",
		ClientSecret: "tu-client-secret",
	})

	client := sire.NewClient(tp)
	data, err := client.GetPeriods(context.Background(), "rvie", "140000")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## Roadmap

- [x] Autenticación OAuth2 con caché por `scope`
- [x] Cliente SIRE (`GetPeriods`)
- [x] Generador XML UBL 2.1 (`Invoice` con `PaymentTerms`)
- [x] Firmador XMLDSig RSA-SHA256 en Go puro (`xmldsig`)
- [x] Cliente SOAP `sendBill` + empaquetado `.zip` + parser de CDR (`soap`)
- [x] Demo end-to-end contra SUNAT Beta (`go run ./cmd/beta-demo`)
- [ ] Soporte de archivos `.p12` / `.pfx` directo (PKCS#12)
- [ ] Cliente GRE (Guías de Remisión Electrónica REST)
- [ ] Notas de Crédito (`07`) y Débito (`08`)
