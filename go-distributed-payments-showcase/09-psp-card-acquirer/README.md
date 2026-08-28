# Case 09 — PSP Adquirente de Tarjetas: ISO 8583 + Routing por BIN

## El sistema

Un PSP (Payment Service Provider) que actúa como adquirente: recibe la solicitud de autorización del comercio, la traduce a ISO 8583 y la enruta a la red de tarjetas correcta según el BIN de la tarjeta.

```
POST /authorize
      │
   idempotency check
      │
   NetworkRouter (BIN → brand)
      │
      ├── 4xxxxxx  → Visa NET     → ISO 8583 0100/0110 (TCP/IP)
      ├── 51-55xxx → Mastercard   → ISO 8583 0100/0110 + DE48
      └── 34/37xx  → AmEx         → REST JSON (closed-loop, no ISO 8583)
```

## Decisiones de diseño

### 1. El servicio habla `CardTransaction`, los adaptadores hablan ISO 8583

El cambio más importante respecto a una implementación naïve: el servicio **nunca** construye mensajes ISO 8583. Eso es responsabilidad exclusiva de cada adaptador.

```go
// ❌ Anti-patrón: el servicio conoce ISO 8583
func (s *Service) Authorize(ctx, txn) {
    msg := iso8583.Message{MTI: "0100", DE4: formatAmount(txn.AmountCents), ...}
    resp := s.visanet.Send(ctx, msg)
}

// ✅ Patrón correcto: el servicio habla dominio
func (s *Service) Authorize(ctx, txn) {
    result, err := s.network.Authorize(ctx, txn) // txn es CardTransaction
}

// internal/visanet/gateway.go — la traducción vive aquí
func (g *Gateway) Authorize(ctx, txn authorization.CardTransaction) (authorization.CardTransaction, error) {
    req := iso8583.Message{MTI: iso8583.AuthorizationRequest, DE4: iso8583.FormatAmount(txn.AmountCents), ...}
    // ... send, decode response, map DE39 → domain error
}
```

Si Visa cambia su API, solo cambia `internal/visanet/gateway.go`. El servicio no se toca.

### 2. AmEx es un caso especial: red cerrada, REST propio

Visa y Mastercard usan ISO 8583 binario sobre TCP/IP. AmEx opera su propia red y expone una API REST con campo names en inglés y sin campos DE-numerados.

```go
// internal/amex/gateway.go — absorbe la diferencia totalmente
type authorizeRequest struct {
    TransactionID string  `json:"transaction_id"` // ≠ DE37 RRN
    Amount        amexAmt `json:"amount"`
    Status        string  // "APPROVED" | "DECLINED" ≠ DE39 "00" | "05"
}
```

El `NetworkRouter` no sabe que AmEx usa REST. Solo sabe que recibe `CardTransaction` y devuelve `CardTransaction`. La diferencia de protocolo está encapsulada.

### 3. Routing por BIN, circuit breaker por red

El BIN determina la marca, la marca determina la red:

```go
// internal/authorization.go
func BrandFromBIN(bin string) CardBrand {
    switch {
    case bin[0] == '4':           return BrandVisa       // 4xxxxxx
    case bin[:2] >= "51" && ...:  return BrandMastercard // 51-55 + 2221-2720
    case bin[:2] == "34" || ...:  return BrandAmex       // 34, 37
    }
}
```

Cada red tiene su propio circuit breaker. Si Mastercard Banknet cae, las tarjetas Visa y AmEx siguen operando.

### 4. ISO 8583 como package de protocolo, no de dominio

```
internal/
├── authorization.go  ← CardTransaction, errores, Service, Router, CircuitBreaker
├── handler.go        ← adaptador HTTP (mismo package)
├── iso8583/          ← tipos de protocolo (MTI, DE fields, ResponseCode)
│   └── message.go
├── visanet/          ← adaptador Visa NET (ISO 8583)
├── mastercard/       ← adaptador Mastercard Banknet (ISO 8583 + DE48)
└── amex/             ← adaptador AmEx (REST propio)
```

`iso8583/` es un sub-package porque es una librería de protocolo reutilizable por los tres adaptadores. No es "dominio" — es el wire format de la industria.

### 5. Idempotencia antes de tocar la red

Cada autorización pasa primero por el filtro de idempotencia. Si el comercio reintenta (timeout de red), el mismo resultado se devuelve sin enviar un segundo 0100 a la red:

```
Intento 1: acquire("txn-001") → hit Visa → 0100/0110 → APPROVED → complete("txn-001")
Intento 2: acquire("txn-001") → already completed → return cached APPROVED (no hit)
```

Sin esto, dos 0100 con el mismo monto pueden resultar en dos cobros.

## ISO 8583 — Campos clave

| Campo | Nombre | Ejemplo |
|-------|--------|---------|
| MTI `0100` | Authorization Request | acquirer → issuer |
| MTI `0110` | Authorization Response | issuer → acquirer |
| DE4 | Amount | `000000005000` = $50.00 |
| DE11 | STAN | `000001` — 6 dígitos, único por terminal/día |
| DE37 | RRN | `RRN000000001` — 12 chars, usado para reversals |
| DE38 | Auth Code | `AUTH01` — 6 chars, solo en aprobaciones |
| DE39 | Response Code | `00`=approved · `51`=insufficient · `91`=issuer down |
| DE49 | Currency | `840`=USD · `032`=ARS · `986`=BRL · `978`=EUR |

## Ejecutar

```bash
cd 09-psp-card-acquirer && go test ./... -count=1 -v
```
